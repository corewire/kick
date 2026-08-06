package gitops

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeProvider struct {
	name     string
	detect   DetectionResult
	owner    Owner
	decision GateDecision
}

func (f fakeProvider) Name() string                         { return f.name }
func (f fakeProvider) Detect(client.Object) DetectionResult { return f.detect }
func (f fakeProvider) ResolveOwner(context.Context, client.Object) (Owner, error) {
	return f.owner, nil
}
func (f fakeProvider) EvaluateGate(context.Context, Owner, time.Time) (GateDecision, error) {
	return f.decision, nil
}

func TestRegistryArbitration(t *testing.T) {
	workload := &metav1.PartialObjectMetadata{}

	one := fakeProvider{name: "one", detect: DetectionResult{Confident: true}}
	two := fakeProvider{name: "two", detect: DetectionResult{Confident: true}}
	none := fakeProvider{name: "none", detect: DetectionResult{Confident: false}}

	registry := NewRegistry(none)
	selected, decision := registry.DetectProvider(workload)
	if selected != nil || decision.Reason != GateOwnerUnknown {
		t.Fatalf("zero detection mismatch: provider=%v reason=%s", selected, decision.Reason)
	}

	registry = NewRegistry(one)
	selected, decision = registry.DetectProvider(workload)
	if selected == nil || selected.Name() != "one" || decision.Reason != GateAllowed {
		t.Fatalf("single detection mismatch: provider=%v reason=%s", selected, decision.Reason)
	}

	registry = NewRegistry(two, one)
	selected, decision = registry.DetectProvider(workload)
	if selected != nil || decision.Reason != GateAmbiguousOwner {
		t.Fatalf("ambiguous detection mismatch: provider=%v reason=%s", selected, decision.Reason)
	}
}

func TestFakeProviderPassesContract(t *testing.T) {
	requeueAt := time.Now().Add(time.Minute)
	provider := fakeProvider{
		name:   "fake",
		detect: DetectionResult{Confident: true, Message: "match"},
		owner:  Owner{Provider: "fake", APIVersion: "v1", Kind: "FakeOwner", Namespace: "ns", Name: "x"},
		decision: GateDecision{
			Allowed:     false,
			Reconciled:  false,
			Reconciling: true,
			RequeueAt:   &requeueAt,
			Reason:      GateOwnerReconciling,
			Message:     "sync running",
		},
	}

	RunProviderContract(t, ContractInput{
		Provider: provider,
		Workload: &metav1.PartialObjectMetadata{},
		Now:      time.Now(),
	})
}

func TestMayExecute(t *testing.T) {
	if !MayExecute(GateDecision{Allowed: true, Reconciled: true, Reconciling: false}) {
		t.Fatalf("expected execute when allowed/reconciled/not-reconciling")
	}
	if MayExecute(GateDecision{Allowed: true, Reconciled: false, Reconciling: false}) {
		t.Fatalf("expected block when not reconciled")
	}
}
