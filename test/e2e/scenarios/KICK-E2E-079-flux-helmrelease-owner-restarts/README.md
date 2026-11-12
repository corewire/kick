# KICK-E2E-079 - Flux HelmRelease owner restarts

A workload installed by a real Flux `HelmRelease` (chart from Git) carries
`helm.toolkit.fluxcd.io/name` and `/namespace` labels. A Secret rotation
restarts it exactly once with the HelmRelease as the resolved owner.

- Feature: KICK-FEAT-031 ([trace.yaml](./trace.yaml))
- Setup: `make e2e-flux-setup`, run with `make test-e2e-flux`
