---
title: KICK
layout: hextra-home
description: Kubernetes operator that restarts workloads when their Secrets and ConfigMaps change, gated by GitOps.
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
{{< hextra/hero-headline >}}
  KICK
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-8">
{{< hextra/hero-subtitle >}}
  Restart workloads when their Secrets & ConfigMaps change — gated by GitOps.
{{< /hextra/hero-subtitle >}}
</div>

> Kubernetes does not restart a Pod when a `Secret` or `ConfigMap` it consumes
> changes. KICK discovers those dependencies, detects relevant changes, checks
> that the running rollout predates the change, asks your GitOps tool for
> permission, and then issues exactly one restart via the standard
> `kubectl.kubernetes.io/restartedAt` annotation.

![How the KICK reconcile flow turns a dependency change into a gated restart](/images/kick-flow.svg "Observation → coalesce → gate → freshness → restart")

---

## I want to...

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Get started"
    subtitle="Install KICK and watch a Secret change restart a Deployment."
    link="docs/installation/"
  >}}
  {{< hextra/feature-card
    title="Understand the model"
    subtitle="The formal operator model: observation, freshness, gating, safety."
    link="docs/theory/operator-model/"
  >}}
  {{< hextra/feature-card
    title="Reference the API"
    subtitle="KickPolicy / KickRequest fields, metrics, events, configuration."
    link="docs/reference/"
  >}}
{{< /hextra/feature-grid >}}
