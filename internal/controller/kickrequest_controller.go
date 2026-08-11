package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/executor"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	argocdprovider "github.com/corewire/kick/internal/gitops/argocd"
	"github.com/corewire/kick/internal/notify"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	"github.com/corewire/kick/internal/schedule"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	statusConditionProgressing = "Progressing"
	defaultRequeueInterval     = 30 * time.Second
	defaultRequestRetention    = 24 * time.Hour
)

type GateResolver interface {
	ResolveOwnerAndGate(ctx context.Context, workload client.Object, providerName string, now time.Time) (kickv1alpha1.GitOpsOwnerStatus, gitops.GateDecision, error)
}

type RegistryGateResolver struct {
	Registry *gitops.Registry
}

func (r *RegistryGateResolver) ResolveOwnerAndGate(ctx context.Context, workload client.Object, providerName string, now time.Time) (kickv1alpha1.GitOpsOwnerStatus, gitops.GateDecision, error) {
	if r == nil || r.Registry == nil {
		return kickv1alpha1.GitOpsOwnerStatus{}, gitops.GateDecision{Reason: gitops.GateConfigurationError, Message: "provider registry is not configured"}, nil
	}

	provider, detectDecision := r.selectProvider(workload, providerName)
	if provider == nil {
		return kickv1alpha1.GitOpsOwnerStatus{}, detectDecision, nil
	}

	owner, err := provider.ResolveOwner(ctx, workload)
	if err != nil {
		reason := gitops.GateOwnerUnknown
		var resolutionErr argocdprovider.ResolutionError
		if errors.As(err, &resolutionErr) {
			reason = resolutionErr.Reason
		}
		return kickv1alpha1.GitOpsOwnerStatus{}, gitops.GateDecision{
			Allowed:     false,
			Reconciled:  false,
			Reconciling: false,
			Reason:      reason,
			Message:     err.Error(),
		}, nil
	}

	decision, err := provider.EvaluateGate(ctx, owner, now)
	if err != nil {
		return ownerToStatus(owner), gitops.GateDecision{
			Allowed:     false,
			Reconciled:  false,
			Reconciling: false,
			Reason:      gitops.GateProviderUnavailable,
			Message:     err.Error(),
		}, nil
	}

	return ownerToStatus(owner), decision, nil
}

// selectProvider honors an explicitly configured provider and falls back to
// detection when the policy asks for Auto. Some providers (Kargo) cannot be
// detected from workload metadata alone and must be named.
func (r *RegistryGateResolver) selectProvider(workload client.Object, providerName string) (gitops.Provider, gitops.GateDecision) {
	if providerName == "" {
		return r.Registry.DetectProvider(workload)
	}
	provider, ok := r.Registry.ProviderByName(providerName)
	if !ok {
		return nil, gitops.GateDecision{Reason: gitops.GateProviderUnavailable, Message: "configured provider is not enabled: " + providerName}
	}
	return provider, gitops.GateDecision{}
}

// gateProviderName maps the policy provider enum onto a registered provider
// name. Auto returns an empty string, which selects detection.
func gateProviderName(pol *kickv1alpha1.KickPolicy) string {
	if pol == nil {
		return ""
	}
	switch pol.Spec.GitOps.Provider {
	case kickv1alpha1.KickPolicyProviderArgoCD:
		return "argocd"
	case kickv1alpha1.KickPolicyProviderFlux:
		return "flux"
	case kickv1alpha1.KickPolicyProviderKargo:
		return "kargo"
	default:
		return ""
	}
}

type FreshnessEvaluator interface {
	Evaluate(ctx context.Context, workload client.Object, currentDependencies []dependency.DependencyRef, latestRelevantChanges map[dependency.DependencyRef]time.Time) (freshness.FreshnessDecision, error)
}

type RestartExecutor interface {
	Execute(ctx context.Context, requestKey types.NamespacedName, targetRef kickv1alpha1.ObjectReference, targetKey types.NamespacedName) (executor.Result, error)
}

type KickRequestReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Recorder           record.EventRecorder
	PolicyMatcher      policy.WorkloadMatcher
	GateResolver       GateResolver
	ObservationStore   observation.Store
	FreshnessEvaluator FreshnessEvaluator
	RestartExecutor    RestartExecutor
	// Notifier receives phase transitions. Delivery is best-effort and never
	// influences reconciliation.
	Notifier         notify.Dispatcher
	Clock            func() time.Time
	RequeueInterval  time.Duration
	RequestRetention time.Duration
}

