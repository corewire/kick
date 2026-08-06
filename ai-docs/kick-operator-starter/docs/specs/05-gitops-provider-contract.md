# GitOps provider contract

## Goal

Keep the KICK core independent from Argo CD, Flux, and future GitOps implementations.

## Provider responsibilities

A provider adapter MUST:

- decide whether workload metadata indicates that the provider may own the workload;
- resolve exactly one GitOps owner or report none/ambiguous;
- evaluate whether provider reconciliation is active;
- evaluate whether desired state has been applied;
- evaluate provider-specific windows or schedules;
- return a next reevaluation time when known;
- expose provider-specific details without leaking provider types into the core.

## Conceptual interface

```go
type GitOpsProvider interface {
    Name() string
    Detect(workload client.Object) DetectionResult
    ResolveOwner(ctx context.Context, workload client.Object) (GitOpsOwner, error)
    EvaluateGate(ctx context.Context, owner GitOpsOwner, now time.Time) (GateDecision, error)
}
```

## Generic owner

```go
type GitOpsOwner struct {
    Provider   string
    APIVersion string
    Kind       string
    Namespace  string
    Name       string
    Project    string
}
```

## Generic gate decision

```go
type GateDecision struct {
    Allowed     bool
    Reconciled  bool
    Reconciling bool
    RequeueAt   *time.Time
    Reason      GateReason
    Message     string
}
```

Required reasons:

```text
Allowed
OutsideSchedule
OwnerOutOfSync
OwnerReconciling
OwnerUnknown
AmbiguousOwner
ProjectUnknown
ProviderUnavailable
ConfigurationError
```

## Core interpretation

The core MAY execute only when:

```text
Allowed == true
Reconciled == true
Reconciling == false
```

All other decisions block execution.

## Provider detection

- The core MAY ask every enabled provider to detect ownership signals.
- Exactly one confident provider proceeds.
- No provider results in `OwnerUnknown`.
- Multiple confident providers result in `AmbiguousOwner`.
- The core MUST NOT choose a provider arbitrarily.

## Testing contract

Each provider MUST ship contract tests proving:

- stable owner resolution;
- deterministic missing and ambiguous behavior;
- gate decisions map to generic reasons;
- next reevaluation time is never treated as authoritative;
- provider object updates re-enqueue affected requests.
