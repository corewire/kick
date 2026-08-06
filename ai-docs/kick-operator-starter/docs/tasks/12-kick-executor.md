# Task 12: Restart executor and rollout observation

## Goal

Trigger and observe exactly one standard Deployment rollout.

## Dependencies

- `tasks/05-kickrequest-api.md`
- `tasks/07-freshness-evaluator.md`
- `specs/10-restart-execution.md`

## Deliverables

- Minimal Pod-template annotation patch.
- Conflict-safe idempotency logic.
- Rollout completion and timeout observer.
- Unit and Envtest coverage.

## Acceptance criteria

- One action creates one ReplicaSet.
- Retry does not create another rollout.
- No Pod deletion.
- Failure does not loop indefinitely.
