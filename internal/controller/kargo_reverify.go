package controller

import (
	"context"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/gitops/kargo"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KargoReverifier snapshots verification identity and sends the Kargo reverify
// annotation. It must not create a Promotion or restart a workload.
type KargoReverifier interface {
	SnapshotVerification(ctx context.Context, namespace, name string) (kickv1alpha1.KargoReverificationStatus, error)
	RequestReverification(ctx context.Context, recorded kickv1alpha1.KargoReverificationStatus) (kargo.ReverifyResult, error)
}

func (r *KickRequestReconciler) captureReverification(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest, matchedPolicy *kickv1alpha1.KickPolicy, ownerStatus kickv1alpha1.GitOpsOwnerStatus) error {
	if !shouldReverify(matchedPolicy, ownerStatus) || kargoReverificationRecorded(request) || r.KargoReverifier == nil {
		return nil
	}
	snapshot, err := r.KargoReverifier.SnapshotVerification(ctx, ownerStatus.Namespace, ownerStatus.Name)
	if err != nil {
		observeControllerError("kickrequest", "SnapshotVerification")
		return err
	}
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(status *kickv1alpha1.KickRequestStatus) {
		status.KargoReverification = &snapshot
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return err
	}
	request.Status.KargoReverification = &snapshot
	return nil
}

// finishReverification runs only after a successful rollout. A patch error
// requeues this request and must not call the restart executor again.
func (r *KickRequestReconciler) finishReverification(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest) (ctrl.Result, error) {
	if !kargoReverifyPending(request) || r.KargoReverifier == nil || request.Status.KargoReverification == nil {
		return r.reconcileTerminalRequest(ctx, request)
	}
	enabled, fresh, err := r.reverifyReady(ctx, req, request)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !enabled {
		return r.persistReverification(ctx, req, skippedReverification(request.Status.KargoReverification))
	}
	if !fresh {
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
	}
	result, err := r.KargoReverifier.RequestReverification(ctx, *request.Status.KargoReverification)
	if err != nil {
		observeControllerError("kickrequest", "RequestReverification")
		return ctrl.Result{}, err
	}
	if result.Retry {
		return ctrl.Result{RequeueAfter: r.requeueInterval()}, nil
	}
	return r.persistReverification(ctx, req, result.Status)
}

func (r *KickRequestReconciler) persistReverification(ctx context.Context, req ctrl.Request, status kickv1alpha1.KargoReverificationStatus) (ctrl.Result, error) {
	if err := r.updateRequestStatus(ctx, req.NamespacedName, func(current *kickv1alpha1.KickRequestStatus) {
		current.KargoReverification = &status
	}); err != nil {
		observeControllerError("kickrequest", "UpdateStatus")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// reverifyReady reports whether the opt-in is still set and the workload is
// fresh. A missing workload cannot be confirmed fresh, so the annotation is
// skipped rather than retried forever.
func (r *KickRequestReconciler) reverifyReady(ctx context.Context, req ctrl.Request, request *kickv1alpha1.KickRequest) (bool, bool, error) {
	targetKey := types.NamespacedName{Namespace: req.Namespace, Name: request.Spec.TargetRef.Name}
	workload, err := loadTargetWorkload(ctx, r.Client, request.Spec.TargetRef, targetKey)
	if apierrors.IsNotFound(err) {
		return false, false, nil
	}
	if err != nil {
		observeControllerError("kickrequest", "GetWorkload")
		return false, false, err
	}
	var matchedPolicy *kickv1alpha1.KickPolicy
	if r.PolicyMatcher != nil {
		match, matchErr := r.PolicyMatcher.MatchWorkload(ctx, workload.GetNamespace(), workloadLabels(workload))
		if matchErr != nil {
			observeControllerError("kickrequest", "PolicyMatch")
			return false, false, matchErr
		}
		if !match.Managed || match.Policy == nil || !match.Policy.Spec.GitOps.ReverifyAfterRestart || match.Policy.Spec.DryRun {
			return false, false, nil
		}
		matchedPolicy = match.Policy
	}
	deps := dependency.ExtractDependenciesForObject(workload)
	deps, err = r.scopeDependencies(ctx, matchedPolicy, deps)
	if err != nil {
		observeControllerError("kickrequest", "ScopeDependencies")
		return false, false, err
	}
	latestChanges, err := r.latestRelevantChanges(ctx, deps)
	if err != nil {
		observeControllerError("kickrequest", "LatestRelevantChanges")
		return false, false, err
	}
	decision, err := r.FreshnessEvaluator.Evaluate(ctx, workload, deps, latestChanges)
	if err != nil {
		observeControllerError("kickrequest", "FreshnessEvaluate")
		return false, false, err
	}
	decision = mergeRecordedChange(decision, request.Status.LatestObservedDependencyChange)
	return true, !decision.RestartRequired && decision.BlockingReason == "", nil
}

func shouldReverify(pol *kickv1alpha1.KickPolicy, owner kickv1alpha1.GitOpsOwnerStatus) bool {
	return pol != nil &&
		pol.Spec.GitOps.ReverifyAfterRestart &&
		pol.Spec.GitOps.Provider == kickv1alpha1.KickPolicyProviderKargo &&
		owner.Provider == "kargo" &&
		!pol.Spec.DryRun
}

func kargoReverifyPending(request *kickv1alpha1.KickRequest) bool {
	return request != nil &&
		request.Status.Phase == kickv1alpha1.KickRequestPhaseSucceeded &&
		request.Status.KargoReverification != nil &&
		request.Status.KargoReverification.State == kickv1alpha1.KargoReverificationPending
}

func kargoReverificationRecorded(request *kickv1alpha1.KickRequest) bool {
	return request.Status.KargoReverification != nil && request.Status.KargoReverification.State != ""
}

func skippedReverification(recorded *kickv1alpha1.KargoReverificationStatus) kickv1alpha1.KargoReverificationStatus {
	status := *recorded
	status.State = kickv1alpha1.KargoReverificationSkipped
	return status
}