func (r *KickRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !r.dependenciesConfigured() {
		observeControllerError("kickrequest", "MissingDependency")
		return ctrl.Result{}, fmt.Errorf("kickrequest reconciler dependencies are not fully configured")
	}

	var request kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &request); err != nil {
		if !apierrors.IsNotFound(err) {
			observeControllerError("kickrequest", "GetRequest")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if isTerminalPhase(request.Status.Phase) {
		return r.reconcileTerminalRequest(ctx, &request)
	}

	if !supportedTargetRef(request.Spec.TargetRef) {
		return r.failInvalidTarget(ctx, req, &request)
	}

	targetKey := types.NamespacedName{Namespace: req.Namespace, Name: request.Spec.TargetRef.Name}
	workload, result, done, err := r.loadWorkloadOrFail(ctx, req, &request, targetKey)
	if done || err != nil {
		return result, err
	}

	matchedPolicy, result, done, err := r.resolveMatchedPolicy(ctx, req, &request, workload)
	if done || err != nil {
		return result, err
	}

	now := r.now().UTC()

	result, done, err = r.evaluateNativeWindows(ctx, req, &request, workload, targetKey, matchedPolicy, now)
	if done || err != nil {
		return result, err
	}

	// No GitOps provider configured: KICK gates on its own. With no native
	// windows above, a stale dependency restarts immediately.
	if !policyGitOpsGated(matchedPolicy) {
		return r.evaluateFreshnessAndExecute(ctx, req, &request, workload, targetKey, matchedPolicy,
			kickv1alpha1.GitOpsOwnerStatus{},
			gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed, Message: "no GitOps provider configured"})
	}

	ownerStatus, gateDecision, err := r.GateResolver.ResolveOwnerAndGate(ctx, workload, gateProviderName(matchedPolicy), now)
	if err != nil {
		observeControllerError("kickrequest", "ResolveOwnerAndGate")
		return ctrl.Result{}, err
	}

	if !gitops.MayExecute(gateDecision) {
		return r.handleClosedGate(ctx, req, &request, ownerStatus, gateDecision, now)
	}

	return r.evaluateFreshnessAndExecute(ctx, req, &request, workload, targetKey, matchedPolicy, ownerStatus, gateDecision)
}

// dependenciesConfigured reports whether all injected collaborators are set.
func (r *KickRequestReconciler) dependenciesConfigured() bool {
	return r.GateResolver != nil && r.ObservationStore != nil && r.FreshnessEvaluator != nil && r.RestartExecutor != nil
}

// failInvalidTarget terminates a request whose targetRef is unsupported or empty.
func (r *KickRequestReconciler) failInvalidTarget(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest) (ctrl.Result, error) {
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "InvalidTarget", "unsupported or empty targetRef")
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return ctrl.Result{}, err
	}
	r.recordTransition(request, kickv1alpha1.KickRequestPhaseFailed, "InvalidTarget", "unsupported or empty targetRef", "")
	return ctrl.Result{}, nil
}

// handleClosedGate records a blocked gate decision and schedules the next check.
func (r *KickRequestReconciler) handleClosedGate(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, ownerStatus kickv1alpha1.GitOpsOwnerStatus, gateDecision gitops.GateDecision, now time.Time) (ctrl.Result, error) {
	phase := phaseForClosedGate(gateDecision.Reason)
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		status.Owner = ownerStatus
		status.Gate = gateToStatus(gateDecision)
		setPhase(status, phase, string(gateDecision.Reason), gateDecision.Message)
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return ctrl.Result{}, err
	}
	r.recordTransition(request, phase, string(gateDecision.Reason), gateDecision.Message, ownerStatus.Provider)
	return requeueForGate(gateDecision, now, r.requeueInterval()), nil
}

// loadWorkloadOrFail fetches the target workload. The boolean return is true
// when the request was terminated (target missing) and the caller should return
// the accompanying result.
func (r *KickRequestReconciler) loadWorkloadOrFail(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, targetKey types.NamespacedName) (client.Object, ctrl.Result, bool, error) {
	workload, err := loadTargetWorkload(ctx, r.Client, request.Spec.TargetRef, targetKey)
	if err == nil {
		return workload, ctrl.Result{}, false, nil
	}
	if !apierrors.IsNotFound(err) {
		observeControllerError("kickrequest", "GetWorkload")
		return nil, ctrl.Result{}, true, err
	}
	if statusErr := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "TargetNotFound", "target workload no longer exists")
	}); statusErr != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return nil, ctrl.Result{}, true, statusErr
	}
	r.recordTransition(request, kickv1alpha1.KickRequestPhaseFailed, "TargetNotFound", "target workload no longer exists", "")
	return nil, ctrl.Result{}, true, nil
}

