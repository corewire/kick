# KICK-E2E-077 - Flux Kustomization owner restarts

A Secret rotated in Git and applied by a real Flux `Kustomization` restarts the
workload exactly once. KICK resolves the owner from the
`kustomize.toolkit.fluxcd.io/name` and `/namespace` labels and waits until the
Kustomization is `Ready` at its current generation. A later Flux reconcile
neither reverts the restart nor triggers another.

- Feature: KICK-FEAT-031 ([trace.yaml](./trace.yaml))
- Setup: `make e2e-flux-setup`, run with `make test-e2e-flux`
