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
	Inspect(ctx context.Context, workload client.Object) (RolloutState, error)
}

// LiveRolloutInspector loads owned ReplicaSets and evaluates current rollout state.
type LiveRolloutInspector struct {
	Client client.Client
}

func (i *LiveRolloutInspector) Inspect(ctx context.Context, workload client.Object) (RolloutState, error) {
	switch obj := workload.(type) {
	case *appsv1.Deployment:
		if obj == nil {
			return RolloutState{Reason: ReasonNoMatchingReplicaSet}, nil
		}
		var list appsv1.ReplicaSetList
		if err := i.Client.List(ctx, &list, client.InNamespace(obj.Namespace)); err != nil {
			return RolloutState{}, err
		}
		state := InspectWithReplicaSets(obj, list.Items)
		if state.StartedAt.IsZero() {
			state.StartedAt = restartStartedAt(obj.Spec.Template.Annotations, obj.CreationTimestamp.Time)
		} else if obj.Spec.Template.Annotations == nil || obj.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] == "" {
			if t, ok := latestDeploymentConditionTime(obj.Status.Conditions); ok && t.After(state.StartedAt) {
				state.StartedAt = t
			}
		}
		return state, nil
	case *appsv1.StatefulSet:
		return inspectStatefulSet(obj), nil
	case *appsv1.DaemonSet:
		return inspectDaemonSet(obj), nil
	default:
		return RolloutState{Reason: ReasonNoMatchingReplicaSet}, nil
	}
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

func latestDeploymentConditionTime(conditions []appsv1.DeploymentCondition) (time.Time, bool) {
	var latest time.Time
	found := false
	for _, cond := range conditions {
		if !cond.LastTransitionTime.IsZero() {
			t := cond.LastTransitionTime.UTC()
			if !found || t.After(latest) {
				latest = t
				found = true
			}
		}
		if !cond.LastUpdateTime.IsZero() {
			t := cond.LastUpdateTime.UTC()
			if !found || t.After(latest) {
				latest = t
				found = true
			}
		}
	}
	return latest, found
}

func inspectStatefulSet(statefulSet *appsv1.StatefulSet) RolloutState {
	if statefulSet == nil {
		return RolloutState{Reason: ReasonNoMatchingReplicaSet}
	}
	desired := int32(1)
	if statefulSet.Spec.Replicas != nil {
		desired = *statefulSet.Spec.Replicas
	}

	inProgress := statefulSet.Status.ObservedGeneration < statefulSet.Generation ||
		statefulSet.Status.UpdatedReplicas < desired ||
		statefulSet.Status.ReadyReplicas < desired ||
		(statefulSet.Status.CurrentRevision != "" && statefulSet.Status.UpdateRevision != "" && statefulSet.Status.CurrentRevision != statefulSet.Status.UpdateRevision)

	state := RolloutState{
		StartedAt:  restartStartedAt(statefulSet.Spec.Template.Annotations, statefulSet.CreationTimestamp.Time),
		InProgress: inProgress,
		Complete:   !inProgress,
	}
	if inProgress {
		state.Reason = ReasonRolloutInProgress
	}
	return state
}

func inspectDaemonSet(daemonSet *appsv1.DaemonSet) RolloutState {
	if daemonSet == nil {
		return RolloutState{Reason: ReasonNoMatchingReplicaSet}
	}
	desired := daemonSet.Status.DesiredNumberScheduled
	inProgress := daemonSet.Status.ObservedGeneration < daemonSet.Generation ||
		daemonSet.Status.UpdatedNumberScheduled < desired ||
		daemonSet.Status.NumberAvailable < desired

	startedAt := restartStartedAt(daemonSet.Spec.Template.Annotations, daemonSet.CreationTimestamp.Time)
	if daemonSet.Spec.Template.Annotations == nil || daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] == "" {
		for _, cond := range daemonSet.Status.Conditions {
			if cond.LastTransitionTime.IsZero() {
				continue
			}
			if cond.LastTransitionTime.After(startedAt) {
				startedAt = cond.LastTransitionTime.UTC()
			}
		}
	}

	state := RolloutState{
		StartedAt:  startedAt,
		InProgress: inProgress,
		Complete:   !inProgress,
	}
	if inProgress {
		state.Reason = ReasonRolloutInProgress
	}
	return state
}

func restartStartedAt(annotations map[string]string, fallback time.Time) time.Time {
	if annotations != nil {
		if raw := annotations["kubectl.kubernetes.io/restartedAt"]; raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				return parsed.UTC()
			}
		}
	}
	if fallback.IsZero() {
		return time.Now().UTC()
	}
	return fallback.UTC()
}