// resolveMatchedPolicy resolves the KickPolicy managing the workload. The
// boolean return is true when the request was terminated (unmanaged scope) and
// the caller should return the accompanying result.
func (r *KickRequestReconciler) resolveMatchedPolicy(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, workload client.Object) (*kickv1alpha1.KickPolicy, ctrl.Result, bool, error) {
	if r.PolicyMatcher == nil {
		return nil, ctrl.Result{}, false, nil
	}
	match, err := r.PolicyMatcher.MatchWorkload(ctx, workload.GetNamespace(), workloadLabels(workload))
	if err != nil {
		observeControllerError("kickrequest", "PolicyMatch")
		return nil, ctrl.Result{}, true, err
	}
	if match.Managed {
		return match.Policy, ctrl.Result{}, false, nil
	}

	reason := match.Reason
	if reason == policy.ReasonNoMatchingPolicy || reason == policy.ReasonPolicyDeleted {
		reason = policy.ReasonPolicyDeleted
	}
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		setPhase(status, kickv1alpha1.KickRequestPhaseFailed, reason, "kickrequest canceled due to policy scope")
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return nil, ctrl.Result{}, true, err
	}
	r.recordTransition(request, kickv1alpha1.KickRequestPhaseFailed, reason, "kickrequest canceled due to policy scope", "")
	return nil, ctrl.Result{}, true, nil
}

// evaluateNativeWindows gates the kick on KICK-native windows (no GitOps owner).
// The boolean return is true when a native window produced a terminal decision
// and the caller should return the accompanying result.
func (r *KickRequestReconciler) evaluateNativeWindows(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, workload client.Object, targetKey types.NamespacedName, matchedPolicy *kickv1alpha1.KickPolicy, now time.Time) (ctrl.Result, bool, error) {
	nativeWindows := policyNativeWindows(matchedPolicy)
	if len(nativeWindows) == 0 {
		return ctrl.Result{}, false, nil
	}

	decision, evalErr := schedule.Evaluate(now, nativeWindows)
	if evalErr != nil {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateConfigurationError), evalErr.Error())
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, true, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateConfigurationError), evalErr.Error(), "")
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, true, nil
	}

	if !decision.Allowed {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Gate = kickv1alpha1.GateStatus{Reason: string(gitops.GateOutsideSchedule), Message: "blocked by KickPolicy window"}
			setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateOutsideSchedule), "blocked by KickPolicy window")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, true, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateOutsideSchedule), "blocked by KickPolicy window", "")
		requeue := r.requeueInterval()
		if decision.RequeueAt != nil {
			if d := decision.RequeueAt.Sub(now); d > 0 {
				requeue = d
			}
		}
		return ctrl.Result{RequeueAfter: requeue}, true, nil
	}

	res, err := r.evaluateFreshnessAndExecute(ctx, req, request, workload, targetKey, matchedPolicy, kickv1alpha1.GitOpsOwnerStatus{}, gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed, Message: "allowed by KickPolicy window"})
	return res, true, err
}

