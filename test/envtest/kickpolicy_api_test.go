package envtest

import (
	"context"
	"path/filepath"
	"testing"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestKickPolicyValidationEnvtest(t *testing.T) {
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
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	valid := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	if err := c.Create(ctx, valid); err != nil {
		t.Fatalf("create valid kickpolicy: %v", err)
	}

	invalidMissing := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "missing", Namespace: "payments"},
		Spec:       kickv1alpha1.KickPolicySpec{},
	}
	if err := c.Create(ctx, invalidMissing); err == nil {
		t.Fatalf("expected missing required fields to fail validation")
	}

	invalidMode := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "badmode", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: "Full"},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	if err := c.Create(ctx, invalidMode); err == nil {
		t.Fatalf("expected invalid discovery mode to fail validation")
	}

	invalidProvider := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "badprovider", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: "Unknown"},
		},
	}
	if err := c.Create(ctx, invalidProvider); err == nil {
		t.Fatalf("expected invalid provider to fail validation")
	}

	invalidInterval := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "badinterval", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery:   kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:      kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
			MinInterval: "-5s",
		},
	}
	if err := c.Create(ctx, invalidInterval); err == nil {
		t.Fatalf("expected negative minInterval to fail validation")
	}
}
