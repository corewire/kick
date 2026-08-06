# Task 03: Change observation research and storage decision

## Goal

Resolve or isolate Kubernetes change-time limitations and select durable observation storage.

## Dependencies

- `specs/03-change-observation.md`
- `specs/17-open-questions.md` sections 2, 8, and 9

## Deliverables

- ADR comparing candidate timestamp/change sources.
- Chosen observation storage model.
- API proposal and garbage-collection behavior.
- Explicit initial-baseline semantics.
- Prototype proving restart recovery.

## Non-goals

- Do not implement full source controllers until the ADR is accepted.

## Acceptance criteria

- Decision explains behavior during controller downtime.
- Metadata-only updates can be distinguished.
- No workload annotation is used.
- Secret data cannot leak through stored state.
