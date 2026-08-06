package rollout

import (
	"context"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReasonNoMatchingReplicaSet     = "NoMatchingReplicaSet"
	ReasonAmbiguousReplicaSet      = "AmbiguousReplicaSetMatch"
	ReasonRolloutInProgress        = "RolloutInProgress"
	ReasonDeploymentPaused         = "DeploymentPaused"
	ReasonProgressDeadlineExceeded = "ProgressDeadlineExceeded"
)

// RolloutState describes the current rollout inspection result.
type RolloutState struct {
	CurrentReplicaSet types.NamespacedName
	StartedAt         time.Time
	InProgress        bool
	Complete          bool
	Failed            bool
	Reason            string
}

// RolloutInspector inspects deployment rollout state using live cluster state.
type RolloutInspector interface {
	Inspect(ctx context.Context, deployment *appsv1.Deployment) (RolloutState, error)
}

// LiveRolloutInspector loads owned ReplicaSets and evaluates current rollout state.
type LiveRolloutInspector struct {
	Client client.Client
}

func (i *LiveRolloutInspector) Inspect(ctx context.Context, deployment *appsv1.Deployment) (RolloutState, error) {
	if deployment == nil {
		return RolloutState{Reason: ReasonNoMatchingReplicaSet}, nil
	}

	var list appsv1.ReplicaSetList
	if err := i.Client.List(ctx, &list, client.InNamespace(deployment.Namespace)); err != nil {
		return RolloutState{}, err
	}
	return InspectWithReplicaSets(deployment, list.Items), nil
}

// InspectWithReplicaSets is the pure algorithm used by tests and live inspector.
func InspectWithReplicaSets(deployment *appsv1.Deployment, replicaSets []appsv1.ReplicaSet) RolloutState {
	state := RolloutState{}
	if deployment == nil {
		state.Reason = ReasonNoMatchingReplicaSet
		return state
	}

	matches := matchingReplicaSets(deployment, replicaSets)
	if len(matches) == 0 {
		state.Reason = ReasonNoMatchingReplicaSet
	} else if len(matches) > 1 {
		state.Reason = ReasonAmbiguousReplicaSet
	} else {
		rs := matches[0]
		state.CurrentReplicaSet = types.NamespacedName{Namespace: rs.Namespace, Name: rs.Name}
		state.StartedAt = rs.CreationTimestamp.Time
	}

	state.Failed = progressingFailed(deployment.Status.Conditions)
	if state.Failed {
		state.Reason = ReasonProgressDeadlineExceeded
	}

	if deployment.Spec.Paused {
		state.InProgress = false
		if state.Reason == "" {
			state.Reason = ReasonDeploymentPaused
		}
		state.Complete = state.CurrentReplicaSet.Name != ""
		return state
	}

	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	state.InProgress = deployment.Status.ObservedGeneration < deployment.Generation ||
		deployment.Status.UpdatedReplicas < desired ||
		deployment.Status.Replicas > deployment.Status.UpdatedReplicas ||
		deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas
	if state.InProgress && state.Reason == "" {
		state.Reason = ReasonRolloutInProgress
	}
	state.Complete = state.CurrentReplicaSet.Name != "" && !state.InProgress && !state.Failed
	return state
}

func matchingReplicaSets(deployment *appsv1.Deployment, replicaSets []appsv1.ReplicaSet) []appsv1.ReplicaSet {
	owned := make([]appsv1.ReplicaSet, 0)
	for _, rs := range replicaSets {
		if !isControlledByDeployment(rs, deployment) {
			continue
		}
		if TemplatesEquivalent(deployment.Spec.Template, rs.Spec.Template) {
			owned = append(owned, rs)
		}
	}
	sort.Slice(owned, func(i, j int) bool {
		if !owned[i].CreationTimestamp.Equal(&owned[j].CreationTimestamp) {
			return owned[i].CreationTimestamp.Before(&owned[j].CreationTimestamp)
		}
		return owned[i].Name < owned[j].Name
	})
	return owned
}

func isControlledByDeployment(rs appsv1.ReplicaSet, deployment *appsv1.Deployment) bool {
	for _, owner := range rs.OwnerReferences {
		if owner.UID == deployment.UID && owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}

func progressingFailed(conditions []appsv1.DeploymentCondition) bool {
	for _, cond := range conditions {
		if cond.Type == appsv1.DeploymentProgressing && cond.Status == corev1.ConditionFalse && cond.Reason == "ProgressDeadlineExceeded" {
			return true
		}
	}
	return false
}
