package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/corewire/kick/internal/gitops"
	fluxprovider "github.com/corewire/kick/internal/gitops/flux"
	kargoprovider "github.com/corewire/kick/internal/gitops/kargo"
	"github.com/corewire/kick/internal/integrations"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// failingProvider fails ownership resolution with a fixed error.
type failingProvider struct {
	name string
	err  error
}

func (p failingProvider) Name() string { return p.name }

func (p failingProvider) Detect(client.Object) gitops.DetectionResult {
	return gitops.DetectionResult{}
}

func (p failingProvider) ResolveOwner(context.Context, client.Object) (gitops.Owner, error) {
	return gitops.Owner{}, p.err
}

func (p failingProvider) EvaluateGate(context.Context, gitops.Owner, time.Time) (gitops.GateDecision, error) {
	return gitops.GateDecision{}, nil
}

// Every provider defines its own resolution error type, so the resolver must
// read the reason through the shared interface instead of one concrete type.
func TestResolveOwnerAndGateReportsProviderResolutionReason(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		err      error
		want     gitops.GateReason
	}{
		{
			name:     "kargo ambiguous owner",
			provider: "Kargo",
			err:      kargoprovider.ResolutionError{Reason: gitops.GateAmbiguousOwner, Message: "application authorizes more than one kargo stage"},
			want:     gitops.GateAmbiguousOwner,
		},
		{
			name:     "kargo provider unavailable",
			provider: "Kargo",
			err:      kargoprovider.ResolutionError{Reason: gitops.GateProviderUnavailable, Message: "argocd resolver not configured"},
			want:     gitops.GateProviderUnavailable,
		},
		{
			name:     "flux owner unknown",
			provider: "Flux",
			err:      fluxprovider.ResolutionError{Reason: gitops.GateOwnerUnknown, Message: "flux ownership labels missing"},
			want:     gitops.GateOwnerUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &RegistryGateResolver{
				Registry: gitops.NewRegistry(failingProvider{name: tt.provider, err: tt.err}),
			}
			workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"}}

			_, decision, err := resolver.ResolveOwnerAndGate(context.Background(), workload, tt.provider, time.Now())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Reason != tt.want {
				t.Fatalf("reason = %q, want %q", decision.Reason, tt.want)
			}
			if decision.Message != tt.err.Error() {
				t.Fatalf("message = %q, want %q", decision.Message, tt.err.Error())
			}
		})
	}
}

// A request blocked by a switched-off integration is only actionable if it says
// which switch to flip.
func TestResolveOwnerAndGateExplainsUnavailableProvider(t *testing.T) {
	registry := gitops.NewRegistry()
	registry.MarkUnavailable("kargo", gitops.Unavailability{
		Reason:  integrations.ReasonDisabled,
		Message: integrations.Kargo.DisabledMessage(),
	})
	resolver := &RegistryGateResolver{Registry: registry}
	workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"}}

	_, decision, err := resolver.ResolveOwnerAndGate(context.Background(), workload, "kargo", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Reason != gitops.GateProviderUnavailable {
		t.Fatalf("reason = %q, want %q", decision.Reason, gitops.GateProviderUnavailable)
	}
	for _, want := range []string{"spec.gitOps.provider", "--enable-kargo", "integrations.kargo.enabled"} {
		if !strings.Contains(decision.Message, want) {
			t.Fatalf("message %q does not mention %q", decision.Message, want)
		}
	}
}

// A provider name KICK has no adapter for must not be reported as a
// configuration the operator could be restarted into.
func TestResolveOwnerAndGateRejectsUnknownProvider(t *testing.T) {
	resolver := &RegistryGateResolver{Registry: gitops.NewRegistry()}
	workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"}}

	_, decision, err := resolver.ResolveOwnerAndGate(context.Background(), workload, "spinnaker", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(decision.Message, "does not implement") {
		t.Fatalf("message = %q", decision.Message)
	}
}
