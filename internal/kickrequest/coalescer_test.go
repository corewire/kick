package kickrequest

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureActiveRequestCoalescesRepeatedEvents(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).Build()
	coalescer := NewCoalescer(client, RetentionConfig{})

	ctx := context.Background()
	target := kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "payments-api"}
	firstAt := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(5 * time.Minute)

	if _, err := coalescer.EnsureActiveRequest(ctx, "payments", target, firstAt); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if _, err := coalescer.EnsureActiveRequest(ctx, "payments", target, secondAt); err != nil {
		t.Fatalf("second ensure: %v", err)
	}

	var list kickv1alpha1.KickRequestList
	if err := client.List(ctx, &list); err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("request count = %d, want 1", len(list.Items))
	}
	item := list.Items[0]
	if item.Status.Phase != kickv1alpha1.KickRequestPhasePending {
		t.Fatalf("phase = %s, want %s", item.Status.Phase, kickv1alpha1.KickRequestPhasePending)
	}
	if item.Status.LatestObservedDependencyChange == nil || !item.Status.LatestObservedDependencyChange.Time.Equal(secondAt) {
		t.Fatalf("latestObservedDependencyChange = %#v, want %s", item.Status.LatestObservedDependencyChange, secondAt)
	}
}

func TestEnsureActiveRequestReopensTerminal(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	seeded := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}},
		Status:     kickv1alpha1.KickRequestStatus{Phase: kickv1alpha1.KickRequestPhaseSucceeded},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(seeded).Build()
	coalescer := NewCoalescer(client, RetentionConfig{})

	now := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	if _, err := coalescer.EnsureActiveRequest(context.Background(), "team-a", kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}, now); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var got kickv1alpha1.KickRequest
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhasePending {
		t.Fatalf("phase = %s, want pending", got.Status.Phase)
	}
}

func TestEnsureActiveRequestNamesStatefulSetRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).Build()
	coalescer := NewCoalescer(client, RetentionConfig{})

	_, err := coalescer.EnsureActiveRequest(context.Background(), "team-a", kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "StatefulSet", Name: "db"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var got kickv1alpha1.KickRequest
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "statefulset-db"}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Spec.TargetRef.Kind != "StatefulSet" {
		t.Fatalf("target kind = %s", got.Spec.TargetRef.Kind)
	}
}
