package executor

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExecutePatchesOnceAndIsIdempotent(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kickv1alpha1.AddToScheme(scheme)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec:       appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "x"}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "x"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}}},
	}
	req := &kickv1alpha1.KickRequest{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"}, Spec: kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	exec := NewRestartExecutor(c, 5*time.Minute)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	exec.Now = func() time.Time { return now }

	res1, err := exec.Execute(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, types.NamespacedName{Namespace: "team-a", Name: "api"})
	if err != nil {
		t.Fatalf("execute first: %v", err)
	}
	if !res1.Patched {
		t.Fatalf("expected first execution to patch")
	}

	res2, err := exec.Execute(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, types.NamespacedName{Namespace: "team-a", Name: "api"})
	if err != nil {
		t.Fatalf("execute second: %v", err)
	}
	if res2.Patched {
		t.Fatalf("expected second execution to be idempotent without new patch")
	}
}

func TestExecuteTimeoutMarksFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kickv1alpha1.AddToScheme(scheme)

	started := metav1.NewTime(time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC))
	req := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}},
		Status: kickv1alpha1.KickRequestStatus{
			Phase:          kickv1alpha1.KickRequestPhaseExecuting,
			CurrentRollout: kickv1alpha1.RolloutStatus{StartedAt: &started},
		},
	}
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(req, dep).Build()
	exec := NewRestartExecutor(c, time.Minute)
	exec.Now = func() time.Time { return started.Add(2 * time.Minute) }

	res, err := exec.Execute(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, types.NamespacedName{Namespace: "team-a", Name: "api"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !res.Failed || res.Reason != "RolloutTimeout" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
