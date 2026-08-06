package envtest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestLiveRolloutInspectorEnvtest(t *testing.T) {
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
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-r"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	one := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-r", Name: "api", Generation: 1},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
	}
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-r", Name: "api"}, deployment); err != nil {
		t.Fatalf("get deployment: %v", err)
	}

	controller := true
	current := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "team-r",
			Name:              "api-current",
			CreationTimestamp: metav1.NewTime(time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)),
			OwnerReferences:   []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: deployment.UID, Controller: &controller}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api", "pod-template-hash": "abc"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api", "pod-template-hash": "abc"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
	}
	newestWrong := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "team-r",
			Name:              "api-newest-wrong",
			CreationTimestamp: metav1.NewTime(time.Date(2026, 8, 6, 11, 0, 0, 0, time.UTC)),
			OwnerReferences:   []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: "api", UID: deployment.UID, Controller: &controller}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "other"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "other"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
	}
	if err := c.Create(ctx, current); err != nil {
		t.Fatalf("create current rs: %v", err)
	}
	if err := c.Create(ctx, newestWrong); err != nil {
		t.Fatalf("create wrong rs: %v", err)
	}

	inspector := &rollout.LiveRolloutInspector{Client: c}
	got, err := inspector.Inspect(ctx, deployment)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if got.CurrentReplicaSet.Name != "api-current" {
		t.Fatalf("selected %s; expected api-current", got.CurrentReplicaSet.Name)
	}
}
