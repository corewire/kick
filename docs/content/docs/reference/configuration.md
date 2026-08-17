# Configuration Reference

Primary configuration surface: Helm chart values in `charts/kick/values.yaml`.

| Key | Description | Default |
| --- | --- | --- |
| `namespace.name` | Controller namespace | `kick-system` |
| `requestRetention` | Completed request retention duration | `24h` |
| `rolloutTimeout` | Restart rollout timeout | `15m` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `podDisruptionBudget.enabled` | Protect single replica availability | `true` |
| `resources` | Controller CPU/memory requests/limits | set in values |
| `integrations.argocd.enabled` | Enable Argo CD adapter logic | `true` |
| `integrations.argocd.namespace` | Argo CD control-plane namespace | `argocd` |
| `integrations.argocd.applicationNamespaces` | Application namespaces to search | `[]` |
| `integrations.flux.enabled` | Enable Flux adapter logic | `true` |
| `integrations.argoRollouts.enabled` | Treat `argoproj.io` Rollouts as restartable workloads, and grant RBAC for them | `false` |
| `integrations.secretsStoreCSI.enabled` | Observe `SecretProviderClassPodStatus` for Secrets Store CSI rotation, and grant RBAC for it | `false` |
| `integrations.kargo.enabled` | Grant RBAC for Kargo `Stages` and `Promotions` so `provider: Kargo` can be used | `false` |

Example:

```yaml
integrations:
  argocd:
    enabled: true
    applicationNamespaces: []
  flux:
    enabled: true
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
| `--rollout-timeout` | Maximum time KICK waits for rollout convergence before marking a request timed out. | `15m` |
| `--enable-argocd` | Gate restarts on Argo CD Application state. Ignored when the CRD is absent. | `true` |
| `--enable-flux` | Gate restarts on Flux Kustomization and HelmRelease state. Ignored when CRDs are absent. | `true` |
| `--enable-kargo` | Block restarts while a Kargo Promotion is active. Ignored when the CRD is absent. | `false` |
| `--enable-argo-rollouts` | Watch and restart `argoproj.io/v1alpha1` Rollouts. Ignored when the CRD is absent. | `false` |
| `--enable-csi-integration` | Watch `SecretProviderClassPodStatus`. Ignored when the CRD is absent. | `false` |

Every integration additionally requires its CRD to exist in the cluster; the
manager probes the REST mapper at startup and skips the integration rather than
failing. The Helm chart sets these flags from matching
`integrations.*.enabled` values.

### Detection happens once at start-up

Integrations — Argo CD, Flux, Kargo, Argo Rollouts and Secrets Store CSI — are
detected exactly once while the manager starts. If their CRDs are installed
later, the integration stays inactive until the KICK manager pod is restarted.
The manager logs one line per skipped integration at start-up, naming the
integration and the group/version/kind it did not find, so `kubectl logs` on
the manager pod tells you what is inactive. The probe is deliberately not
dynamic: registering a watch or a field index for a kind that does not exist
aborts the manager, so the check has to happen before the manager runs.

Controller runtime defaults and remaining flags are defined by chart templates and manager args in Kubernetes manifests.