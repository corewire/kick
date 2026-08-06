# Task 04: Secret and ConfigMap observation controller

## Goal

Implement relevant-change detection and durable observations.

## Dependencies

- `tasks/02-dependency-index.md`
- accepted output of `tasks/03-observation-storage-spike.md`
- `specs/03-change-observation.md`

## Deliverables

- Secret and ConfigMap watches.
- Relevant-content comparison.
- Baseline creation.
- Observation persistence.
- Consumer lookup and request-enqueue interface.

## Acceptance criteria

- Content updates enqueue consumers.
- Metadata-only and identical-content updates do not.
- Optional object creation is handled.
- State survives manager restart.
