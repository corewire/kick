package gitops

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ContractInput configures reusable provider contract checks for adapters.
type ContractInput struct {
	Provider Provider
	Workload client.Object
	Now      time.Time
}

// RunProviderContract validates deterministic core behavior expected from all providers.
func RunProviderContract(t *testing.T, in ContractInput) {
	t.Helper()

	if in.Provider == nil {
		t.Fatalf("provider must not be nil")
	}
	if in.Provider.Name() == "" {
		t.Fatalf("provider name must not be empty")
	}

	first := in.Provider.Detect(in.Workload)
	second := in.Provider.Detect(in.Workload)
	if first != second {
		t.Fatalf("detect must be deterministic: first=%+v second=%+v", first, second)
	}
	if !first.Confident {
		return
	}

	owner, err := in.Provider.ResolveOwner(context.Background(), in.Workload)
	if err != nil {
		t.Fatalf("resolve owner: %v", err)
	}
	if owner.Provider == "" {
		owner.Provider = in.Provider.Name()
	}
	decision, err := in.Provider.EvaluateGate(context.Background(), owner, in.Now)
	if err != nil {
		t.Fatalf("evaluate gate: %v", err)
	}
	if decision.Reason == "" {
		t.Fatalf("gate reason must not be empty")
	}
	if decision.RequeueAt != nil {
		_ = decision.RequeueAt // advisory only; presence is allowed in any state.
	}
}
