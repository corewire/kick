package rollout

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// argoRolloutPhaseHealthy is the only Argo Rollouts phase in which a rollout
	// is finished and serving the desired revision.
	argoRolloutPhaseHealthy = "Healthy"
	// argoRolloutPhaseDegraded means the rollout will not progress on its own.
	argoRolloutPhaseDegraded = "Degraded"

	ReasonArgoRolloutDegraded = "ArgoRolloutDegraded"
)

// inspectArgoRollout derives rollout state for an Argo Rollout.
//
// A Rollout starts new pods in two distinct ways: a spec change creates a new
// ReplicaSet, while spec.restartAt recycles pods in place without creating one.
// The rollout start time is therefore the later of the newest owned ReplicaSet
// and status.restartedAt.
func inspectArgoRollout(ctx context.Context, c client.Client, obj *unstructured.Unstructured) (RolloutState, error) {
	if obj == nil {
		return RolloutState{Reason: ReasonNoMatchingReplicaSet}, nil
	}

	startedAt, err := argoRolloutStartedAt(ctx, c, obj)
	if err != nil {
		return RolloutState{}, err
	}

	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	desired := argoRolloutDesiredReplicas(obj)
	updated := nestedInt64(obj, "status", "updatedReplicas")
	available := nestedInt64(obj, "status", "availableReplicas")

	state := RolloutState{StartedAt: startedAt}
	state.Failed = phase == argoRolloutPhaseDegraded
	if state.Failed {
		state.Reason = ReasonArgoRolloutDegraded
	}
	state.InProgress = phase != argoRolloutPhaseHealthy || updated < desired || available < desired
	if state.InProgress && state.Reason == "" {
		state.Reason = ReasonRolloutInProgress
	}
	state.Complete = !state.InProgress && !state.Failed
	return state, nil
}

// argoRolloutStartedAt returns when the currently running pods were last
// started.
func argoRolloutStartedAt(ctx context.Context, c client.Client, obj *unstructured.Unstructured) (time.Time, error) {
	startedAt := obj.GetCreationTimestamp().Time

	var replicaSets appsv1.ReplicaSetList
	if err := c.List(ctx, &replicaSets, client.InNamespace(obj.GetNamespace())); err != nil {
		return time.Time{}, err
	}
	for i := range replicaSets.Items {
		rs := &replicaSets.Items[i]
		if !isControlledByUID(rs.OwnerReferences, string(obj.GetUID())) {
			continue
		}
		if rs.CreationTimestamp.After(startedAt) {
			startedAt = rs.CreationTimestamp.Time
		}
	}

	if restarted, ok := nestedTime(obj, "status", "restartedAt"); ok && restarted.After(startedAt) {
		startedAt = restarted
	}
	return startedAt.UTC(), nil
}

func argoRolloutDesiredReplicas(obj *unstructured.Unstructured) int64 {
	if value, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas"); err == nil && found {
		return value
	}
	return 1
}

func nestedInt64(obj *unstructured.Unstructured, fields ...string) int64 {
	value, found, err := unstructured.NestedInt64(obj.Object, fields...)
	if err != nil || !found {
		return 0
	}
	return value
}

func nestedTime(obj *unstructured.Unstructured, fields ...string) (time.Time, bool) {
	raw, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil || !found || raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func isControlledByUID(owners []metav1.OwnerReference, uid string) bool {
	for _, owner := range owners {
		if string(owner.UID) == uid && owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}

// ArgoRolloutComplete reports whether an Argo Rollout finished rolling out.
func ArgoRolloutComplete(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase != argoRolloutPhaseHealthy {
		return false
	}
	desired := argoRolloutDesiredReplicas(obj)
	return nestedInt64(obj, "status", "updatedReplicas") >= desired &&
		nestedInt64(obj, "status", "availableReplicas") >= desired
}
