# Concepts

Kubernetes does **not** restart your workloads when a `Secret` or `ConfigMap`
they consume changes. KICK closes that gap. It watches a workload's
dependencies, detects when one changed **after** the last rollout, optionally
asks your GitOps tool for permission, and then issues exactly one controlled
restart.

![How KICK turns a dependency change into a gated restart](/images/kick-flow.svg "Observation → coalesce → gate → freshness → restart")

## The pipeline

Every restart decision follows the same four steps. Each concept page below
covers one of them in depth.

| Step | What happens | Concept |
|------|--------------|---------|
| 1. Detect the gap | A `Secret`/`ConfigMap` changes, but the running Pod keeps its old config. | [The hidden restart requirement](hidden-restart-requirement/) |
| 2. Discover deps | Find the `Secret`s and `ConfigMap`s the workload consumes via env and volumes. `imagePullSecrets` are excluded. | [Dependency discovery](dependency-discovery/) |
| 3. Check freshness | Compare the latest dependency change against the current rollout start. Newer dependency = stale = restart needed. | [Freshness](freshness/) |
| 4. Gate the restart | If a provider or windows are configured, restart only when the owner is in sync and the window is open. | [GitOps gates](gitops-gates/) |

By default (no `gitOps` provider) step 4 is a no-op and KICK restarts as soon as
a workload is stale.

{{< cards >}}
  {{< card link="hidden-restart-requirement/" title="Hidden restart requirement" subtitle="Why changing a Secret or ConfigMap silently drifts your Pods." >}}
  {{< card link="dependency-discovery/" title="Dependency discovery" subtitle="How KICK finds the Secrets and ConfigMaps a workload consumes." >}}
  {{< card link="freshness/" title="Freshness" subtitle="How KICK decides a running rollout is stale and needs a restart." >}}
  {{< card link="gitops-gates/" title="GitOps gates" subtitle="How native windows and a GitOps provider gate the restart." >}}
{{< /cards >}}

---

For the same model in formal notation, see the
[operator model](../theory/operator-model/). The diagrams are editable draw.io
SVGs (`docs/static/images/*.drawio.svg`) — open them in
[draw.io](https://app.diagrams.net) to edit, then save in place.

