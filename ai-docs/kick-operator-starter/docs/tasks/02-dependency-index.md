# Task 02: Dependency reverse indexes

## Goal

Register controller-runtime indexes and prove source-to-Deployment lookup.

## Dependencies

- `tasks/01-dependency-extractor.md`
- `specs/02-dependency-discovery.md`
- `specs/11-controller-architecture.md`

## Deliverables

- Secret-reference and ConfigMap-reference field indexes.
- Lookup service returning consuming Deployment keys.
- Envtest coverage for Deployment create, update, and delete.

## Acceptance criteria

- Lookup changes immediately after Deployment reference changes.
- Optional missing references are indexed.
- No full Deployment list is needed per source event.
