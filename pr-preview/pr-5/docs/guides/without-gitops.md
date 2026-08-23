# Running without GitOps

GitOps gating is the feature KICK is best known for, but it is **not** a
requirement. `spec.gitOps.provider` defaults to `None`, and a policy with no
`gitOps` block at all restarts workloads on any Kubernetes cluster.

## Minimal policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: team-a
spec:
  discovery:
    workloadSelector:
      matchLabels:
        kick.corewire.io/enabled: "true"
```

No `gitOps` block means no ownership resolution and no gate evaluation. A
detected change goes straight to the freshness check and then to the restart.
Cron windows, rate limiting and `dryRun` all work unchanged.

## Choosing a provider

| `spec.gitOps.provider` | Behaviour |
|---|---|
| `None` (default) | No ownership resolution, no gate. Restarts proceed. |
| `Auto` | Ask every registered provider to identify the owner. |
| `ArgoCD` | Resolve the owning `Application` and its `AppProject` sync windows. |
| `Flux` | Resolve the owning `Kustomization` or `HelmRelease` and require `Ready`. |
| `Kargo` | Resolve the authorised Kargo `Stage`, then delegate to Argo CD. |

## Do not use `Auto` without a GitOps controller

`Auto` asks each provider to detect ownership. On a cluster with neither Argo CD
nor Flux installed, no provider is confident, the gate resolves to
`ProviderUnavailable`, and the `KickRequest` waits indefinitely. This is
intentional — silently restarting a workload whose ownership could not be
established would violate KICK's core safety rule — but it means `Auto` is the
wrong choice here. Use `None`.

## Kargo is never auto-detected

Kargo does not write to workloads; Argo CD does. A Kargo-managed workload
therefore looks exactly like an Argo CD-managed one, and detection cannot tell
them apart. Set `provider: Kargo` explicitly to also gate on in-flight Stage
promotions.

## What you still get

- Change detection for env, `envFrom`, volume and Secrets Store CSI references
- Freshness comparison against the running rollout, so redundant restarts are skipped
- Cron restart windows and per-policy rate limiting
- Durable `KickRequest` objects, events, metrics and the timeline API
- `dryRun` previews and `NotificationPolicy` webhooks

