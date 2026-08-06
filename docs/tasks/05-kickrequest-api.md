# Task 05: KickRequest API

## Goal

Implement the durable request CRD, status model, and coalescing rules.

## Dependencies

- `specs/09-restart-request-api.md`
- accepted observation storage design

## Deliverables

- `kick.corewire.io/v1alpha1` API types.
- CRD validation and printer columns.
- Status conditions and phases.
- Helper that creates or updates one active request per Deployment.
- Retention configuration skeleton.

## Acceptance criteria

- Repeated events coalesce.
- Status updates use status subresource.
- Envtest covers defaults, validation, and conflict retries.
