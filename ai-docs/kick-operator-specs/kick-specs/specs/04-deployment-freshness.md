# Deployment freshness evaluation

## Goal

Determine, from live state, whether a Deployment still needs a kick.

## Inputs

The evaluator requires:

- the current Deployment;
- its current dependency set;
- the latest relevant observation for each existing dependency;
- the ReplicaSet corresponding to the Deployment's current Pod template;
- current rollout status.

## Current ReplicaSet

The evaluator MUST identify the ReplicaSet owned by the Deployment whose Pod template corresponds to the Deployment's current Pod template.

It MUST NOT select a ReplicaSet merely because it is the newest object in the namespace.

Potential implementation inputs include:

- Deployment UID owner reference;
- `pod-template-hash`;
- normalized Pod-template comparison;
- Deployment revision metadata.

The exact robust selection algorithm remains an open question and MUST be isolated in:

```go
type RolloutInspector interface {
    Inspect(ctx context.Context, deployment *appsv1.Deployment) (RolloutState, error)
}
```

## Rollout state

The result should include:

```go
type RolloutState struct {
    CurrentReplicaSet types.NamespacedName
    StartedAt         time.Time
    InProgress        bool
    Complete          bool
    Failed            bool
    Reason            string
}
```

## Freshness decision

```go
type FreshnessDecision struct {
    RestartRequired bool
    LatestChange    *time.Time
    RolloutStarted  time.Time
    BlockingReason  string
}
```

Rules:

1. If a rollout is active, return blocked and do not start another rollout.
2. If no current ReplicaSet can be identified, return an explicit error/blocking decision.
3. Ignore observations for sources no longer referenced.
4. If no dependency has a relevant observation newer than the rollout, no restart is required.
5. If at least one current dependency is newer, restart is required.
6. Re-read all inputs immediately before execution.

## Rollout anchor

The rollout anchor `σ(w)` is the moment the currently running Pod template came
into being. It MUST be the latest moment that is provably true, because a source
change may only be dismissed when the running Pods provably already carry it.

| workload | `σ(w)` |
| --- | --- |
| any kind whose Pod template carries `kubectl.kubernetes.io/restartedAt` | that annotation |
| Deployment | `max(` creation time of the current ReplicaSet, latest `lastUpdateTime`/`lastTransitionTime` over `status.conditions` `)` |
| DaemonSet | `max(` creation time of the workload, latest condition transition time `)` |
| StatefulSet | creation time of the workload |
| Argo Rollout | `status.restartAt`, else the current ReplicaSet, else the workload creation time |

The Deployment condition anchor is effectively the moment the rollout became
available, which is the tightest recorded upper bound for "the Pods are running
this template".

StatefulSets and DaemonSets expose no comparable completion timestamp, so
`σ(w)` is only a lower bound there. A source created after the workload object
but before its Pods started is consequently counted as newer and produces
exactly one adoption restart. Manifest ordering (Helm and Argo CD apply Secrets
and ConfigMaps before workloads) makes this rare, and one extra restart is the
safe side of the ambiguity.

## Staleness

With `Λ(w)` the latest recorded change time over the in-scope dependencies of
`w` (`-∞` if none is observed):

```text
stale(w) ⟺ complete(w) ∧ Λ(w) > σ(w)
```

The comparison is strict, so equal timestamps count as fresh. Both operands MUST
retain sub-second precision (see `03-change-observation`, "Timestamp
precision").

## Adoption

Adoption is the first evaluation of a workload that already existed before KICK
did. It is restart-free exactly when every in-scope source is dated no later
than the rollout, which holds for sources untouched since before the workload
rolled out.

A workload created after all of its sources were observed is never evaluated at
all (`03-change-observation`, "Evaluation trigger") and has no `KickRequest`.
This is the correct no-op rather than a missed restart: its Pods started from
the current content of every source.

## Scaled-to-zero Deployments

The evaluator MUST support `spec.replicas: 0`. It should still identify the current ReplicaSet and permit a Pod-template rollout mutation when required. Exact rollout-completion semantics for zero replicas MUST be covered by e2e tests.

## Acceptance criteria

- A source newer than the current ReplicaSet returns restart required.
- All sources older returns no restart.
- A removed source no longer influences the result.
- A normal Deployment rollout newer than the source clears the requirement.
- Active rollout blocks a second kick.
- Scaled-to-zero behavior is deterministic and tested.
- Adopting a workload whose sources predate its rollout restarts nothing, for every supported kind.
- A workload created after its sources exist has no `KickRequest`, and a later change to one of those sources still restarts it.
- A change recorded in the same second as the rollout it supersedes restarts the workload.