func (r *KickRequestReconciler) evaluateFreshnessAndExecute(
	ctx context.Context,
	req ctrl.Request,
	request *kickv1alpha1.KickRequest,
	workload client.Object,
	targetKey types.NamespacedName,
	matchedPolicy *kickv1alpha1.KickPolicy,
	ownerStatus kickv1alpha1.GitOpsOwnerStatus,
	gateDecision gitops.GateDecision,
) (ctrl.Result, error) {
	deps := dependency.ExtractDependenciesForObject(workload)
	deps, err := r.scopeDependencies(ctx, matchedPolicy, deps)
	if err != nil {
		observeControllerError("kickrequest", "ScopeDependencies")
		return ctrl.Result{}, err
	}
	latestChanges, err := r.latestRelevantChanges(ctx, deps)
	if err != nil {
		observeControllerError("kickrequest", "LatestRelevantChanges")
		return ctrl.Result{}, err
	}

	freshnessDecision, err := r.FreshnessEvaluator.Evaluate(ctx, workload, deps, latestChanges)
	if err != nil {
		observeControllerError("kickrequest", "FreshnessEvaluate")
		return ctrl.Result{}, err
	}

	// Once KICK has issued the restart, the executor owns the request. The
	// workload is fresh precisely because of that restart, so re-running the
	// freshness gate here would terminate the request as NoLongerRequired and
	// discard the outcome of the rollout KICK started.
	if !restartIssued(request) {
		if res, done, err := r.handleFreshnessGate(ctx, req, request, ownerStatus, gateDecision, freshnessDecision); done || err != nil {
			return res, err
		}

		if policyDryRun(matchedPolicy) {
			return r.recordDryRun(ctx, req, request, ownerStatus, gateDecision, freshnessDecision)
		}
	}

	// Transition to Executing once. The executor owns CurrentRollout.StartedAt
	// (the time KICK issues the restart); pre-seeding it from the existing
	// ReplicaSet would make the executor treat the restart as already issued.
	if request.Status.Phase != kickv1alpha1.KickRequestPhaseExecuting {
		if err := r.markExecuting(ctx, req, request, ownerStatus, gateDecision, freshnessDecision); err != nil {
			return ctrl.Result{}, err
		}
	}

	result, err := r.RestartExecutor.Execute(ctx, req.NamespacedName, request.Spec.TargetRef, targetKey)
	if err != nil {
		observeControllerError("kickrequest", "ExecuteRestart")
		return ctrl.Result{}, err
	}

	return r.finalizeRestart(ctx, req, request, ownerStatus, result)
}

// restartIssued reports whether KICK already patched the workload for this
// request and is now only waiting for the rollout it started to settle.
func restartIssued(request *kickv1alpha1.KickRequest) bool {
	return request.Status.Phase == kickv1alpha1.KickRequestPhaseExecuting &&
		request.Status.CurrentRollout.StartedAt != nil
}

// handleFreshnessGate records the terminal outcomes that require no restart
// (rollout still in progress, or already fresh). The boolean return is true
// when it produced a result the caller should return.
func (r *KickRequestReconciler) handleFreshnessGate(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, ownerStatus kickv1alpha1.GitOpsOwnerStatus, gateDecision gitops.GateDecision, freshnessDecision freshness.FreshnessDecision) (ctrl.Result, bool, error) {
	if freshnessDecision.BlockingReason != "" && !freshnessDecision.RestartRequired {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.CurrentRollout = rolloutToStatus(freshnessDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForRollout, freshnessDecision.BlockingReason, "workload rollout is still in progress")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, true, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseWaitingForRollout, freshnessDecision.BlockingReason, "workload rollout is still in progress", ownerStatus.Provider)
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, true, nil
	}

	if !freshnessDecision.RestartRequired {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.CurrentRollout = rolloutToStatus(freshnessDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseNoLongerRequired, "Fresh", "live workload is already fresh")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, true, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseNoLongerRequired, "Fresh", "live workload is already fresh", ownerStatus.Provider)
		return ctrl.Result{}, true, nil
	}

	return ctrl.Result{}, false, nil
}

// recordDryRun terminates a request that would have restarted the workload, but
// whose policy runs in dry-run mode.
func (r *KickRequestReconciler) recordDryRun(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, ownerStatus kickv1alpha1.GitOpsOwnerStatus, gateDecision gitops.GateDecision, freshnessDecision freshness.FreshnessDecision) (ctrl.Result, error) {
	const message = "restart required, not performed because the policy is in dry-run mode"
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		status.Owner = ownerStatus
		status.Gate = gateToStatus(gateDecision)
		status.CurrentRollout = rolloutToStatus(freshnessDecision)
		status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
		setPhase(status, kickv1alpha1.KickRequestPhaseDryRun, "DryRun", message)
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return ctrl.Result{}, err
	}
	r.recordTransition(request, kickv1alpha1.KickRequestPhaseDryRun, "DryRun", message, ownerStatus.Provider)
	return ctrl.Result{}, nil
}

