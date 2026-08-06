# Task 18: Feature-to-test traceability enforcement

## Goal

Implement machine-enforced comparison of normative KICK features against unit, Envtest, and end-to-end test coverage.

## Feature IDs

- `KICK-FEAT-018`

## Dependencies

- `specs/14-testing-strategy.md`
- `specs/15-e2e-scenarios.md`
- `specs/19-framework-and-test-traceability.md`
- `traceability/features.yaml`
- `traceability/e2e-scenarios.yaml`

## Deliverables

- `make feature-coverage` target.
- A coverage-checking command or script with unit tests.
- Validation of known feature and scenario IDs.
- Validation that all required test levels have concrete coverage.
- Validation that mapped Chainsaw directories exist and are enabled.
- A Markdown coverage report suitable for a CI artifact and PR summary.
- CI execution on every pull request.

## Required behavior

The checker MUST fail for:

- an implemented feature without all required test levels;
- a required e2e feature without a scenario;
- a scenario referencing an unknown feature;
- a missing or disabled required scenario;
- duplicate feature or scenario IDs;
- a `not_applicable` test level without a rationale;
- an implementation task referencing an unknown feature;
- planned-only tests for a feature declared implemented.

## Non-goals

- No runtime Kubernetes controller behavior.
- No calculation of statement or branch coverage percentages.
- No claim that traceability replaces ordinary code coverage tools.

## Acceptance criteria

- A valid repository produces a passing report.
- Fixture repositories for every required failure mode are covered by unit tests.
- Removing an e2e mapping from a required feature makes CI fail.
- Removing a required Chainsaw directory makes CI fail.
- Adding an unknown feature ID to a test makes CI fail.
- The report compares each feature with its required and actual unit, Envtest, and e2e coverage.
