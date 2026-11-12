# KICK-E2E-080 - Flux missing owner blocks

A Deployment whose Flux ownership labels point at a `Kustomization` that does
not exist is never restarted: the KickRequest stays in `WaitingForOwner` with
gate reason `OwnerUnknown` and the Deployment keeps generation 1.

- Feature: KICK-FEAT-031 ([trace.yaml](./trace.yaml))
- Setup: `make e2e-flux-setup`, run with `make test-e2e-flux`
