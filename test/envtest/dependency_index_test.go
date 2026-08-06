package envtest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func TestDependencyReverseIndexTracksDeploymentChanges(t *testing.T) {
	t.Parallel()

	env := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}
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
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := dependency.RegisterDeploymentReverseIndexes(context.Background(), mgr.GetFieldIndexer()); err != nil {
		t.Fatalf("register indexes: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = mgr.Start(ctx)
	}()

	c := mgr.GetClient()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "demo"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "demo"}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "app",
					Image: "registry.k8s.io/pause:3.10",
					EnvFrom: []corev1.EnvFromSource{{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-one"}},
					}},
				}}},
			},
		},
	}
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	mustEventuallyMatchLookups(t, ctx, c,
		dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "team-a", Name: "secret-one"},
		[]string{"team-a/app"},
	)

	deployment.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{{
		SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "secret-two"}},
	}}
	if err := c.Update(ctx, deployment); err != nil {
		t.Fatalf("update deployment: %v", err)
	}

	mustEventuallyMatchLookups(t, ctx, c,
		dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "team-a", Name: "secret-one"},
		[]string{},
	)
	mustEventuallyMatchLookups(t, ctx, c,
		dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "team-a", Name: "secret-two"},
		[]string{"team-a/app"},
	)

	if err := c.Delete(ctx, deployment); err != nil {
		t.Fatalf("delete deployment: %v", err)
	}
	mustEventuallyMatchLookups(t, ctx, c,
		dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "team-a", Name: "secret-two"},
		[]string{},
	)
}

func TestDependencyReverseIndexIncludesOptionalMissingRefs(t *testing.T) {
	t.Parallel()

	env := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}
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
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := dependency.RegisterDeploymentReverseIndexes(context.Background(), mgr.GetFieldIndexer()); err != nil {
		t.Fatalf("register indexes: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = mgr.Start(ctx)
	}()

	c := mgr.GetClient()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	optional := true
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "optional", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "opt"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "opt"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "registry.k8s.io/pause:3.10",
						Env: []corev1.EnvVar{{
							Name: "TOKEN",
							ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "missing-secret"},
								Key:                  "token",
								Optional:             &optional,
							}},
						}},
					}},
				},
			},
		},
	}
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	mustEventuallyMatchLookups(t, ctx, c,
		dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "team-a", Name: "missing-secret"},
		[]string{"team-a/optional"},
	)
}

func mustEventuallyMatchLookups(t *testing.T, ctx context.Context, c client.Client, ref dependency.DependencyRef, want []string) {
	t.Helper()
	if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 8*time.Second, true, func(context.Context) (bool, error) {
		got, err := dependency.LookupConsumingDeployments(ctx, c, ref)
		if err != nil {
			return false, err
		}
		gotKeys := make([]string, 0, len(got))
		for _, key := range got {
			gotKeys = append(gotKeys, key.Namespace+"/"+key.Name)
		}
		return reflect.DeepEqual(gotKeys, want), nil
	}); err != nil {
		t.Fatalf("lookup mismatch for %+v: %v", ref, err)
	}
}
