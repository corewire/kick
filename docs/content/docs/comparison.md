---
title: Comparison
weight: 35
description: How KICK compares to Stakater Reloader and Wave, including where those projects are the better choice.
llmsDescription: |
  Comparison of KICK against Stakater Reloader and wave-k8s/wave. All three
  restart workloads when a Secret or ConfigMap changes. Reloader and Wave opt
  in per workload via annotations; KICK uses a cluster-scoped KickPolicy with
  label selectors. Reloader writes a dummy env var or a last-reloaded-from
  annotation, Wave writes a config-hash annotation, KICK writes only
  kubectl.kubernetes.io/restartedAt. Only KICK gates restarts on GitOps state
  (Argo CD AppProject sync windows and Application Synced, Flux Kustomization
  and HelmRelease Ready) and on cron windows. Reloader is more mature and
  supports CSI SecretProviderClass, alerting webhooks, OpenShift
  DeploymentConfig and Argo Rollouts, which KICK does not.
---

KICK is not the first operator to restart workloads on configuration change.
[Stakater Reloader](https://github.com/stakater/Reloader) and
[Wave](https://github.com/wave-k8s/wave) solve the same core problem and are
both older and more widely deployed. This page explains what is actually
different, and when you should pick one of them instead.

## At a glance

| | Reloader | Wave | KICK |
|---|---|---|---|
| Opt-in | Workload annotation | Workload annotation | `KickPolicy` selectors |
| Writes | Env var, or annotation | `config-hash` | `restartedAt` |
| Trigger | Event | Hash differs | Change newer than rollout |
| Kinds | + `DeploymentConfig`, Argo Rollout | Deployment, StatefulSet, DaemonSet | Deployment, StatefulSet, DaemonSet |
| Unreferenced deps | By name, plus CSI | By name | Not supported |
| Timing | Debounce | Rate limit | Cron windows |
| GitOps gate | None | None | Argo CD, Flux |

## What is different about KICK

**It gates on GitOps state.** Neither Reloader nor Wave reads Argo CD or Flux
state. Reloader's documentation is candid that mutating the pod template makes
Argo CD report `OutOfSync`, and its answer is to mutate less. That solves drift,
not timing: a restart still fires during a freeze, because Argo CD sync windows
only ever constrained Argo CD's own syncs. KICK reads `spec.syncWindows` from
the owning `AppProject` and blocks until the window opens. See
[GitOps gates](../concepts/gitops-gates/).

**No annotations on your workloads.** Reloader and Wave both need an annotation
on each workload, which means you have to own the rendering. A `KickPolicy`
selects workloads by label, so KICK also covers third-party Helm charts and
vendored manifests you do not template.

**Freshness is a comparison, not a hash.** KICK compares the last relevant
change against the running rollout's start time, so a restart somebody else
already performed satisfies the condition, and restarting the operator does not
re-trigger anything. See [Freshness](../concepts/freshness/).

## When to use Reloader or Wave instead

KICK is narrower and much newer. Prefer the alternatives when:

- **You do not run Argo CD or Flux.** Without GitOps gating, KICK's main
  advantage does not apply, and Reloader is far better proven.
- **Your secrets are mounted through the Secrets Store CSI driver.** Reloader
  watches `SecretProviderClassPodStatus`; KICK has no equivalent.
- **Your application reads a Secret through the API** rather than through env
  or a volume. Reloader and Wave can both be told about such a dependency by
  name; KICK only discovers env and volume references.
- **You need alerting webhooks, OpenShift `DeploymentConfig`, or Argo
  Rollouts.** These are Reloader features with no KICK counterpart.
- **You want to avoid the initial restart when adopting the tool.** Wave's
  mutating webhook sets the first hash without a rollout, and holds pods
  `Pending` while a required Secret is missing.

## One thing that is not different

KICK also modifies the pod template — that is unavoidable, since changing the
template is what starts a rollout. The difference is what gets written: the
standard `kubectl.kubernetes.io/restartedAt` annotation that `kubectl rollout
restart` uses, rather than tool-specific hashes, environment variables, or
KICK-owned fields.

Checked against the Reloader and Wave documentation in August 2026. Both
projects move quickly; verify current behaviour before relying on this table.
