# Task 06: Deployment rollout inspector research

## Goal

Select and prove the current ReplicaSet algorithm.

## Dependencies

- `specs/04-deployment-freshness.md`
- `specs/17-open-questions.md` section 3

## Deliverables

- ADR for current ReplicaSet selection.
- Pure/template comparison helpers.
- Prototype tests for normal rollout, active rollout, rollback, pause, zero replicas, and history cleanup.

## Acceptance criteria

- Algorithm does not rely on newest ReplicaSet alone.
- Ambiguous state is represented explicitly.
- Behavior is verified against a real API server or Kind.
