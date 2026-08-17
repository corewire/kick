package envtest

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/controller"
	"github.com/corewire/kick/internal/gitops"
	"github.com/corewire/kick/internal/integrations"
)

// kargoStub stands in for the Kargo provider: the policy validator only asks
// the registry whether the provider is there.
type kargoStub struct{}

func (kargoStub) Name() string { return integrations.Kargo.Name }

func (kargoStub) Detect(client.Object) gitops.DetectionResult { return gitops.DetectionResult{} }

func (kargoStub) ResolveOwner(context.Context, client.Object) (gitops.Owner, error) {
	return gitops.Owner{}, nil
}

func (kargoStub) EvaluateGate(context.Context, gitops.Owner, time.Time) (gitops.GateDecision, error) {
	return gitops.GateDecision{}, nil
}

// A policy that selects a GitOps provider the operator was not started with
// would otherwise look healthy until a dependency happened to change.
func TestKickPolicyReportsUnavailableIntegrationEnvtest(t *testing.T) {
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
	const namespace = "kickpolicy-integration"
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	policy := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "kargo-gated", Namespace: namespace},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payments-api"}},
			},
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderKargo},
		},
	}
	if err := c.Create(ctx, policy); err != nil {
		t.Fatalf("create policy: %v", err)
	}

	registry := gitops.NewRegistry()
	registry.MarkUnavailable(integrations.Kargo.Name, gitops.Unavailability{
		Reason:  integrations.ReasonDisabled,
		Message: integrations.Kargo.DisabledMessage(),
	})
	reconciler := &controller.KickPolicyReconciler{Client: c, Scheme: scheme, Registry: registry}
	key := types.NamespacedName{Namespace: namespace, Name: "kargo-gated"}

	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var reconciled kickv1alpha1.KickPolicy
	if err := c.Get(ctx, key, &reconciled); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	ready := apimeta.FindStatusCondition(reconciled.Status.Conditions, controller.KickPolicyReadyCondition)
	if ready == nil {
		t.Fatal("Ready condition not persisted")
	}
	if ready.Status != metav1.ConditionFalse || ready.Reason != integrations.ReasonDisabled {
		t.Fatalf("condition = %s/%s, want False/%s", ready.Status, ready.Reason, integrations.ReasonDisabled)
	}
	for _, want := range []string{integrations.Kargo.Flag, integrations.Kargo.HelmValue} {
		if !strings.Contains(ready.Message, want) {
			t.Fatalf("message %q does not name %q", ready.Message, want)
		}
	}
	if reconciled.Status.ObservedGeneration != reconciled.Generation {
		t.Fatalf("observedGeneration = %d, want %d", reconciled.Status.ObservedGeneration, reconciled.Generation)
	}

	registry.Register(kargoStub{})
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("reconcile after enabling: %v", err)
	}
	if err := c.Get(ctx, key, &reconciled); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	ready = apimeta.FindStatusCondition(reconciled.Status.Conditions, controller.KickPolicyReadyCondition)
	if ready.Status != metav1.ConditionTrue {
		t.Fatalf("policy stayed %s after the integration was enabled: %s", ready.Status, ready.Message)
	}
}
