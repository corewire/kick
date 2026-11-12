# KICK-E2E-078 - Flux suspended owner blocks

While the owning `Kustomization` has `spec.suspend: true`, a Secret rotation
leaves the KickRequest in `WaitingForApplicationSync` with gate reason
`OwnerOutOfSync` and the Deployment untouched. Resuming the Kustomization opens
the gate and KICK restarts the workload exactly once.

- Feature: KICK-FEAT-031 ([trace.yaml](./trace.yaml))
- Setup: `make e2e-flux-setup`, run with `make test-e2e-flux`
