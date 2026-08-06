package gitops

import (
	"context"
	"fmt"
	"sort"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Owner struct {
	Provider   string
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	Project    string
}

type GateReason string

const (
	GateAllowed             GateReason = "Allowed"
	GateOutsideSchedule     GateReason = "OutsideSchedule"
	GateOwnerOutOfSync      GateReason = "OwnerOutOfSync"
	GateOwnerReconciling    GateReason = "OwnerReconciling"
	GateOwnerUnknown        GateReason = "OwnerUnknown"
	GateAmbiguousOwner      GateReason = "AmbiguousOwner"
	GateProjectUnknown      GateReason = "ProjectUnknown"
	GateProviderUnavailable GateReason = "ProviderUnavailable"
	GateConfigurationError  GateReason = "ConfigurationError"
)

type DetectionResult struct {
	Confident bool
	Message   string
}

type GateDecision struct {
	Allowed     bool
	Reconciled  bool
	Reconciling bool
	RequeueAt   *time.Time
	Reason      GateReason
	Message     string
}

// MayExecute is the provider-neutral core gate predicate.
func MayExecute(decision GateDecision) bool {
	return decision.Allowed && decision.Reconciled && !decision.Reconciling
}

// Provider isolates GitOps-specific ownership and scheduling semantics from
// KICK's provider-neutral freshness and rollout logic.
type Provider interface {
	Name() string
	Detect(client.Object) DetectionResult
	ResolveOwner(context.Context, client.Object) (Owner, error)
	EvaluateGate(context.Context, Owner, time.Time) (GateDecision, error)
}

// Registry arbitrates enabled providers with deterministic zero/one/many outcomes.
type Registry struct {
	providers []Provider
}

func NewRegistry(providers ...Provider) *Registry {
	copyProviders := append([]Provider(nil), providers...)
	return &Registry{providers: copyProviders}
}

func (r *Registry) Register(provider Provider) {
	r.providers = append(r.providers, provider)
}

func (r *Registry) DetectProvider(workload client.Object) (Provider, GateDecision) {
	confident := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		if provider.Detect(workload).Confident {
			confident = append(confident, provider)
		}
	}

	if len(confident) == 0 {
		return nil, GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: GateOwnerUnknown, Message: "no provider detected ownership signals"}
	}
	if len(confident) > 1 {
		names := make([]string, 0, len(confident))
		for _, provider := range confident {
			names = append(names, provider.Name())
		}
		sort.Strings(names)
		return nil, GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: GateAmbiguousOwner, Message: fmt.Sprintf("multiple providers detected ownership: %v", names)}
	}

	return confident[0], GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: GateAllowed, Message: "exactly one provider detected"}
}
