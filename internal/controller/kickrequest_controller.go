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
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	"github.com/corewire/kick/internal/schedule"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ResolveOwnerAndGate(context.Context, client.Object, time.Time) (kickv1alpha1.GitOpsOwnerStatus, gitops.GateDecision, error)
}

type RegistryGateResolver struct {
	Registry *gitops.Registry
}

func (r *RegistryGateResolver) ResolveOwnerAndGate(ctx context.Context, workload client.Object, now time.Time) (kickv1alpha1.GitOpsOwnerStatus, gitops.GateDecision, error) {
	if r == nil || r.Registry == nil {
		return kickv1alpha1.GitOpsOwnerStatus{}, gitops.GateDecision{Reason: gitops.GateConfigurationError, Message: "provider registry is not configured"}, nil
	}

	provider, detectDecision := r.Registry.DetectProvider(workload)
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
	Clock              func() time.Time
	RequeueInterval    time.Duration
	RequestRetention   time.Duration
}

func (r *KickRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := otel.Tracer("kick.controller").Start(ctx, "kickrequest.reconcile")
	defer span.End()
	span.SetAttributes(attribute.String("kick.request.namespace", req.Namespace), attribute.String("kick.request.name", req.Name))

	if r.GateResolver == nil || r.ObservationStore == nil || r.FreshnessEvaluator == nil || r.RestartExecutor == nil {
		observeControllerError("kickrequest", "MissingDependency")
		span.RecordError(fmt.Errorf("missing dependency"))
		span.SetStatus(codes.Error, "missing dependency")
		return ctrl.Result{}, fmt.Errorf("kickrequest reconciler dependencies are not fully configured")
	}

	var request kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &request); err != nil {
		if !apierrors.IsNotFound(err) {
			observeControllerError("kickrequest", "GetRequest")
			span.RecordError(err)
			span.SetStatus(codes.Error, "get request failed")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	span.SetAttributes(attribute.String("kick.target.kind", request.Spec.TargetRef.Kind), attribute.String("kick.target.name", request.Spec.TargetRef.Name))
	if isTerminalPhase(request.Status.Phase) {
		return r.reconcileTerminalRequest(ctx, &request)
	}

	if !supportedTargetRef(request.Spec.TargetRef) {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "InvalidTarget", "unsupported or empty targetRef")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, err
		}
		r.recordTransition(&request, kickv1alpha1.KickRequestPhaseFailed, "InvalidTarget", "unsupported or empty targetRef", "")
		return ctrl.Result{}, nil
	}

	targetKey := types.NamespacedName{Namespace: req.Namespace, Name: request.Spec.TargetRef.Name}
	workload, err := loadTargetWorkload(ctx, r.Client, request.Spec.TargetRef, targetKey)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if statusErr := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
				setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "TargetNotFound", "target workload no longer exists")
			}); statusErr != nil {
				observeControllerError("kickrequest", "UpdateStatus")
				return ctrl.Result{}, statusErr
			}
			r.recordTransition(&request, kickv1alpha1.KickRequestPhaseFailed, "TargetNotFound", "target workload no longer exists", "")
			return ctrl.Result{}, nil
		}
		observeControllerError("kickrequest", "GetWorkload")
		return ctrl.Result{}, err
	}

	var matchedPolicy *kickv1alpha1.KickPolicy
	if r.PolicyMatcher != nil {
		match, err := r.PolicyMatcher.MatchWorkload(ctx, workload.GetNamespace(), workloadLabels(workload))
		if err != nil {
			observeControllerError("kickrequest", "PolicyMatch")
			return ctrl.Result{}, err
		}
		if !match.Managed {
			reason := match.Reason
			if reason == policy.ReasonNoMatchingPolicy || reason == policy.ReasonPolicyDeleted {
				reason = policy.ReasonPolicyDeleted
			}
			if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
				setPhase(status, kickv1alpha1.KickRequestPhaseFailed, reason, "kickrequest canceled due to policy scope")
			}); err != nil {
				observeControllerError("kickrequest", "UpdateStatus")
				return ctrl.Result{}, err
			}
			r.recordTransition(&request, kickv1alpha1.KickRequestPhaseFailed, reason, "kickrequest canceled due to policy scope", "")
			return ctrl.Result{}, nil
		}
		matchedPolicy = match.Policy
	}

	now := r.now().UTC()

	// KICK-native windows gate the kick without a GitOps provider or owner.
	if nativeWindows := policyNativeWindows(matchedPolicy); len(nativeWindows) > 0 {
		decision, evalErr := schedule.Evaluate(now, nativeWindows)
		if evalErr != nil {
			if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
				setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateConfigurationError), evalErr.Error())
			}); err != nil {
				observeControllerError("kickrequest", "UpdateStatus")
				return ctrl.Result{}, err
			}
			r.recordTransition(&request, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateConfigurationError), evalErr.Error(), "")
			return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
		}
		if !decision.Allowed {
			if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
				status.Gate = kickv1alpha1.GateStatus{Reason: string(gitops.GateOutsideSchedule), Message: "blocked by KickPolicy window"}
				setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateOutsideSchedule), "blocked by KickPolicy window")
			}); err != nil {
				observeControllerError("kickrequest", "UpdateStatus")
				return ctrl.Result{}, err
			}
			r.recordTransition(&request, kickv1alpha1.KickRequestPhaseWaitingForGate, string(gitops.GateOutsideSchedule), "blocked by KickPolicy window", "")
			requeue := r.requeueInterval()
			if decision.RequeueAt != nil {
				if d := decision.RequeueAt.Sub(now); d > 0 {
					requeue = d
				}
			}
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		return r.evaluateFreshnessAndExecute(ctx, req, &request, workload, targetKey, kickv1alpha1.GitOpsOwnerStatus{}, gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed, Message: "allowed by KickPolicy window"})
	}

	ownerStatus, gateDecision, err := r.GateResolver.ResolveOwnerAndGate(ctx, workload, now)
	if err != nil {
		observeControllerError("kickrequest", "ResolveOwnerAndGate")
		return ctrl.Result{}, err
	}

	if !gitops.MayExecute(gateDecision) {
		phase := phaseForClosedGate(gateDecision.Reason)
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			setPhase(status, phase, string(gateDecision.Reason), gateDecision.Message)
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, err
		}
		r.recordTransition(&request, phase, string(gateDecision.Reason), gateDecision.Message, ownerStatus.Provider)
		return requeueForGate(gateDecision, now, r.requeueInterval()), nil
	}

	return r.evaluateFreshnessAndExecute(ctx, req, &request, workload, targetKey, ownerStatus, gateDecision)
}

