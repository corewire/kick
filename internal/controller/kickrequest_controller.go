package controller

import (
	"context"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KickRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *KickRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var request kickv1alpha1.KickRequest
	if err := r.Get(ctx, req.NamespacedName, &request); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Implementation begins in tasks/13-kickrequest-controller.md. Live state
	// must be recomputed; status is never the final authority.
	return ctrl.Result{}, nil
}

func (r *KickRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&kickv1alpha1.KickRequest{}).Complete(r)
}
