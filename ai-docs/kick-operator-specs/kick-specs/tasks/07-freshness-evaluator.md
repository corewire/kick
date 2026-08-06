# Task 07: Freshness evaluator

## Goal

Implement provider-independent stale/fresh decisions.

## Dependencies

- accepted output of `tasks/06-rollout-inspector-spike.md`
- `specs/01-domain-model.md`
- `specs/04-deployment-freshness.md`

## Deliverables

- `RolloutInspector` implementation.
- Pure `FreshnessEvaluator`.
- Unit tests and Envtest integration.

## Acceptance criteria

- Newer dependency requires kick.
- Newer rollout clears requirement.
- Removed dependencies are ignored.
- Active rollout returns blocked.
- Zero-replica behavior is tested.
