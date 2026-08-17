package controller

import (
	"context"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/gitops"
)

const (
	// KickPolicyReadyCondition reports whether the policy can be enforced as written.
	KickPolicyReadyCondition = "Ready"

	kickPolicyReasonAccepted            = "Accepted"
	kickPolicyReasonProviderUnavailable = "ProviderUnavailable"
)

// ProviderRegistry is the part of the GitOps registry the policy validator needs.
type ProviderRegistry interface {
	ProviderByName(string) (gitops.Provider, bool)
	Unavailability(string) (gitops.Unavailability, bool)
}

// KickPolicyReconciler reports whether a policy's GitOps provider is actually
// wired up in this operator. Without it the mismatch would only surface per
// workload, on KickRequests that are created long after the policy is applied.
type KickPolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Registry ProviderRegistry
}

func (r *KickPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var policy kickv1alpha1.KickPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	condition := metav1.Condition{
		Type:               KickPolicyReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             kickPolicyReasonAccepted,
		Message:            "policy accepted",
		ObservedGeneration: policy.Generation,
	}
	if reason, message, ok := r.providerProblem(&policy); ok {
		condition.Status = metav1.ConditionFalse
		condition.Reason = reason
		condition.Message = message
	}

	patched := policy.DeepCopy()
	patched.Status.ObservedGeneration = policy.Generation
	changed := apimeta.SetStatusCondition(&patched.Status.Conditions, condition)
	if changed && condition.Status == metav1.ConditionFalse && r.Recorder != nil {
		r.Recorder.Event(&policy, "Warning", condition.Reason, condition.Message)
	}
	if !changed && patched.Status.ObservedGeneration == policy.Status.ObservedGeneration {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.Status().Update(ctx, patched)
}

// providerProblem reports the condition reason and remediation for a policy
// whose configured provider is not registered.
func (r *KickPolicyReconciler) providerProblem(policy *kickv1alpha1.KickPolicy) (string, string, bool) {
	name := gateProviderName(policy)
	if name == "" || r.Registry == nil {
		return "", "", false
	}
	if _, ok := r.Registry.ProviderByName(name); ok {
		return "", "", false
	}
	reason := kickPolicyReasonProviderUnavailable
	if unavailability, ok := r.Registry.Unavailability(name); ok {
		reason = unavailability.Reason
	}
	return reason, providerUnavailableMessage(r.Registry, name), true
}

func (r *KickPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		//nolint:staticcheck // controller-runtime still returns the client-go EventRecorder interface from this method.
		r.Recorder = mgr.GetEventRecorderFor("kickpolicy-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("kickpolicy").
		For(&kickv1alpha1.KickPolicy{}).
		Complete(r)
}
