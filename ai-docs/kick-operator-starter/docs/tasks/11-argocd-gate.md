# Task 11: Argo CD gate evaluator

## Goal

Implement exact supported sync-window and Application-state decisions.

## Dependencies

- `tasks/10-argocd-owner-resolver.md`
- accepted `tasks/09-argocd-research.md`
- `specs/07-argocd-sync-windows.md`
- `specs/08-argocd-application-gate.md`

## Deliverables

- Window evaluator.
- Application operation/sync-state evaluator.
- `RequeueAt` calculation.
- Watches that re-enqueue affected requests.

## Acceptance criteria

- Compatibility fixtures pass.
- OutOfSync and active operation block.
- Synced idle allows only when schedule permits.
- Window edits re-evaluate immediately.
