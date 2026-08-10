# Configuration Reference

Primary configuration surface: Helm chart values in `charts/kick/values.yaml`.

| Key | Description | Default |
| --- | --- | --- |
| `namespace.name` | Controller namespace | `kick-system` |
| `argocd.enabled` | Enable Argo CD adapter logic | `true` |
| `argocd.namespace` | Argo CD control-plane namespace | `argocd` |
| `argocd.applicationNamespaces` | Application namespaces to search | `["*"]` |
| `requestRetention` | Completed request retention duration | `24h` |
| `rolloutTimeout` | Restart rollout timeout | `15m` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `podDisruptionBudget.enabled` | Protect single replica availability | `true` |
| `resources` | Controller CPU/memory requests/limits | set in values |
| `integrations.argoRollouts.enabled` | Treat `argoproj.io` Rollouts as restartable workloads, and grant RBAC for them | `false` |
| `integrations.secretsStoreCSI.enabled` | Observe `SecretProviderClassPodStatus` for Secrets Store CSI rotation, and grant RBAC for it | `false` |
| `integrations.kargo.enabled` | Grant RBAC for Kargo `Stages` and `Promotions` so `provider: Kargo` can be used | `false` |

Example:

```yaml
integrations:
  argoRollouts:
    enabled: true
  secretsStoreCSI:
    enabled: true
  kargo:
    enabled: false
```

## Manager flags

| Flag | Description | Default |
| --- | --- | --- |
| `--enable-argo-rollouts` | Watch and restart `argoproj.io/v1alpha1` Rollouts. Ignored when the CRD is absent. | `false` |
| `--enable-csi-integration` | Watch `SecretProviderClassPodStatus`. Ignored when the CRD is absent. | `false` |

Each optional integration is off by default and additionally requires its CRD to
exist in the cluster; the manager probes the REST mapper at startup and skips
the integration rather than failing. The Helm chart sets both flags from the
matching `integrations.*.enabled` value.

Controller runtime defaults and remaining flags are defined by chart templates and manager args in Kubernetes manifests.