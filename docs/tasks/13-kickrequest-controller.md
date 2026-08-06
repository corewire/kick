# Task 13: End-to-end KickRequest reconciler

## Goal

Compose owner resolution, gate evaluation, freshness evaluation, and execution.

## Dependencies

- tasks 05, 07, 08, 10, 11, and 12
- `specs/11-controller-architecture.md`

## Deliverables

- KickRequest reconciler.
- Complete phase/condition transitions.
- Startup recovery of non-terminal requests.
- Leader-election-safe behavior.

## Acceptance criteria

- Closed gate waits durably.
- Open gate always re-checks live freshness before action.
- No-longer-required transitions correctly.
- Manager restart during every waiting/executing phase is safe.
