# Task 10: Argo CD owner resolver

## Goal

Resolve Deployment ownership to one Application and AppProject.

## Dependencies

- `tasks/08-provider-contract.md`
- accepted `tasks/09-argocd-research.md`
- `specs/06-argocd-ownership.md`

## Deliverables

- Tracking-ID parser/resolver.
- Application lookup across configured namespaces.
- Exact fallback membership index.
- AppProject lookup in control-plane namespace.
- Provider contract tests and Envtest.

## Acceptance criteria

- Non-control-plane Application namespace works.
- Stale annotation is rejected.
- No/multiple owners return typed blocks.
- Project namespace is never guessed from Application namespace.
