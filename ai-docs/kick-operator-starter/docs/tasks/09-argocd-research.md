# Task 09: Argo CD compatibility research

## Goal

Resolve the Argo CD-specific open questions before production adapter implementation.

## Dependencies

- `specs/06-argocd-ownership.md`
- `specs/07-argocd-sync-windows.md`
- `specs/08-argocd-application-gate.md`
- `specs/17-open-questions.md` sections 1, 4, 5, 6, and 7

## Deliverables

- Supported Argo CD version matrix.
- ADR for control-plane namespace discovery.
- Tracking-ID format fixtures and parser decision.
- Ownership fallback data-source decision.
- Sync-window semantics fixtures.
- Rollout-annotation self-heal test results.

## Acceptance criteria

- Every unresolved provider assumption is documented.
- Unsupported Argo CD modes are listed explicitly.
- Test fixtures are reusable by the adapter.
