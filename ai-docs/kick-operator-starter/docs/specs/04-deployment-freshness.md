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

## Scaled-to-zero Deployments

The evaluator MUST support `spec.replicas: 0`. It should still identify the current ReplicaSet and permit a Pod-template rollout mutation when required. Exact rollout-completion semantics for zero replicas MUST be covered by e2e tests.

## Acceptance criteria

- A source newer than the current ReplicaSet returns restart required.
- All sources older returns no restart.
- A removed source no longer influences the result.
- A normal Deployment rollout newer than the source clears the requirement.
- Active rollout blocks a second kick.
- Scaled-to-zero behavior is deterministic and tested.
