# Feature and test traceability

This directory defines the machine-readable relationship between KICK features and their required tests.

## Files

- `features.yaml`: normative feature inventory and required test levels.
- `e2e-scenarios.yaml`: required end-to-end scenarios and the features they prove.

## Intended implementation repository workflow

1. Add a feature to `features.yaml` before or with implementation.
2. Add feature IDs to the implementing task and tests.
3. Add or update a Chainsaw scenario with a `trace.yaml` file.
4. Run `make feature-coverage`.
5. CI rejects missing, unknown, disabled, or contradictory coverage.

The included checker validates the specification bundle. The production repository should extend it to inspect actual Go test files and Chainsaw directories.
