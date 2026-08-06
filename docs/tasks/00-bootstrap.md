# Task 00: Repository bootstrap

## Goal

Create the Kubebuilder/controller-runtime project skeleton and common developer workflow.

## Feature IDs

- `KICK-FEAT-017`

## Dependencies

- `specs/00-product-and-scope.md`
- `specs/14-testing-strategy.md`
- `specs/16-documentation-and-release.md`

## Deliverables

- Plain Kubebuilder project initialized with `controller-runtime`; no Operator SDK or OLM dependency.
- Go module and controller manager entry point.
- Local `bin/` tool installation.
- Make targets required by `AGENTS.md`.
- Basic Helm chart and namespace/service account.
- CI workflow for format, vet, lint, unit tests, Envtest smoke tests, generated diff, Helm lint/template, and feature coverage validation.
- Empty docs site or docs build placeholder.

## Non-goals

- No CRDs or reconcilers beyond placeholders.
- No Argo CD dependency.
- No Operator SDK, OLM bundle, CSV, or catalog packaging.

## Acceptance criteria

- `make fmt vet lint test helm-lint helm-template` succeeds.
- Generated files are reproducible.
- CI uses pinned tool versions.
- Repository structure is recognizably generated from Kubebuilder.
- `make feature-coverage` succeeds.
