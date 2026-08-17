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

// GateReasoner is implemented by provider errors that already know which gate
// reason they must surface. ResolveOwner failures are reported through this
// interface so the caller does not have to know the concrete error type of
// every provider.
type GateReasoner interface {
	error
	GateReason() GateReason
}

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
	providers   []Provider
	unavailable map[string]Unavailability
}

// Unavailability explains why a provider KICK knows about is not registered,
// and how the operator is reconfigured to register it.
type Unavailability struct {
	Reason  string
	Message string
}

func NewRegistry(providers ...Provider) *Registry {
	copyProviders := append([]Provider(nil), providers...)
	return &Registry{providers: copyProviders, unavailable: map[string]Unavailability{}}
}

func (r *Registry) Register(provider Provider) {
	r.providers = append(r.providers, provider)
	delete(r.unavailable, provider.Name())
}

// MarkUnavailable records a provider that exists in KICK but was not wired up.
// Without it a policy naming the provider could only be told that it is
// unknown, which says nothing about how to make it work.
func (r *Registry) MarkUnavailable(name string, unavailability Unavailability) {
	if r.unavailable == nil {
		r.unavailable = map[string]Unavailability{}
	}
	r.unavailable[name] = unavailability
}

// Unavailability returns the recorded explanation for a provider that is not
// registered.
func (r *Registry) Unavailability(name string) (Unavailability, bool) {
	unavailability, ok := r.unavailable[name]
	return unavailability, ok
}

// ProviderByName returns the registered provider with the given name. Explicit
// selection is required for providers that cannot be detected from the workload
// alone.
func (r *Registry) ProviderByName(name string) (Provider, bool) {
	for _, provider := range r.providers {
		if provider.Name() == name {
			return provider, true
		}
	}
	return nil, false
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
