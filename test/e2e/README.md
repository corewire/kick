# KICK e2e suite

This suite uses Chainsaw scenario directories under test/e2e/scenarios.

Run locally:

- make test-e2e
- make test-e2e-core
- make test-e2e-argocd
- make test-e2e-recovery
- make test-e2e-scenario E2E=KICK-E2E-032
- make test-e2e-render

Conventions:

- Each scenario directory maps to one stable scenario ID.
- Each scenario includes trace.yaml for feature mapping metadata.
- Scenarios are scaffolded placeholders until behavior-specific assertions are implemented.
