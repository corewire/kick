package envtest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/rollout"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestFreshnessEvaluatorWithLiveRolloutInspectorEnvtest(t *testing.T) {
	t.Parallel()

	env := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")}}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("stop envtest: %v", err)
		}
	}()
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "fresh-ns"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	zero := int32(0)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fresh-ns", Name: "api"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &zero,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 0, UpdatedReplicas: 0, AvailableReplicas: 0},
	}
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "fresh-ns", Name: "api"}, deployment); err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	deployment.Status.ObservedGeneration = deployment.Generation
	deployment.Status.Replicas = 0
	deployment.Status.UpdatedReplicas = 0
	deployment.Status.AvailableReplicas = 0
	if err := c.Status().Update(ctx, deployment); err != nil {
		t.Fatalf("update deployment status: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "fresh-ns", Name: "api"}, deployment); err != nil {
		t.Fatalf("refresh deployment after status update: %v", err)
	}

	controller := true
	rsStart := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "fresh-ns",
			Name:              "api-current",
			CreationTimestamp: metav1.NewTime(rsStart),
			OwnerReferences:   []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: deployment.UID, Controller: &controller}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api", "pod-template-hash": "x"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api", "pod-template-hash": "x"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
	}
	if err := c.Create(ctx, rs); err != nil {
		t.Fatalf("create rs: %v", err)
	}

	liveInspector := &rollout.LiveRolloutInspector{Client: c}
	evaluator := &freshness.Evaluator{Inspector: liveInspector}
	state, err := liveInspector.Inspect(ctx, deployment)
	if err != nil {
		t.Fatalf("inspect rollout state: %v", err)
	}
	if state.CurrentReplicaSet.Name == "" {
		t.Fatalf("expected current replicaset to be identified")
	}
	depRef := dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "fresh-ns", Name: "sec"}
	newerThanRollout := state.StartedAt.Add(time.Minute)
	olderThanRollout := state.StartedAt.Add(-time.Minute)
	decision, err := evaluator.Evaluate(ctx, deployment, []dependency.DependencyRef{depRef}, map[dependency.DependencyRef]time.Time{depRef: newerThanRollout})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !decision.RestartRequired {
		t.Fatalf("expected restart required for newer dependency")
	}

	decision2, err := evaluator.Evaluate(ctx, deployment, []dependency.DependencyRef{depRef}, map[dependency.DependencyRef]time.Time{depRef: olderThanRollout})
	if err != nil {
		t.Fatalf("evaluate older: %v", err)
	}
	if decision2.RestartRequired {
		t.Fatalf("expected no restart when dependency older than rollout: decision=%+v", decision2)
	}
}
