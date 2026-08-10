---
title: KICK
layout: hextra-home
description: Kubernetes operator that restarts workloads when their Secrets and ConfigMaps change, on a schedule you control or gated by GitOps.
llmsDescription: |
  KICK is a Kubernetes operator that restarts a workload (Deployment,
  StatefulSet, DaemonSet) when a Secret or ConfigMap it consumes changes.
  Pipeline: source change -> observation (fingerprint) -> KickRequest per target
  (coalesced) -> gate (native windows or GitOps provider None/Auto/ArgoCD/Flux)
  -> freshness (latest relevant change vs current rollout start) -> executor
  stamps kubectl.kubernetes.io/restartedAt. API group kick.corewire.io/v1alpha1.
  imagePullSecrets are ignored; KICK injects no state into workloads.
---

<div class="hx-mt-6 hx-mb-6">
  <img src="images/kick-long-dark.png" alt="KICK — Give Stale Workloads the Boot" class="kick-logo kick-logo-on-light" style="max-width: 307px; width: 100%; margin: 0 auto; display: block">
  <img src="images/kick-long-light.png" alt="KICK — Give Stale Workloads the Boot" class="kick-logo kick-logo-on-dark" style="max-width: 307px; width: 100%; margin: 0 auto; display: none">
</div>

<style>
.dark .kick-logo-on-light { display: none !important; }
.dark .kick-logo-on-dark { display: block !important; }
/* hextra-home has no prose styling, so home section headings need explicit sizing. */
.kick-home-heading { font-size: 1.75rem; font-weight: 600; letter-spacing: -0.02em; margin: 3rem 0 0.5rem; }
.kick-home-lead { color: rgb(107 114 128); margin-bottom: 1.5rem; }
.dark .kick-home-lead { color: rgb(156 163 175); }
/* hextra-home wraps content in flex-col/items-start, which shrinks the diagram to 300px. */
main pre.mermaid, main div:has(> pre.mermaid) { width: 100%; }
main pre.mermaid { max-width: 900px; margin-inline: auto; }
.kick-gap { border-left: 3px solid rgb(99 102 241); padding: 0.5rem 0 0.5rem 1rem; margin: 2rem 0 1.25rem; }
.kick-body { margin-bottom: 1.25rem; }
.kick-intro { margin-bottom: 2rem; }
.kick-modes { display: grid; grid-template-columns: max-content 1fr; gap: 0.75rem 2rem; margin: 0 0 2rem; }
.kick-modes dt { font-weight: 600; }
.kick-modes dd { margin: 0; color: rgb(107 114 128); }
.dark .kick-modes dd { color: rgb(156 163 175); }
.kick-modes code { font-size: 0.875em; }
@media (max-width: 640px) {
  .kick-modes { grid-template-columns: 1fr; gap: 0; }
  .kick-modes dd { margin-bottom: 1rem; }
}
</style>

<div class="hx-mb-8">
{{< hextra/hero-subtitle >}}
  Restart workloads when their Secrets &amp; ConfigMaps change &mdash; without breaking your Argo CD sync windows.
{{< /hextra/hero-subtitle >}}
</div>

<div class="kick-gap">
Kubernetes never restarts a Pod when a <code>Secret</code> or <code>ConfigMap</code> it
consumes changes. Environment variables are read once, at Pod start. Mounted
files do update on the node, but only take effect if the process re-reads them.
The Pod keeps serving its old config &mdash; no error, no signal.
</div>

<p class="kick-body">Under GitOps this hurts twice. Argo CD happily syncs the new <code>Secret</code>,
reports <code>Synced</code>, and moves on &mdash; the workload stays stale while the
dashboard is green. So somebody runs <code>kubectl rollout restart</code>, by hand or
from a pipeline. That restart is an undeclared change, and it ignores the
freeze windows on your <code>AppProject</code>, because sync windows only ever
constrained Argo CD's own syncs.</p>

<p class="kick-intro">KICK closes the gap from inside the GitOps model. It detects the change,
confirms the running rollout is older than it, and reads <code>spec.syncWindows</code>
straight off the owning <code>AppProject</code> &mdash; no sidecar, no plugin, no window
definitions copied into a second place. A restart KICK triggers is subject to
the same freeze as a sync. All it writes is the standard
<code>kubectl.kubernetes.io/restartedAt</code> annotation: no hashes, no environment
variables, no KICK-owned fields on your workloads.</p>

{{< tabs >}}

