package rollout

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newRollout(name string, replicas int64, status map[string]any) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Rollout",
		"metadata": map[string]any{
			"name":              name,
			"namespace":         "default",
			"uid":               "rollout-uid",
			"creationTimestamp": "2024-01-01T00:00:00Z",
		},
		"spec":   map[string]any{"replicas": replicas},
		"status": status,
	}}
	return obj
}

func fakeClientWithReplicaSets(rs ...appsv1.ReplicaSet) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for i := range rs {
		builder = builder.WithObjects(&rs[i])
	}
	return builder
}

func TestInspectArgoRolloutHealthyIsComplete(t *testing.T) {
	obj := newRollout("web", 2, map[string]any{
		"phase":             "Healthy",
		"updatedReplicas":   int64(2),
		"availableReplicas": int64(2),
	})
	c := fakeClientWithReplicaSets().Build()

	state, err := inspectArgoRollout(context.Background(), c, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Complete || state.InProgress || state.Failed {
		t.Fatalf("expected complete state, got %+v", state)
	}
}

func TestInspectArgoRolloutProgressingIsInProgress(t *testing.T) {
	obj := newRollout("web", 3, map[string]any{
		"phase":             "Progressing",
		"updatedReplicas":   int64(1),
		"availableReplicas": int64(1),
	})
	c := fakeClientWithReplicaSets().Build()

	state, err := inspectArgoRollout(context.Background(), c, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.InProgress || state.Complete {
		t.Fatalf("expected in-progress state, got %+v", state)
	}
	if state.Reason != ReasonRolloutInProgress {
		t.Fatalf("expected reason %q, got %q", ReasonRolloutInProgress, state.Reason)
	}
}

func TestInspectArgoRolloutDegradedIsFailed(t *testing.T) {
	obj := newRollout("web", 1, map[string]any{"phase": "Degraded"})
	c := fakeClientWithReplicaSets().Build()

	state, err := inspectArgoRollout(context.Background(), c, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Failed || state.Complete {
		t.Fatalf("expected failed state, got %+v", state)
	}
	if state.Reason != ReasonArgoRolloutDegraded {
		t.Fatalf("expected reason %q, got %q", ReasonArgoRolloutDegraded, state.Reason)
	}
}

// A restart recycles pods without creating a ReplicaSet, so status.restartedAt
// must win over the newest owned ReplicaSet.
func TestArgoRolloutStartedAtPrefersRestartedAt(t *testing.T) {
	rsCreated := metav1.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	controller := true
	rs := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-abc",
			Namespace:         "default",
			CreationTimestamp: rsCreated,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Rollout",
				Name:       "web",
				UID:        types.UID("rollout-uid"),
				Controller: &controller,
			}},
		},
	}
	obj := newRollout("web", 1, map[string]any{
		"phase":             "Healthy",
		"restartedAt":       "2024-01-03T00:00:00Z",
		"updatedReplicas":   int64(1),
		"availableReplicas": int64(1),
	})
	c := fakeClientWithReplicaSets(rs).Build()

	startedAt, err := argoRolloutStartedAt(context.Background(), c, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	if !startedAt.Equal(want) {
		t.Fatalf("expected %s, got %s", want, startedAt)
	}
}

// A ReplicaSet owned by a different Rollout must not move the start time.
func TestArgoRolloutStartedAtIgnoresForeignReplicaSets(t *testing.T) {
	controller := true
	rs := appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "other-abc",
			Namespace:         "default",
			CreationTimestamp: metav1.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Rollout",
				Name:       "other",
				UID:        types.UID("other-uid"),
				Controller: &controller,
			}},
		},
	}
	obj := newRollout("web", 1, map[string]any{"phase": "Healthy"})
	c := fakeClientWithReplicaSets(rs).Build()

	startedAt, err := argoRolloutStartedAt(context.Background(), c, obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !startedAt.Equal(want) {
		t.Fatalf("expected %s, got %s", want, startedAt)
	}
}

func TestArgoRolloutComplete(t *testing.T) {
	cases := []struct {
		name string
		obj  *unstructured.Unstructured
		want bool
	}{
		{
			name: "healthy and fully available",
			obj:  newRollout("web", 2, map[string]any{"phase": "Healthy", "updatedReplicas": int64(2), "availableReplicas": int64(2)}),
			want: true,
		},
		{
			name: "healthy but not all updated",
			obj:  newRollout("web", 2, map[string]any{"phase": "Healthy", "updatedReplicas": int64(1), "availableReplicas": int64(2)}),
			want: false,
		},
		{
			name: "paused",
			obj:  newRollout("web", 2, map[string]any{"phase": "Paused", "updatedReplicas": int64(2), "availableReplicas": int64(2)}),
			want: false,
		},
		{
			name: "nil",
			obj:  nil,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ArgoRolloutComplete(tc.obj); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