// markExecuting transitions the request to Executing before issuing the restart.
func (r *KickRequestReconciler) markExecuting(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, ownerStatus kickv1alpha1.GitOpsOwnerStatus, gateDecision gitops.GateDecision, freshnessDecision freshness.FreshnessDecision) error {
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		status.Owner = ownerStatus
		status.Gate = gateToStatus(gateDecision)
		status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
		setPhase(status, kickv1alpha1.KickRequestPhaseExecuting, "RestartRequired", "restart required after live freshness check")
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return err
	}
	r.recordTransition(request, kickv1alpha1.KickRequestPhaseExecuting, "RestartRequired", "restart required after live freshness check", ownerStatus.Provider)
	return nil
}

// finalizeRestart re-reads the request after execution and maps the terminal
// rollout phase to metrics, transitions, and the next requeue.
func (r *KickRequestReconciler) finalizeRestart(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, ownerStatus kickv1alpha1.GitOpsOwnerStatus, result executor.Result) (ctrl.Result, error) {
	var updated kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &updated); err != nil {
		if !apierrors.IsNotFound(err) {
			observeControllerError("kickrequest", "GetUpdatedRequest")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if updated.Status.Phase == kickv1alpha1.KickRequestPhaseSucceeded {
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseSucceeded, "Completed", "rollout completed", ownerStatus.Provider)
		observeRestartResult(ownerStatus.Provider, "succeeded")
	}
	if updated.Status.Phase == kickv1alpha1.KickRequestPhaseFailed {
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseFailed, "Failed", "rollout failed", ownerStatus.Provider)
		observeRestartResult(ownerStatus.Provider, "failed")
	}
	if isTerminalPhase(updated.Status.Phase) || result.Complete || result.Failed {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
}

func policyNativeWindows(pol *kickv1alpha1.KickPolicy) []schedule.Window {
	if pol == nil {
		return nil
	}
	specWindows := pol.Spec.Schedule.Windows
	if len(specWindows) == 0 {
		return nil
	}
	out := make([]schedule.Window, 0, len(specWindows))
	for _, w := range specWindows {
		out = append(out, schedule.Window{
			Allow:    w.Type == kickv1alpha1.KickPolicyWindowTypeAllow,
			Schedule: w.Cron,
			Duration: w.Duration,
			TimeZone: w.TimeZone,
		})
	}
	return out
}

// policyDryRun reports whether the matched policy evaluates without patching.
func policyDryRun(pol *kickv1alpha1.KickPolicy) bool {
	return pol != nil && pol.Spec.DryRun
}

// policyGitOpsGated reports whether restart decisions defer to a GitOps
// provider. An unset or "None" provider means KICK gates on its own.
func policyGitOpsGated(pol *kickv1alpha1.KickPolicy) bool {
	if pol == nil {
		return true
	}
	switch pol.Spec.GitOps.Provider {
	case "", kickv1alpha1.KickPolicyProviderNone:
		return false
	default:
		return true
	}
}

// dependencySelectorMatches reports whether a changed dependency (identified by
// its labels) is in the policy's trigger scope. A nil policy or empty selector
// matches every dependency.
func dependencySelectorMatches(pol *kickv1alpha1.KickPolicy, depLabels map[string]string) (bool, error) {
	if pol == nil || pol.Spec.Discovery.DependencySelector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(pol.Spec.Discovery.DependencySelector)
	if err != nil {
		return false, err
	}
	if selector.Empty() {
		return true, nil
	}
	return selector.Matches(labels.Set(depLabels)), nil
}

// scopeDependencies filters a workload's dependencies to those matching the
// policy's dependencySelector, so both the trigger and freshness evaluation
// ignore out-of-scope Secrets/ConfigMaps. A nil/empty selector returns deps
// unchanged without extra reads.
func (r *KickRequestReconciler) scopeDependencies(ctx context.Context, pol *kickv1alpha1.KickPolicy, deps []dependency.DependencyRef) ([]dependency.DependencyRef, error) {
	if pol == nil || pol.Spec.Discovery.DependencySelector == nil {
		return deps, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(pol.Spec.Discovery.DependencySelector)
	if err != nil {
		return nil, err
	}
	if selector.Empty() {
		return deps, nil
	}
	out := make([]dependency.DependencyRef, 0, len(deps))
	for _, dep := range deps {
		depLabels, found, err := dependencyLabels(ctx, r.Client, dep)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if selector.Matches(labels.Set(depLabels)) {
			out = append(out, dep)
		}
	}
	return out, nil
}

// dependencyLabels fetches the labels of a Secret or ConfigMap dependency.
// found is false when the object no longer exists.
func dependencyLabels(ctx context.Context, c client.Client, dep dependency.DependencyRef) (map[string]string, bool, error) {
	key := client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name}
	switch dep.Kind {
	case dependency.Secret:
		var secret corev1.Secret
		if err := c.Get(ctx, key, &secret); err != nil {
			return nil, false, client.IgnoreNotFound(err)
		}
		return secret.GetLabels(), true, nil
	case dependency.ConfigMap:
		var configMap corev1.ConfigMap
		if err := c.Get(ctx, key, &configMap); err != nil {
			return nil, false, client.IgnoreNotFound(err)
		}
		return configMap.GetLabels(), true, nil
	case dependency.SecretProviderClass:
		obj := dependency.NewSecretProviderClassObject()
		if err := c.Get(ctx, key, obj); err != nil {
			return nil, false, client.IgnoreNotFound(err)
		}
		return obj.GetLabels(), true, nil
	default:
		return nil, false, nil
	}
}

