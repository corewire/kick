# Task 16: Chainsaw e2e suite

## Goal

Implement the required behavior scenarios against Kind and real Argo CD.

## Feature IDs

- All feature IDs marked `e2e: required` in `traceability/features.yaml`

## Dependencies

- `specs/15-e2e-scenarios.md`
- `specs/20-e2e-suite-conventions.md`
- completed controllers and Helm chart

## Deliverables

- One scenario directory per behavior group.
- One `trace.yaml` metadata file per scenario with stable scenario and feature IDs.
- reusable Kind and real Argo CD setup.
- render-only scenario validation.
- stable-state and lifecycle assertions, including manager restart recovery.
- redacted diagnostic artifact collection.
- compatibility matrix jobs.

## Acceptance criteria

- Core success flow passes.
- All blocking/race/recovery cases pass.
- Self-heal does not cause duplicate rollout.
- Tests prove imagePullSecrets are ignored.
- `make feature-coverage` confirms every mandatory feature has e2e coverage and every required scenario exists.
- Every behavior scenario proves stable convergence rather than only transient readiness.
- Failed runs upload controller, Argo CD, workload, event, and Chainsaw diagnostics without Secret content.
- Local and CI runs use the same Chainsaw scenario definitions.
