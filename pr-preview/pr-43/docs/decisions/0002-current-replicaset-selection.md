# 
# ADR 0002: Current ReplicaSet Selection for Deployment Freshness

## Status

Accepted for implementation guidance.

## Context

Task 06 requires a robust way to identify the Deployment ReplicaSet corresponding to the current Deployment Pod template. The algorithm must not rely on "newest ReplicaSet wins" and must represent ambiguous states explicitly.

## Decision

Use a two-step algorithm:

1. Filter ReplicaSets to those controlled by the Deployment UID.
2. Select ReplicaSets whose PodTemplateSpec is equivalent to the Deployment template after normalizing hash-only labels (`pod-template-hash`).

Selection outcomes:

- exactly one match: current ReplicaSet selected;
- zero matches: explicit `NoMatchingReplicaSet`;
- more than one match: explicit `AmbiguousReplicaSetMatch`.

## Why this approach

- Owner UID scoping prevents cross-workload contamination.
- Template equivalence handles rollback and history cleanup correctly.
- It remains correct when the newest ReplicaSet is not current.

## Rejected approach

Newest ReplicaSet by creation timestamp alone:

- fails during rollback;
- fails during overlapping rollout states;
- can choose wrong ReplicaSet when history is retained.

## Rollout progress interpretation

`InProgress` is true when Deployment status indicates rollout activity (generation not observed, updatedReplicas lagging desired, stale old replicas, or insufficient availability). `Paused` produces a non-in-progress blocked reason.

## Zero replicas

`spec.replicas: 0` is supported. Current ReplicaSet can still be selected via template matching; completion remains deterministic when no rollout progress is active.

## Prototype evidence

- Unit tests cover normal rollout, active rollout, rollback, pause, zero replicas, and history cleanup.
- Envtest validates behavior against a real API server cache/list path.