func (r *KickRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	registerControllerMetrics()
	if r.Recorder == nil {
		//nolint:staticcheck // controller-runtime still returns the client-go EventRecorder interface from this method.
		r.Recorder = mgr.GetEventRecorderFor("kickrequest-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kickv1alpha1.KickRequest{}).
		Watches(&kickv1alpha1.KickPolicy{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			var list kickv1alpha1.KickRequestList
			if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]reconcile.Request, 0, len(list.Items))
			for _, item := range list.Items {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: item.Namespace, Name: item.Name}})
			}
			return requests
		})).
		Complete(r)
}

func (r *KickRequestReconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}

func (r *KickRequestReconciler) requeueInterval() time.Duration {
	if r.RequeueInterval > 0 {
		return r.RequeueInterval
	}
	return defaultRequeueInterval
}

func (r *KickRequestReconciler) requestRetention() time.Duration {
	if r.RequestRetention > 0 {
		return r.RequestRetention
	}
	return defaultRequestRetention
}

func (r *KickRequestReconciler) reconcileTerminalRequest(ctx context.Context, request *kickv1alpha1.KickRequest) (ctrl.Result, error) {
	completedAt := terminalPhaseTransitionTime(request)
	expiresAt := completedAt.Add(r.requestRetention())
	now := r.now().UTC()

	if !now.Before(expiresAt) {
		if err := r.Delete(ctx, request); err != nil && !apierrors.IsNotFound(err) {
			observeControllerError("kickrequest", "DeleteTerminalRequest")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: expiresAt.Sub(now)}, nil
}

func terminalPhaseTransitionTime(request *kickv1alpha1.KickRequest) time.Time {
	if request != nil {
		for _, cond := range request.Status.Conditions {
			if cond.Type == statusConditionProgressing {
				if cond.LastTransitionTime.IsZero() {
					break
				}
				return cond.LastTransitionTime.UTC()
			}
		}
		if !request.CreationTimestamp.IsZero() {
			return request.CreationTimestamp.UTC()
		}
	}
	return time.Now().UTC()
}

func (r *KickRequestReconciler) updateRequestStatus(ctx context.Context, key types.NamespacedName, mutate func(*kickv1alpha1.KickRequestStatus)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var request kickv1alpha1.KickRequest
		if err := r.Get(ctx, key, &request); err != nil {
			return err
		}
		mutate(&request.Status)
		return r.Status().Update(ctx, &request)
	})
}

func (r *KickRequestReconciler) latestRelevantChanges(ctx context.Context, deps []dependency.DependencyRef) (map[dependency.DependencyRef]time.Time, error) {
	out := make(map[dependency.DependencyRef]time.Time, len(deps))
	for _, dep := range deps {
		identity, ok := dependencyToSourceIdentity(dep)
		if !ok {
			continue
		}
		record, found, err := r.ObservationStore.Get(ctx, identity)
		if err != nil {
			return nil, err
		}
		if !found || record.LastRelevantChangeTime.IsZero() {
			continue
		}
		out[dep] = record.LastRelevantChangeTime.UTC()
	}
	return out, nil
}

