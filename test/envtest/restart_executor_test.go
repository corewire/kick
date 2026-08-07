package envtest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/executor"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestRestartExecutorEnvtestIdempotentPatch(t *testing.T) {
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
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kickv1alpha1.AddToScheme(scheme)
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()

	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "x"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "x"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
	}
	if err := c.Create(ctx, dep); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	kr := &kickv1alpha1.KickRequest{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"}, Spec: kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}}}
	if err := c.Create(ctx, kr); err != nil {
		t.Fatalf("create kickrequest: %v", err)
	}

	exec := executor.NewRestartExecutor(c, 5*time.Minute)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	exec.Now = func() time.Time { return now }

	if _, err := exec.Execute(ctx, types.NamespacedName{Namespace: "team-a", Name: "api"}, kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}, types.NamespacedName{Namespace: "team-a", Name: "api"}); err != nil {
		t.Fatalf("execute first: %v", err)
	}
	var got appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	first := got.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	if first == "" {
		t.Fatalf("expected restartedAt annotation")
	}

	if _, err := exec.Execute(ctx, types.NamespacedName{Namespace: "team-a", Name: "api"}, kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}, types.NamespacedName{Namespace: "team-a", Name: "api"}); err != nil {
		t.Fatalf("execute second: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get deployment second: %v", err)
	}
	second := got.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	if first != second {
		t.Fatalf("expected idempotent annotation value; first=%s second=%s", first, second)
	}
}
