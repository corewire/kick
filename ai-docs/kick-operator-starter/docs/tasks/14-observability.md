# Task 14: Observability

## Goal

Add stable conditions, events, metrics, and safe structured logs.

## Dependencies

- `specs/13-observability.md`
- implemented KickRequest state machine

## Deliverables

- Metrics registration and tests.
- Event emission on transitions.
- Printer columns.
- Safe logging conventions.

## Acceptance criteria

- No Secret content appears in logs/events.
- Metrics have bounded cardinality.
- Stable reasons are documented and tested.
