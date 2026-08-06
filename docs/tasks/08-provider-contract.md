# Task 08: GitOps provider framework

## Goal

Implement the provider-neutral registry and generic gate types.

## Dependencies

- `specs/05-gitops-provider-contract.md`

## Deliverables

- Provider interface and registry.
- Typed owner and gate decisions.
- Detection arbitration for zero, one, or multiple providers.
- Contract-test harness reusable by adapters.

## Non-goals

- No Argo CD implementation in this task.

## Acceptance criteria

- Ambiguous providers block deterministically.
- Core packages do not import Argo CD or Flux APIs.
- Fake provider passes contract tests.