func dependencyToSourceIdentity(dep dependency.DependencyRef) (observation.SourceIdentity, bool) {
	identity := observation.SourceIdentity{APIVersion: dep.APIVersion, Namespace: dep.Namespace, Name: dep.Name}
	switch dep.Kind {
	case dependency.Secret:
		identity.Kind = observation.SourceKindSecret
	case dependency.ConfigMap:
		identity.Kind = observation.SourceKindConfigMap
	case dependency.SecretProviderClass:
		identity.Kind = observation.SourceKindSecretProviderClass
	default:
		return observation.SourceIdentity{}, false
	}
	return identity, true
}

func supportedTargetRef(ref kickv1alpha1.ObjectReference) bool {
	if ref.Name == "" {
		return false
	}
	if dependency.IsArgoRollout(ref.APIVersion, ref.Kind) {
		return true
	}
	if ref.APIVersion != "apps/v1" {
		return false
	}
	switch ref.Kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		return true
	default:
		return false
	}
}

func loadTargetWorkload(ctx context.Context, c client.Client, ref kickv1alpha1.ObjectReference, key types.NamespacedName) (client.Object, error) {
	if dependency.IsArgoRollout(ref.APIVersion, ref.Kind) {
		obj := dependency.NewArgoRolloutObject()
		if err := c.Get(ctx, key, obj); err != nil {
			return nil, err
		}
		return obj, nil
	}
	switch ref.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := c.Get(ctx, key, &deployment); err != nil {
			return nil, err
		}
		return &deployment, nil
	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		if err := c.Get(ctx, key, &statefulSet); err != nil {
			return nil, err
		}
		return &statefulSet, nil
	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		if err := c.Get(ctx, key, &daemonSet); err != nil {
			return nil, err
		}
		return &daemonSet, nil
	default:
		return nil, apierrors.NewBadRequest("unsupported target kind")
	}
}

func workloadLabels(obj client.Object) map[string]string {
	if obj == nil {
		return nil
	}
	return obj.GetLabels()
}

func phaseForClosedGate(reason gitops.GateReason) kickv1alpha1.KickRequestPhase {
	switch reason {
	case gitops.GateOwnerUnknown, gitops.GateAmbiguousOwner:
		return kickv1alpha1.KickRequestPhaseWaitingForOwner
	case gitops.GateOwnerOutOfSync, gitops.GateOwnerReconciling:
		return kickv1alpha1.KickRequestPhaseWaitingForAppSync
	default:
		return kickv1alpha1.KickRequestPhaseWaitingForGate
	}
}

func gateToStatus(decision gitops.GateDecision) kickv1alpha1.GateStatus {
	status := kickv1alpha1.GateStatus{Reason: string(decision.Reason), Message: decision.Message}
	if decision.RequeueAt != nil {
		t := metav1.NewTime(decision.RequeueAt.UTC())
		status.RequeueAt = &t
	}
	return status
}

func requeueForGate(decision gitops.GateDecision, now time.Time, fallback time.Duration) ctrl.Result {
	if decision.RequeueAt != nil {
		if decision.RequeueAt.After(now) {
			return ctrl.Result{RequeueAfter: decision.RequeueAt.Sub(now)}
		}
		return ctrl.Result{RequeueAfter: fallback}
	}
	if decision.Reason == gitops.GateOwnerOutOfSync || decision.Reason == gitops.GateOwnerReconciling || decision.Reason == gitops.GateOwnerUnknown || decision.Reason == gitops.GateAmbiguousOwner || decision.Reason == gitops.GateProviderUnavailable {
		return ctrl.Result{RequeueAfter: fallback}
	}
	return ctrl.Result{}
}

func rolloutToStatus(decision freshness.FreshnessDecision) kickv1alpha1.RolloutStatus {
	status := kickv1alpha1.RolloutStatus{}
	if !decision.RolloutStarted.IsZero() {
		t := metav1.NewTime(decision.RolloutStarted.UTC())
		status.StartedAt = &t
	}
	return status
}

func setPhase(status *kickv1alpha1.KickRequestStatus, phase kickv1alpha1.KickRequestPhase, reason, message string) {
	status.Phase = phase
	condition := metav1.Condition{
		Type:               statusConditionProgressing,
		ObservedGeneration: 0,
		LastTransitionTime: metav1.Now(),
		Reason:             sanitizeReason(reason),
		Message:            message,
		Status:             metav1.ConditionTrue,
	}
	if phase == kickv1alpha1.KickRequestPhaseSucceeded || phase == kickv1alpha1.KickRequestPhaseNoLongerRequired || phase == kickv1alpha1.KickRequestPhaseFailed || phase == kickv1alpha1.KickRequestPhaseDryRun {
		condition.Status = metav1.ConditionFalse
	}
	apimeta.SetStatusCondition(&status.Conditions, condition)
}

