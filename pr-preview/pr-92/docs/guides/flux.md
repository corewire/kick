# Flux

With `provider: Flux`, KICK restarts a Flux-managed workload only when its
`Kustomization` or `HelmRelease` has finished reconciling. Verified against
Flux v2.10.0 ([installer](https://github.com/corewire/kick/blob/main/test/e2e/setup/flux/install-flux.sh)).

## Enable the integration

On by default: `--enable-flux` / Helm `integrations.flux.enabled`. The manager
skips it when the Flux CRDs are absent at start-up
([configuration](../../reference/configuration/)).

## Policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: kick-e2e-077
spec:
  discovery:
    workloadSelector: {}
  gitOps:
    provider: Flux
```

## How the owner is found

Flux labels every object it applies. KICK reads them from the workload
([provider](https://github.com/corewire/kick/blob/main/internal/gitops/flux/provider.go)):

| Labels | Owner |
|---|---|
| `kustomize.toolkit.fluxcd.io/name`, `/namespace` | `Kustomization` |
| `helm.toolkit.fluxcd.io/name`, `/namespace` | `HelmRelease` |

The owner object must exist. No labels or a missing owner block with
`OwnerUnknown`; both label sets block with `AmbiguousOwner`.

## Gate

| Owner state | Gate reason |
|---|---|
| `spec.suspend: true` | `OwnerOutOfSync` |
| `status.observedGeneration` behind `metadata.generation` | `OwnerReconciling` |
| `Reconciling=True` or `Ready=Unknown` | `OwnerReconciling` |
| `Ready=False` | `OwnerOutOfSync` |
| `Ready=True` | `Allowed` |

A blocked request is re-checked every 30 seconds.

## Examples

| Scenario | Proves |
|---|---|
| [KICK-E2E-077](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-077-flux-kustomization-owner-restarts) | Secret rotated in Git restarts once; a later Flux apply keeps it |
| [KICK-E2E-078](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-078-flux-suspended-owner-blocks) | Suspended owner blocks; resume restarts once |
| [KICK-E2E-079](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-079-flux-helmrelease-owner-restarts) | `HelmRelease` owner restarts once |
| [KICK-E2E-080](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-080-flux-missing-owner-blocks) | Missing owner blocks |

Traceability: KICK-FEAT-031 in [features.yaml](https://github.com/corewire/kick/blob/main/traceability/features.yaml).

