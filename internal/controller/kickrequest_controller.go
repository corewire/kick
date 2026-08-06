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
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	statusConditionProgressing = "Progressing"
	defaultRequeueInterval     = 30 * time.Second
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
	Evaluate(ctx context.Context, deployment *appsv1.Deployment, currentDependencies []dependency.DependencyRef, latestRelevantChanges map[dependency.DependencyRef]time.Time) (freshness.FreshnessDecision, error)
}

type RestartExecutor interface {
	Execute(ctx context.Context, requestKey, deploymentKey types.NamespacedName) (executor.Result, error)
}

type KickRequestReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	GateResolver       GateResolver
	ObservationStore   observation.Store
	FreshnessEvaluator FreshnessEvaluator
	RestartExecutor    RestartExecutor
	Clock              func() time.Time
	RequeueInterval    time.Duration
}

func (r *KickRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.GateResolver == nil || r.ObservationStore == nil || r.FreshnessEvaluator == nil || r.RestartExecutor == nil {
		return ctrl.Result{}, fmt.Errorf("kickrequest reconciler dependencies are not fully configured")
	}

	var request kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &request); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if isTerminalPhase(request.Status.Phase) {
		return ctrl.Result{}, nil
	}

	if request.Spec.TargetRef.Kind != "Deployment" || request.Spec.TargetRef.Name == "" {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "InvalidTarget", "unsupported or empty targetRef")
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	deploymentKey := types.NamespacedName{Namespace: req.Namespace, Name: request.Spec.TargetRef.Name}
	var deployment appsv1.Deployment
	if err := r.Get(ctx, deploymentKey, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			if statusErr := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
				setPhase(status, kickv1alpha1.KickRequestPhaseFailed, "TargetNotFound", "target deployment no longer exists")
			}); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	now := r.now().UTC()
	ownerStatus, gateDecision, err := r.GateResolver.ResolveOwnerAndGate(ctx, &deployment, now)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !gitops.MayExecute(gateDecision) {
		phase := phaseForClosedGate(gateDecision.Reason)
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			setPhase(status, phase, string(gateDecision.Reason), gateDecision.Message)
		}); err != nil {
			return ctrl.Result{}, err
		}
		return requeueForGate(gateDecision, now, r.requeueInterval()), nil
	}

	deps := dependency.ExtractDependencies(&deployment)
	latestChanges, err := r.latestRelevantChanges(ctx, deps)
	if err != nil {
		return ctrl.Result{}, err
	}

	freshnessDecision, err := r.FreshnessEvaluator.Evaluate(ctx, &deployment, deps, latestChanges)
	if err != nil {
		return ctrl.Result{}, err
	}

	if freshnessDecision.BlockingReason != "" && !freshnessDecision.RestartRequired {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.CurrentRollout = rolloutToStatus(freshnessDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseWaitingForRollout, freshnessDecision.BlockingReason, "deployment rollout is still in progress")
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
	}

	if !freshnessDecision.RestartRequired {
		if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
			status.Owner = ownerStatus
			status.Gate = gateToStatus(gateDecision)
			status.CurrentRollout = rolloutToStatus(freshnessDecision)
			status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
			setPhase(status, kickv1alpha1.KickRequestPhaseNoLongerRequired, "Fresh", "live deployment is already fresh")
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		status.Owner = ownerStatus
		status.Gate = gateToStatus(gateDecision)
		status.CurrentRollout = rolloutToStatus(freshnessDecision)
		status.LatestObservedDependencyChange = toMetav1TimePtr(freshnessDecision.LatestChange)
		setPhase(status, kickv1alpha1.KickRequestPhaseExecuting, "RestartRequired", "restart required after live freshness check")
	}); err != nil {
		return ctrl.Result{}, err
	}

	result, err := r.RestartExecutor.Execute(ctx, req.NamespacedName, deploymentKey)
	if err != nil {
		return ctrl.Result{}, err
	}

	var updated kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &updated); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if isTerminalPhase(updated.Status.Phase) || result.Complete || result.Failed {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
}

func (r *KickRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&kickv1alpha1.KickRequest{}).Complete(r)
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
