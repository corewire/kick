package gitops

import (
	"context"
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
	GateAllowed            GateReason = "Allowed"
	GateOutsideSchedule    GateReason = "OutsideSchedule"
	GateOwnerOutOfSync     GateReason = "OwnerOutOfSync"
	GateOwnerReconciling   GateReason = "OwnerReconciling"
	GateOwnerUnknown       GateReason = "OwnerUnknown"
	GateAmbiguousOwner     GateReason = "AmbiguousOwner"
	GateConfigurationError GateReason = "ConfigurationError"
)

type GateDecision struct {
	Allowed     bool
	Reconciled  bool
	Reconciling bool
	RequeueAt   *time.Time
	Reason      GateReason
	Message     string
}

// Provider isolates GitOps-specific ownership and scheduling semantics from
// KICK's provider-neutral freshness and rollout logic.
type Provider interface {
	Name() string
	ResolveOwner(context.Context, client.Object) (Owner, error)
	EvaluateGate(context.Context, Owner, time.Time) (GateDecision, error)
}
