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

Controller runtime defaults and flags are defined by chart templates and manager args in Kubernetes manifests.