func sanitizeReason(reason string) string {
	if reason == "" {
		return "Unknown"
	}
	return reason
}

func isTerminalPhase(phase kickv1alpha1.KickRequestPhase) bool {
	return phase == kickv1alpha1.KickRequestPhaseSucceeded || phase == kickv1alpha1.KickRequestPhaseNoLongerRequired || phase == kickv1alpha1.KickRequestPhaseFailed || phase == kickv1alpha1.KickRequestPhaseDryRun
}

func ownerToStatus(owner gitops.Owner) kickv1alpha1.GitOpsOwnerStatus {
	return kickv1alpha1.GitOpsOwnerStatus{
		Provider:   owner.Provider,
		APIVersion: owner.APIVersion,
		Kind:       owner.Kind,
		Namespace:  owner.Namespace,
		Name:       owner.Name,
		Project:    owner.Project,
	}
}

func toMetav1TimePtr(v *time.Time) *metav1.Time {
	if v == nil {
		return nil
	}
	t := metav1.NewTime(v.UTC())
	return &t
}

func (r *KickRequestReconciler) recordTransition(request *kickv1alpha1.KickRequest, newPhase kickv1alpha1.KickRequestPhase, reason, message, provider string) {
	if request == nil || request.Status.Phase == newPhase {
		return
	}

	switch newPhase {
	case kickv1alpha1.KickRequestPhaseSucceeded:
		observeRequestResult(provider, "succeeded")
	case kickv1alpha1.KickRequestPhaseNoLongerRequired:
		observeRequestResult(provider, "no_longer_required")
	case kickv1alpha1.KickRequestPhaseFailed:
		observeRequestResult(provider, "failed")
	case kickv1alpha1.KickRequestPhaseDryRun:
		observeRequestResult(provider, "dry_run")
	case kickv1alpha1.KickRequestPhaseExecuting:
		observeRestartResult(provider, "started")
	}

	if r.Recorder != nil {
		eventType, eventReason := phaseToEvent(newPhase, reason)
		if eventReason != "" {
			r.Recorder.Event(request, eventType, eventReason, message)
		}
	}
	if r.Notifier != nil {
		r.Notifier.Notify(notify.Event{
			Namespace:      request.Namespace,
			RequestName:    request.Name,
			Phase:          string(newPhase),
			Reason:         reason,
			Message:        message,
			TargetKind:     request.Spec.TargetRef.Kind,
			TargetName:     request.Spec.TargetRef.Name,
			GitOpsProvider: provider,
			OccurredAt:     r.now().UTC(),
		}.WithWorkloadLabels(request.Labels))
	}
	request.Status.Phase = newPhase
}

func phaseToEvent(phase kickv1alpha1.KickRequestPhase, reason string) (string, string) {
	switch phase {
	case kickv1alpha1.KickRequestPhaseWaitingForGate:
		return corev1.EventTypeNormal, eventWaitingForSchedule
	case kickv1alpha1.KickRequestPhaseWaitingForAppSync:
		return corev1.EventTypeNormal, eventWaitingForGitOpsSync
	case kickv1alpha1.KickRequestPhaseWaitingForRollout:
		return corev1.EventTypeNormal, eventWaitingForRollout
	case kickv1alpha1.KickRequestPhaseWaitingForOwner:
		if reason == string(gitops.GateAmbiguousOwner) {
			return corev1.EventTypeWarning, eventOwnerAmbiguous
		}
		return corev1.EventTypeWarning, eventOwnerUnknown
	case kickv1alpha1.KickRequestPhaseExecuting:
		return corev1.EventTypeNormal, eventKickStarted
	case kickv1alpha1.KickRequestPhaseSucceeded:
		return corev1.EventTypeNormal, eventKickSucceeded
	case kickv1alpha1.KickRequestPhaseNoLongerRequired:
		return corev1.EventTypeNormal, eventKickNoLongerRequired
	case kickv1alpha1.KickRequestPhaseFailed:
		return corev1.EventTypeWarning, eventKickFailed
	case kickv1alpha1.KickRequestPhaseDryRun:
		return corev1.EventTypeNormal, eventKickDryRun
	default:
		return "", ""
	}
}
