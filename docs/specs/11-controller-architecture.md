# Controller architecture

KICK uses a plain Kubebuilder project and `controller-runtime` directly. The architecture MUST remain recognizable to maintainers familiar with upstream Kubebuilder controllers.

## Goal

Separate graph discovery, source observation, provider indexing, and request execution into small controllers with clear ownership.

## Recommended controllers

### DependencyIndexController

Responsibilities:

- watch Deployments;
- extract current dependencies;
- maintain controller-runtime indexes;
- enqueue affected request reconciliation when references change.

### SourceObservationController

Responsibilities:

- watch Secrets and ConfigMaps;
- distinguish relevant content changes from metadata-only changes;
- persist observations;
- use reverse indexes to create or update KickRequests.

### GitOpsProviderIndexController

Responsibilities:

- provider-specific watches and indexes;
- Application-resource reverse index for Argo CD fallback;
- re-enqueue requests on owner, status, project, or window changes.

### KickRequestController

Responsibilities:

- resolve owner;
- evaluate provider gate;
- inspect rollout;
- evaluate freshness;
- execute kick;
- observe completion;
- update phase and conditions.

## Shared services

Use interfaces for:

```text
DependencyExtractor
ChangeObserver
ObservationStore
RolloutInspector
FreshnessEvaluator
GitOpsProviderRegistry
RestartExecutor
Clock
```

## Recovery

At manager startup, non-terminal KickRequests MUST be reconciled from Kubernetes state. Workqueue delays are hints only.

Requests with past `requeueAt`, `Executing`, or stale status MUST be re-evaluated safely.

## Leader election

Production deployments MUST support controller-runtime leader election. Reconciliation and request creation must remain idempotent during leader transitions.

## Cache and indexing

- Prefer informer caches and field indexes over API-wide scans.
- Provider fallback indexes must be updated from watches.
- Avoid unbounded in-memory maps not reconstructable after restart.

## Acceptance criteria

- Controllers can be unit tested through injected interfaces.
- Manager restart preserves pending work.
- Two controller replicas with leader election do not duplicate kicks.
- Dependency and Application indexes update on object changes.
