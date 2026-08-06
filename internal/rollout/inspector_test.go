package rollout

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestInspectWithReplicaSetsScenarios(t *testing.T) {
	uid := types.UID("dep-uid")
	base := newDeployment("ns", "api", uid)

	tests := []struct {
		name   string
		dep    *appsv1.Deployment
		rss    []appsv1.ReplicaSet
		assert func(t *testing.T, got RolloutState)
	}{
		{
			name: "normal rollout selects matching owner not newest",
			dep:  base,
			rss: []appsv1.ReplicaSet{
				newReplicaSet("ns", "api-newest-nonmatch", uid, map[string]string{"app": "other"}, time.Date(2026, 8, 6, 11, 0, 0, 0, time.UTC)),
				newReplicaSet("ns", "api-current", uid, map[string]string{"app": "api"}, time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)),
			},
			assert: func(t *testing.T, got RolloutState) {
				if got.CurrentReplicaSet.Name != "api-current" {
					t.Fatalf("current rs = %s", got.CurrentReplicaSet.Name)
				}
			},
		},
		{
			name: "active rollout is in progress",
			dep: func() *appsv1.Deployment {
				d := base.DeepCopy()
				d.Status.UpdatedReplicas = 1
				d.Status.Replicas = 3
				d.Status.AvailableReplicas = 1
				return d
			}(),
			rss: []appsv1.ReplicaSet{newReplicaSet("ns", "api-current", uid, map[string]string{"app": "api"}, time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC))},
			assert: func(t *testing.T, got RolloutState) {
				if !got.InProgress || got.Reason != ReasonRolloutInProgress {
					t.Fatalf("inProgress=%v reason=%s", got.InProgress, got.Reason)
				}
			},
		},
		{
			name: "rollback chooses old matching replicaset",
			dep:  base,
			rss: []appsv1.ReplicaSet{
				newReplicaSet("ns", "api-rev3", uid, map[string]string{"app": "v3"}, time.Date(2026, 8, 6, 11, 0, 0, 0, time.UTC)),
				newReplicaSet("ns", "api-rev2", uid, map[string]string{"app": "api"}, time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)),
			},
			assert: func(t *testing.T, got RolloutState) {
				if got.CurrentReplicaSet.Name != "api-rev2" {
					t.Fatalf("rollback current rs=%s", got.CurrentReplicaSet.Name)
				}
			},
		},
		{
			name: "pause is explicit",
			dep: func() *appsv1.Deployment {
				d := base.DeepCopy()
				d.Spec.Paused = true
				return d
			}(),
			rss: []appsv1.ReplicaSet{newReplicaSet("ns", "api-current", uid, map[string]string{"app": "api"}, time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC))},
			assert: func(t *testing.T, got RolloutState) {
				if got.Reason != ReasonDeploymentPaused || got.InProgress {
					t.Fatalf("paused reason=%s inProgress=%v", got.Reason, got.InProgress)
				}
			},
		},
		{
			name: "zero replicas deterministic",
			dep: func() *appsv1.Deployment {
				d := base.DeepCopy()
				zero := int32(0)
				d.Spec.Replicas = &zero
				d.Status.Replicas = 0
				d.Status.UpdatedReplicas = 0
				d.Status.AvailableReplicas = 0
				return d
			}(),
			rss: []appsv1.ReplicaSet{newReplicaSet("ns", "api-current", uid, map[string]string{"app": "api"}, time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC))},
			assert: func(t *testing.T, got RolloutState) {
				if !got.Complete || got.InProgress {
					t.Fatalf("complete=%v inProgress=%v", got.Complete, got.InProgress)
				}
			},
		},
		{
			name: "history cleanup no matching",
			dep:  base,
			rss:  []appsv1.ReplicaSet{newReplicaSet("ns", "api-old", uid, map[string]string{"app": "gone"}, time.Date(2026, 8, 6, 8, 0, 0, 0, time.UTC))},
			assert: func(t *testing.T, got RolloutState) {
				if got.Reason != ReasonNoMatchingReplicaSet {
					t.Fatalf("reason=%s", got.Reason)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InspectWithReplicaSets(tt.dep, tt.rss)
			tt.assert(t, got)
		})
	}
}

func newDeployment(namespace, name string, uid types.UID) *appsv1.Deployment {
	one := int32(3)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, UID: uid, Generation: 2},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 3, UpdatedReplicas: 3, AvailableReplicas: 3},
	}
}

func newReplicaSet(namespace, name string, ownerUID types.UID, labels map[string]string, created time.Time) appsv1.ReplicaSet {
	controller := true
	return appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         namespace,
			Name:              name,
			CreationTimestamp: metav1.NewTime(created),
			OwnerReferences:   []metav1.OwnerReference{{UID: ownerUID, Controller: &controller}},
		},
		Spec: appsv1.ReplicaSetSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}}},
	}
}
