package executor

import (
	"context"
	"encoding/json"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

type Result struct {
	Patched  bool
	Complete bool
	Failed   bool
	Reason   string
}

type RestartExecutor struct {
	Client  client.Client
	Now     func() time.Time
	Timeout time.Duration
}

func NewRestartExecutor(c client.Client, timeout time.Duration) *RestartExecutor {
	return &RestartExecutor{Client: c, Now: time.Now, Timeout: timeout}
}

func (e *RestartExecutor) Execute(ctx context.Context, requestKey, deploymentKey types.NamespacedName) (Result, error) {
	var request kickv1alpha1.KickRequest
	if err := e.Client.Get(ctx, requestKey, &request); err != nil {
		return Result{}, err
	}
	var deployment appsv1.Deployment
	if err := e.Client.Get(ctx, deploymentKey, &deployment); err != nil {
		return Result{}, err
	}

	if request.Status.Phase == kickv1alpha1.KickRequestPhaseSucceeded ||
		request.Status.Phase == kickv1alpha1.KickRequestPhaseNoLongerRequired ||
		request.Status.Phase == kickv1alpha1.KickRequestPhaseFailed {
		return Result{}, nil
	}

	if request.Status.Phase == kickv1alpha1.KickRequestPhaseExecuting && request.Status.CurrentRollout.StartedAt != nil {
		if rolloutComplete(&deployment) {
			if err := e.updateRequestStatus(ctx, requestKey, func(status *kickv1alpha1.KickRequestStatus) {
				status.Phase = kickv1alpha1.KickRequestPhaseSucceeded
			}); err != nil {
				return Result{}, err
			}
			return Result{Complete: true}, nil
		}
		if e.Timeout > 0 && e.Now().UTC().After(request.Status.CurrentRollout.StartedAt.Add(e.Timeout)) {
			if err := e.updateRequestStatus(ctx, requestKey, func(status *kickv1alpha1.KickRequestStatus) {
				status.Phase = kickv1alpha1.KickRequestPhaseFailed
			}); err != nil {
				return Result{}, err
			}
			return Result{Failed: true, Reason: "RolloutTimeout"}, nil
		}
		return Result{}, nil
	}

	now := e.Now().UTC()
	if err := e.updateRequestStatus(ctx, requestKey, func(status *kickv1alpha1.KickRequestStatus) {
		status.Phase = kickv1alpha1.KickRequestPhaseExecuting
		t := metav1.NewTime(now)
		status.CurrentRollout.StartedAt = &t
	}); err != nil {
		return Result{}, err
	}

	patch, err := restartPatch(now)
	if err != nil {
		return Result{}, err
	}
	if err := e.Client.Patch(ctx, &deployment, client.RawPatch(types.MergePatchType, patch)); err != nil {
		if apierrors.IsConflict(err) {
			return Result{}, nil
		}
		return Result{}, err
	}
	return Result{Patched: true}, nil
}

func (e *RestartExecutor) updateRequestStatus(ctx context.Context, key types.NamespacedName, mutate func(*kickv1alpha1.KickRequestStatus)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var req kickv1alpha1.KickRequest
		if err := e.Client.Get(ctx, key, &req); err != nil {
			return err
		}
		mutate(&req.Status)
		return e.Client.Status().Update(ctx, &req)
	})
}

func restartPatch(now time.Time) ([]byte, error) {
	obj := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{restartedAtAnnotation: now.Format(time.RFC3339)},
				},
			},
		},
	}
	return json.Marshal(obj)
}

func rolloutComplete(deployment *appsv1.Deployment) bool {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.AvailableReplicas >= desired
}
