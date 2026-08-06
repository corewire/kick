# KickRequest API and state machine

## Naming

The durable request resource SHOULD be named `KickRequest` to match the product. If compatibility with an existing prototype requires `RestartRequest`, document the migration before release.

## Purpose

A KickRequest is:

- durable wake-up state;
- an audit record;
- a place for conditions and provider decisions;
- a recovery mechanism after controller restart.

It is not the final authority for deciding whether a kick is required. The controller always recomputes from live state.

## Cardinality

There MUST be at most one active KickRequest per target Deployment and policy scope.

Additional dependency changes update the existing request.

## Proposed shape

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickRequest
metadata:
  name: payments-api
  namespace: payments
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: payments-api
status:
  phase: WaitingForGate
  owner:
    provider: argocd
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    namespace: application-team-a
    name: payments-app
    project: production
  gate:
    reason: OutsideSchedule
    requeueAt: "2026-08-07T02:00:00Z"
  latestObservedDependencyChange: "2026-08-06T08:20:00Z"
  currentRollout:
    replicaSet: payments-api-7d4b78f97c
    startedAt: "2026-08-06T07:00:00Z"
  conditions: []
```

## Suggested phases

```text
Pending
WaitingForGate
WaitingForOwner
WaitingForApplicationSync
WaitingForRollout
Executing
Succeeded
NoLongerRequired
Failed
```

Terminal phases:

```text
Succeeded
NoLongerRequired
Failed
```

## State rules

- `requeueAt` is advisory.
- Conditions carry precise reason and message.
- A request may move from a waiting phase to `NoLongerRequired` after live reevaluation.
- `Executing` left behind by a controller crash MUST be safely recoverable.
- Request deletion MUST NOT require a finalizer unless external cleanup is introduced.
- Completed requests SHOULD be garbage-collected through configurable retention.

## Status ownership

The controller owns status. Users MUST NOT need to mutate annotations to approve or progress requests in v1.

## Acceptance criteria

- Repeated dependency events coalesce into one active request.
- Pending requests survive manager restart.
- Stale advisory timers do not cause incorrect execution.
- Terminal requests are retained and cleaned according to configuration.
