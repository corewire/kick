# Task 17: Documentation and release pipeline

## Goal

Create user, operator, and coding-agent documentation plus signed releases.

## Dependencies

- `specs/16-documentation-and-release.md`
- stable API and behavior

## Deliverables

- docs structure and quickstart.
- generated CRD, metrics, events, and CLI references.
- `llms.txt`, `llms-full.txt`, and maintained `AGENTS.md`.
- release image/chart/SBOM/signing pipeline.

## Acceptance criteria

- Quickstart succeeds on a clean Kind cluster with Argo CD.
- `docs-gen-check` catches drift.
- release notes declare compatibility and limitations.