func (r *KickRequestReconciler) evaluateFreshnessAndExecute(
	ctx context.Context,
	req ctrl.Request,
	request *kickv1alpha1.KickRequest,
	workload client.Object,
	targetKey types.NamespacedName,
	ownerStatus kickv1alpha1.GitOpsOwnerStatus,
	gateDecision gitops.GateDecision,
) (ctrl.Result, error) {
	deps := dependency.ExtractDependenciesForObject(workload)
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

	if freshnessDecision.BlockingReason != "" && !freshnessDecision.RestartRequired {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.CurrentRollout = rolloutToStatus(freshnessDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForRollout, freshnessDecision.BlockingReason, "workload rollout is still in progress")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseWaitingForRollout, freshnessDecision.BlockingReason, "workload rollout is still in progress", ownerStatus.Provider)
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
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
			return ctrl.Result{}, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseNoLongerRequired, "Fresh", "live workload is already fresh", ownerStatus.Provider)
		return ctrl.Result{}, nil
	}

	// Transition to Executing once. The executor owns CurrentRollout.StartedAt
	// (the time KICK issues the restart); pre-seeding it from the existing
	// ReplicaSet would make the executor treat the restart as already issued.
	if request.Status.Phase != kickv1alpha1.KickRequestPhaseExecuting {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseExecuting, "RestartRequired", "restart required after live freshness check")
		}); err != nil {
			observeControllerError("kickrequest", "UpdateStatus")
			return ctrl.Result{}, err
		}
		r.recordTransition(request, kickv1alpha1.KickRequestPhaseExecuting, "RestartRequired", "restart required after live freshness check", ownerStatus.Provider)
	}

	result, err := r.RestartExecutor.Execute(ctx, req.NamespacedName, request.Spec.TargetRef, targetKey)
	if err != nil {
		observeControllerError("kickrequest", "ExecuteRestart")
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
		span.SetStatus(codes.Error, "restart execution failed")
		return ctrl.Result{}, err
	}

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
	specWindows := pol.Spec.GitOps.Schedule.Windows
	if len(specWindows) == 0 {
		return nil
	}
	out := make([]schedule.Window, 0, len(specWindows))
	for _, w := range specWindows {
		out = append(out, schedule.Window{
			Allow:    w.Kind == kickv1alpha1.KickPolicyWindowKindAllow,
			Schedule: w.Schedule,
			Duration: w.Duration,
			TimeZone: w.TimeZone,
		})
	}
	return out
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
				return cond.LastTransitionTime.Time.UTC()
			}
		}
		if !request.CreationTimestamp.IsZero() {
			return request.CreationTimestamp.Time.UTC()
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
	default:
		return observation.SourceIdentity{}, false
	}
	return identity, true
}

func supportedTargetRef(ref kickv1alpha1.ObjectReference) bool {
	if ref.Name == "" || ref.APIVersion != "apps/v1" {
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
	if phase == kickv1alpha1.KickRequestPhaseSucceeded || phase == kickv1alpha1.KickRequestPhaseNoLongerRequired || phase == kickv1alpha1.KickRequestPhaseFailed {
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
	return phase == kickv1alpha1.KickRequestPhaseSucceeded || phase == kickv1alpha1.KickRequestPhaseNoLongerRequired || phase == kickv1alpha1.KickRequestPhaseFailed
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
	case kickv1alpha1.KickRequestPhaseExecuting:
		observeRestartResult(provider, "started")
	}

	if r.Recorder != nil {
		eventType, eventReason := phaseToEvent(newPhase, reason)
		if eventReason != "" {
			r.Recorder.Event(request, eventType, eventReason, message)
		}
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
	default:
		return "", ""
	}
}
