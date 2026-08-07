package policy

import (
	"context"
	"testing"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMatchDeploymentNoPolicy(t *testing.T) {
	matcher := newMatcher(t)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"}}

	result, err := matcher.MatchDeployment(context.Background(), dep)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if result.Managed {
		t.Fatalf("expected unmanaged without policy")
	}
	if result.Reason != ReasonPolicyDeleted {
		t.Fatalf("reason = %s, want %s", result.Reason, ReasonPolicyDeleted)
	}
}

func TestMatchDeploymentSingleMatch(t *testing.T) {
	policy := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	matcher := newMatcher(t, policy)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments", Labels: map[string]string{"app": "api"}}}

	result, err := matcher.MatchDeployment(context.Background(), dep)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if !result.Managed {
		t.Fatalf("expected managed for one matching policy")
	}
	if result.Policy == nil || result.Policy.Name != "default" {
		t.Fatalf("unexpected selected policy: %#v", result.Policy)
	}
}

func TestMatchDeploymentSelectorMatchAndConflict(t *testing.T) {
	p1 := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"tier": "prod"}},
			},
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	p2 := p1.DeepCopy()
	p2.Name = "p2"

	matcher := newMatcher(t, p1, p2)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments", Labels: map[string]string{"tier": "prod"}}}
	result, err := matcher.MatchDeployment(context.Background(), dep)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if result.Managed {
		t.Fatalf("expected unmanaged for conflicting policies")
	}
	if result.Reason != ReasonConflictingPolicy {
		t.Fatalf("reason = %s, want %s", result.Reason, ReasonConflictingPolicy)
	}
}

func TestMatchDeploymentSelectorExcludes(t *testing.T) {
	policy := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "payments"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"enabled": "true"}},
			},
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	matcher := newMatcher(t, policy)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments", Labels: map[string]string{"enabled": "false"}}}

	result, err := matcher.MatchDeployment(context.Background(), dep)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if result.Managed {
		t.Fatalf("expected unmanaged for non-matching selector")
	}
	if result.Reason != ReasonNoMatchingPolicy {
		t.Fatalf("reason = %s, want %s", result.Reason, ReasonNoMatchingPolicy)
	}
}

func newMatcher(t *testing.T, objects ...runtime.Object) *DeploymentPolicyMatcher {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
	return &DeploymentPolicyMatcher{Client: c}
}
