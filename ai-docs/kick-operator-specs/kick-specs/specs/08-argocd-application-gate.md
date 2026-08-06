# Argo CD Application gate

## Goal

Prevent KICK from racing with Argo CD and avoid unnecessary double rollouts.

## Gate requirements

The Argo CD adapter allows execution only when:

```text
AppProject schedule permits action
AND Application status is Synced
AND no Argo CD operation is active
```

Application health is not required by default.

## Decision order

1. Resolve Application and AppProject.
2. Evaluate effective sync windows.
3. If blocked by schedule, return `OutsideSchedule`.
4. If an operation is active, return `OwnerReconciling`.
5. If sync status is not `Synced`, return `OwnerOutOfSync`.
6. Otherwise return allowed and reconciled.

## Why Synced is checked before execution

When a window opens, Argo CD may have pending Git changes. KICK MUST wait for Argo CD to finish first, then recompute Deployment freshness.

Possible outcomes:

- Argo CD creates a new ReplicaSet newer than the dependency change: no kick remains necessary.
- Argo CD syncs unrelated resources and the Deployment remains older: KICK proceeds.

## Event-driven reevaluation

Application status changes MUST re-enqueue associated pending requests. Polling alone is insufficient.

## Degraded Applications

A `Synced` but `Degraded` Application is allowed by default because stale credentials may be the reason for degradation. A future configurable `requireHealthy` option MAY be added but is not required for v1.

## Acceptance criteria

- OutOfSync blocks.
- Active operation blocks.
- Synced and idle allows if the window allows.
- A completed Argo CD rollout can make the request unnecessary.
- Synced plus Degraded does not block by default.