{{< tab name="Restart" >}}
{{< asciinema file="casts/restart.cast" autoplay="true" loop="true" speed="0.75" >}}
{{< /tab >}}

{{< tab name="KickRequest" >}}
{{< asciinema file="casts/kickrequest.cast" autoplay="true" loop="true" speed="0.5" >}}
{{< /tab >}}

{{< tab name="Events" >}}
{{< asciinema file="casts/events.cast" autoplay="true" loop="true" speed="0.5" >}}
{{< /tab >}}

{{< /tabs >}}

```mermaid
flowchart LR
  A[Secret / ConfigMap<br/>changes] --> B[Observe<br/>fingerprint]
  B --> C[KickRequest<br/>per workload]
  C --> D{Schedule<br/>window}
  D -->|closed| W[Wait]
  D -->|open| G{GitOps gate<br/>if enabled}
  G -->|blocked| W
  G -->|allowed| E{Rollout<br/>stale?}
  E -->|no| S[Skip]
  E -->|yes| R[Restart<br/>restartedAt]
```

<h2 class="kick-home-heading">When the restart happens</h2>

<dl class="kick-modes">
  <dt>Inside your Argo CD sync windows</dt>
  <dd>Read from the owning <code>AppProject</code>. A closed window blocks the restart with <code>OutsideSchedule</code> and re-checks when it opens.</dd>
  <dt>Once Argo CD is settled</dt>
  <dd>Waits for the owning <code>Application</code> to finish its operation and report <code>Synced</code>.</dd>
  <dt>When Flux is done</dt>
  <dd>Waits for the owning <code>Kustomization</code> or <code>HelmRelease</code> to be <code>Ready</code> and not reconciling.</dd>
  <dt>Inside a window you define</dt>
  <dd>Cron <code>Allow</code> / <code>Deny</code> windows in the time zone you specify, for clusters with no GitOps tool.</dd>
  <dt>Right away</dt>
  <dd>The default. No GitOps tool, no schedule, no configuration.</dd>
</dl>

---

<h2 class="kick-home-heading">I want to...</h2>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    icon="lightning-bolt"
    title="Get started"
    subtitle="Install with Helm, or from source."
    link="docs/installation/"
  >}}
  {{< hextra/feature-card
    icon="play"
    title="See it work"
    subtitle="A Secret change restarting a Deployment on Kind."
    link="docs/quickstart/"
  >}}
  {{< hextra/feature-card
    icon="light-bulb"
    title="Understand the concepts"
    subtitle="Dependency discovery, freshness, gating."
    link="docs/concepts/"
  >}}
  {{< hextra/feature-card
    icon="book-open"
    title="Follow a guide"
    subtitle="Argo CD, External Secrets, troubleshooting."
    link="docs/guides/"
  >}}
  {{< hextra/feature-card
    icon="code"
    title="Reference the API"
    subtitle="KickPolicy, KickRequest, metrics, events."
    link="docs/reference/"
  >}}
  {{< hextra/feature-card
    icon="shield-check"
    title="Run it in production"
    subtitle="RBAC, security, scalability, upgrades."
    link="docs/operations/"
  >}}
  {{< hextra/feature-card
    icon="academic-cap"
    title="Read the theory"
    subtitle="The formal operator model."
    link="docs/theory/operator-model/"
  >}}
  {{< hextra/feature-card
    icon="terminal"
    title="Hack on KICK"
    subtitle="Debugging, the timeline UI, workflow."
    link="docs/development/"
  >}}
  {{< hextra/feature-card
    icon="archive"
    title="Review the decisions"
    subtitle="Architecture decision records."
    link="docs/decisions/"
  >}}
{{< /hextra/feature-grid >}}

<h2 class="kick-home-heading">Using an AI agent?</h2>

<p class="kick-home-lead">Point it at one URL instead of crawling the site.</p>

{{< hextra/feature-grid cols="3" >}}
  {{< hextra/feature-card
    icon="sparkles"
    title="Feed to an AI agent"
    subtitle="Endpoints, Markdown output, agent instructions."
    link="docs/for-ai-agents/"
  >}}
  {{< hextra/feature-card
    icon="document-text"
    title="llms.txt"
    subtitle="Compact summary: APIs, runtime model, key references."
    link="llms.txt"
  >}}
  {{< hextra/feature-card
    icon="database"
    title="llms-full.txt"
    subtitle="Every page in one file."
    link="llms-full.txt"
  >}}
{{< /hextra/feature-grid >}}
