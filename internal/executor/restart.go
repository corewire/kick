package executor

import (
	"context"
	"encoding/json"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

func (e *RestartExecutor) Execute(ctx context.Context, requestKey types.NamespacedName, targetRef kickv1alpha1.ObjectReference, targetKey types.NamespacedName) (Result, error) {
	_, span := otel.Tracer("kick.executor").Start(ctx, "restart.execute")
	defer span.End()
	span.SetAttributes(
		attribute.String("kick.request.namespace", requestKey.Namespace),
		attribute.String("kick.request.name", requestKey.Name),
		attribute.String("kick.target.kind", targetRef.Kind),
		attribute.String("kick.target.name", targetRef.Name),
	)

	var request kickv1alpha1.KickRequest
	if err := e.Client.Get(ctx, requestKey, &request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get request failed")
		return Result{}, err
	}
	workload, err := getWorkload(ctx, e.Client, targetRef, targetKey)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get workload failed")
		return Result{}, err
	}

	if request.Status.Phase == kickv1alpha1.KickRequestPhaseSucceeded ||
		request.Status.Phase == kickv1alpha1.KickRequestPhaseNoLongerRequired ||
		request.Status.Phase == kickv1alpha1.KickRequestPhaseFailed {
		return Result{}, nil
	}

	if request.Status.Phase == kickv1alpha1.KickRequestPhaseExecuting && request.Status.CurrentRollout.StartedAt != nil {
		if rolloutComplete(workload) {
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
	if err := e.Client.Patch(ctx, workload, client.RawPatch(types.MergePatchType, patch)); err != nil {
		if apierrors.IsConflict(err) {
			return Result{}, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "patch failed")
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

func rolloutComplete(workload client.Object) bool {
	switch obj := workload.(type) {
	case *appsv1.Deployment:
		return rolloutCompleteDeployment(obj)
	case *appsv1.StatefulSet:
		return rolloutCompleteStatefulSet(obj)
	case *appsv1.DaemonSet:
		return rolloutCompleteDaemonSet(obj)
	default:
		return false
	}
}

func rolloutCompleteDeployment(deployment *appsv1.Deployment) bool {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.AvailableReplicas >= desired
}

func rolloutCompleteStatefulSet(statefulSet *appsv1.StatefulSet) bool {
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}
	return statefulSet.Status.ObservedGeneration >= statefulSet.Generation &&
		statefulSet.Status.UpdatedReplicas == desired &&
		statefulSet.Status.ReadyReplicas >= desired &&
		(statefulSet.Status.CurrentRevision == "" || statefulSet.Status.UpdateRevision == "" || statefulSet.Status.CurrentRevision == statefulSet.Status.UpdateRevision)
}

func rolloutCompleteDaemonSet(daemonSet *appsv1.DaemonSet) bool {
	desired := daemonSet.Status.DesiredNumberScheduled
	return daemonSet.Status.ObservedGeneration >= daemonSet.Generation &&
		daemonSet.Status.UpdatedNumberScheduled == desired &&
		daemonSet.Status.NumberAvailable >= desired
}

func getWorkload(ctx context.Context, c client.Client, targetRef kickv1alpha1.ObjectReference, key types.NamespacedName) (client.Object, error) {
	switch targetRef.Kind {
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
