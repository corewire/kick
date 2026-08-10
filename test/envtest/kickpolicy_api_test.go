package envtest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payments-api"}},
			},
			GitOps:  kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
			Restart: kickv1alpha1.KickPolicyRestartSpec{MinInterval: "45s"},
		},
	}
	if err := c.Create(ctx, valid); err != nil {
		t.Fatalf("create valid kickpolicy: %v", err)
	}
	var persisted kickv1alpha1.KickPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(valid), &persisted); err != nil {
		t.Fatalf("get valid kickpolicy: %v", err)
	}
	if persisted.Spec.Discovery.WorkloadSelector == nil || persisted.Spec.Discovery.WorkloadSelector.MatchLabels["app"] != "payments-api" {
		t.Fatalf("unexpected workloadSelector: %#v", persisted.Spec.Discovery.WorkloadSelector)
	}
	if persisted.Spec.GitOps.Provider != kickv1alpha1.KickPolicyProviderAuto {
		t.Fatalf("unexpected provider: %s", persisted.Spec.GitOps.Provider)
	}
	if persisted.Spec.GitOps.RequireReconciled == nil || !*persisted.Spec.GitOps.RequireReconciled {
		t.Fatalf("requireReconciled default not applied: %#v", persisted.Spec.GitOps.RequireReconciled)
	}
	if persisted.Spec.Restart.MinInterval != "45s" {
		t.Fatalf("unexpected minInterval: %s", persisted.Spec.Restart.MinInterval)
	}

	// A policy without a gitOps block is accepted; provider defaults to None.
	noGitOps := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "no-gitops", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "x"}},
			},
		},
	}
	if err := c.Create(ctx, noGitOps); err != nil {
		t.Fatalf("expected gitOps-less kickpolicy to be accepted: %v", err)
	}
	var persistedNoGitOps kickv1alpha1.KickPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(noGitOps), &persistedNoGitOps); err != nil {
		t.Fatalf("get gitOps-less kickpolicy: %v", err)
	}
	if persistedNoGitOps.Spec.GitOps.Provider != kickv1alpha1.KickPolicyProviderNone {
		t.Fatalf("provider default not applied: %q", persistedNoGitOps.Spec.GitOps.Provider)
	}

	// A dependencySelector is accepted and round-trips.
	withDepSelector := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "dep-selector", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector:   &metav1.LabelSelector{},
				DependencySelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kick.corewire.io/watch": "true"}},
			},
		},
	}
	if err := c.Create(ctx, withDepSelector); err != nil {
		t.Fatalf("expected dependencySelector kickpolicy to be accepted: %v", err)
	}
	var persistedDep kickv1alpha1.KickPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(withDepSelector), &persistedDep); err != nil {
		t.Fatalf("get dependencySelector kickpolicy: %v", err)
	}
	if persistedDep.Spec.Discovery.DependencySelector == nil ||
		persistedDep.Spec.Discovery.DependencySelector.MatchLabels["kick.corewire.io/watch"] != "true" {
		t.Fatalf("dependencySelector not persisted: %#v", persistedDep.Spec.Discovery.DependencySelector)
	}

	// An explicit empty workloadSelector opts in to all workloads and is valid.
	explicitAll := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "explicit-all", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{WorkloadSelector: &metav1.LabelSelector{}},
		},
	}
	if err := c.Create(ctx, explicitAll); err != nil {
		t.Fatalf("expected explicit empty workloadSelector to be accepted: %v", err)
	}

	// A policy that omits workloadSelector is rejected: wide scope is never accidental.
	missingSelector := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "missing-selector", Namespace: "payments"},
		Spec:       kickv1alpha1.KickPolicySpec{},
	}
	if err := c.Create(ctx, missingSelector); err == nil {
		t.Fatalf("expected policy without workloadSelector to fail validation")
	}

	invalidProvider := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "badprovider", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{WorkloadSelector: &metav1.LabelSelector{}},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: "Unknown"},
		},
	}
	if err := c.Create(ctx, invalidProvider); err == nil {
		t.Fatalf("expected invalid provider to fail validation")
	}

	invalidInterval := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "badinterval", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{WorkloadSelector: &metav1.LabelSelector{}},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
			Restart:   kickv1alpha1.KickPolicyRestartSpec{MinInterval: "-5s"},
		},
	}
	if err := c.Create(ctx, invalidInterval); err == nil {
		t.Fatalf("expected negative minInterval to fail validation")
	}
}

func TestKickPolicyStatusRoundTripEnvtest(t *testing.T) {
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
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-status"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	policy := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "team-status"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{WorkloadSelector: &metav1.LabelSelector{}},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	if err := c.Create(ctx, policy); err != nil {
		t.Fatalf("create kickpolicy: %v", err)
	}

	var got kickv1alpha1.KickPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(policy), &got); err != nil {
		t.Fatalf("get kickpolicy before status update: %v", err)
	}
	got.Status = kickv1alpha1.KickPolicyStatus{
		ObservedGeneration: 7,
		MatchedWorkloads:   11,
		BlockedWorkloads:   3,
		Conditions:         []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Indexed", Message: "policy applied", LastTransitionTime: metav1.NewTime(time.Date(2026, 8, 6, 12, 5, 0, 0, time.UTC))}},
	}
	if err := c.Status().Update(ctx, &got); err != nil {
		t.Fatalf("update kickpolicy status: %v", err)
	}

	var updated kickv1alpha1.KickPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(policy), &updated); err != nil {
		t.Fatalf("get kickpolicy after status update: %v", err)
	}
	if updated.Status.ObservedGeneration != 7 || updated.Status.MatchedWorkloads != 11 || updated.Status.BlockedWorkloads != 3 {
		t.Fatalf("unexpected status counters: %#v", updated.Status)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Type != "Ready" || updated.Status.Conditions[0].Reason != "Indexed" {
		t.Fatalf("unexpected kickpolicy conditions: %#v", updated.Status.Conditions)
	}
}
