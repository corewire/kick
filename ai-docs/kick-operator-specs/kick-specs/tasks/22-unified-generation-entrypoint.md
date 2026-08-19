# Task 22: Unified generation entry point

## Goal

Make `make generate` the only command anyone needs to run for code, manifest, traceability, and
docs generation.

## Dependencies

- `specs/23-unified-generation-entrypoint.md`
- `specs/14-testing-strategy.md`
- `specs/20-e2e-suite-conventions.md`

## Deliverables

- `generate` becomes the umbrella target: `generate-deepcopy manifests api-field-coverage-gen docs-gen`.
- Deepcopy-only recipe moved to `generate-deepcopy`.
- `codegen` kept as an alias of `generate`.
- CI "Generate" step reduced to `make generate` before the `git diff --exit-code` drift check.
- Source docs updated to reference `make generate` only
  (`docs/content/docs/development/token-efficient-workflow.md` still says `make manifests generate`).

## Acceptance criteria

- `make generate` run twice in a row leaves `git status --porcelain` empty.
- `make verify` passes.
- `make test` passes.
- `make test-e2e` passes on a clean Kind cluster, with every scenario converging and exact rollout
  counts asserted. No scenario may be skipped, quarantined, or loosened for this task.
- No source document references `make codegen` or `make manifests generate` as the generation step.
- No generated file is edited by hand.

## Out of scope

- Changing what any individual generator emits.
- Adding new generators.
