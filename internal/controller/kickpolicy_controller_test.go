package controller

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/gitops"
	argocdprovider "github.com/corewire/kick/internal/gitops/argocd"
	"github.com/corewire/kick/internal/integrations"
)

func kickPolicyScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	return scheme
}

// A policy that names a provider the operator was not started with must say so
// on the policy itself, and must name the flag and the Helm value that fix it.
func TestReconcileReportsUnavailableProvider(t *testing.T) {
	tests := []struct {
		name         string
		provider     kickv1alpha1.KickPolicyProvider
		unavailable  *gitops.Unavailability
		wantStatus   metav1.ConditionStatus
		wantReason   string
		wantContains []string
	}{
		{
			name:       "registered provider is accepted",
			provider:   kickv1alpha1.KickPolicyProviderFlux,
			wantStatus: metav1.ConditionTrue,
			wantReason: kickPolicyReasonAccepted,
		},
		{
			name:       "provider without a gate is accepted",
			provider:   kickv1alpha1.KickPolicyProviderNone,
			wantStatus: metav1.ConditionTrue,
			wantReason: kickPolicyReasonAccepted,
		},
		{
			name:     "disabled integration names both switches",
			provider: kickv1alpha1.KickPolicyProviderKargo,
			unavailable: &gitops.Unavailability{
				Reason:  integrations.ReasonDisabled,
				Message: integrations.Kargo.DisabledMessage(),
			},
			wantStatus:   metav1.ConditionFalse,
			wantReason:   integrations.ReasonDisabled,
			wantContains: []string{"--enable-kargo", "integrations.kargo.enabled"},
		},
		{
			name:     "missing CRD is distinguished from a disabled integration",
			provider: kickv1alpha1.KickPolicyProviderArgoCD,
			unavailable: &gitops.Unavailability{
				Reason:  integrations.ReasonKindNotInstalled,
				Message: integrations.ArgoCD.KindNotInstalledMessage(argocdprovider.ApplicationGVK),
			},
			wantStatus:   metav1.ConditionFalse,
			wantReason:   integrations.ReasonKindNotInstalled,
			wantContains: []string{"Application", "install its CRDs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &kickv1alpha1.KickPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "policy", Namespace: "ns", Generation: 3},
				Spec:       kickv1alpha1.KickPolicySpec{GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: tt.provider}},
			}
			scheme := kickPolicyScheme(t)
			registry := gitops.NewRegistry(failingProvider{name: "flux"})
			if tt.unavailable != nil {
				registry.MarkUnavailable(gateProviderName(policy), *tt.unavailable)
			}
			recorder := record.NewFakeRecorder(10)
			reconciler := &KickPolicyReconciler{
				Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(policy).WithStatusSubresource(policy).Build(),
				Scheme:   scheme,
				Recorder: recorder,
				Registry: registry,
			}

			key := types.NamespacedName{Name: "policy", Namespace: "ns"}
			if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
				t.Fatalf("reconcile: %v", err)
			}

			var reconciled kickv1alpha1.KickPolicy
			if err := reconciler.Get(context.Background(), key, &reconciled); err != nil {
				t.Fatalf("get policy: %v", err)
			}
			if reconciled.Status.ObservedGeneration != 3 {
				t.Fatalf("observedGeneration = %d, want 3", reconciled.Status.ObservedGeneration)
			}
			condition := findCondition(reconciled.Status.Conditions, KickPolicyReadyCondition)
			if condition == nil {
				t.Fatal("Ready condition not set")
			}
			if condition.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", condition.Status, tt.wantStatus)
			}
			if condition.Reason != tt.wantReason {
				t.Fatalf("reason = %s, want %s", condition.Reason, tt.wantReason)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(condition.Message, want) {
					t.Fatalf("message %q does not mention %q", condition.Message, want)
				}
			}
			if tt.wantStatus == metav1.ConditionFalse {
				select {
				case event := <-recorder.Events:
					if !strings.Contains(event, tt.wantReason) {
						t.Fatalf("event %q does not carry reason %s", event, tt.wantReason)
					}
				default:
					t.Fatal("no warning event recorded")
				}
			}
		})
	}
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
