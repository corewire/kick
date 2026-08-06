# Task 01: Deployment dependency extractor

## Goal

Implement the pure Pod-template dependency extraction function.

## Dependencies

- `specs/01-domain-model.md`
- `specs/02-dependency-discovery.md`

## Deliverables

- `DependencyRef` type.
- Pure deterministic `ExtractDependencies` function.
- Table-driven unit tests for every reference path.

## Non-goals

- No controller, cache, API reads, or CRD.

## Acceptance criteria

- All supported env and volume references are returned.
- Init containers are included.
- Duplicates are removed.
- `imagePullSecrets` are excluded.
- Output ordering is stable.
