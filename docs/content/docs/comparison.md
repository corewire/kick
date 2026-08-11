---
title: Comparison
weight: 35
description: How KICK compares to Stakater Reloader and Wave, including where those projects are the better choice.
llmsDescription: |
  Comparison of KICK against Stakater Reloader and wave-k8s/wave. All three
  restart workloads when a Secret or ConfigMap changes. Reloader and Wave opt
  in per workload via annotations; KICK uses a KickPolicy with label selectors.
  Reloader writes a dummy env var or a last-reloaded-from annotation, Wave
  writes a config-hash annotation, KICK writes only
  kubectl.kubernetes.io/restartedAt (spec.restartAt for Argo Rollouts).
  KICK does not require GitOps: gitOps.provider defaults to None and gating is
  opt-in. Only KICK gates restarts on GitOps state (Argo CD AppProject sync
  windows and Application Synced, Flux Kustomization and HelmRelease Ready,
  Kargo Stage promotions) and on cron windows. KICK supports Secrets Store CSI
  SecretProviderClass rotation, Argo Rollouts, delivery webhooks via
  NotificationPolicy, and a dryRun mode. KICK deliberately does not support
  dependencies declared by name, holding pods Pending, or OpenShift
  DeploymentConfig. Reloader remains more mature and more widely deployed.
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
| Kinds | + `DeploymentConfig`, Argo Rollout | Deployment, StatefulSet, DaemonSet | + Argo Rollout (opt-in) |
| Secrets Store CSI | Yes | No | Yes (opt-in) |
| Deps declared by name | Yes | Yes | Deliberately not supported |
| Timing | Debounce | Rate limit | Cron windows, rate limit |
| GitOps gate | None | None | Optional: Argo CD, Flux, Kargo |
| Restart-time webhooks | Yes | No | Yes (`NotificationPolicy`) |
| Preview without acting | No | No | Yes (`dryRun`) |

## What is different about KICK

**It can gate on GitOps state.** Neither Reloader nor Wave reads Argo CD or
Flux state. Reloader's documentation is candid that mutating the pod template
makes Argo CD report `OutOfSync`, and its answer is to mutate less. That solves
drift, not timing: a restart still fires during a freeze, because Argo CD sync
windows only ever constrained Argo CD's own syncs. KICK reads `spec.syncWindows`
from the owning `AppProject` and blocks until the window opens. See
[GitOps gates](../concepts/gitops-gates/).

Gating is **opt-in**. `spec.gitOps.provider` defaults to `None`, so KICK runs on
a cluster with no GitOps controller at all. See
[Running without GitOps](../guides/without-gitops/).

**No annotations on your workloads.** Reloader and Wave both need an annotation
on each workload, which means you have to own the rendering. A `KickPolicy`
selects workloads by label, so KICK also covers third-party Helm charts and
vendored manifests you do not template.

**Freshness is a comparison, not a hash.** KICK compares the last relevant
change against the running rollout's start time, so a restart somebody else
already performed satisfies the condition, and restarting the operator does not
re-trigger anything. See [Freshness](../concepts/freshness/).

**Adoption does not restart what is already fresh.** The first time KICK sees a
Secret or ConfigMap it anchors the baseline to the last write the API server
recorded for it, unmodified. A source untouched since its creation is older than
the running rollout, so installing KICK over a healthy cluster performs no
restarts, and it needs no mutating webhook to achieve that. A source that *was*
written after the workload last rolled out is genuinely stale, and KICK restarts
it once. Drift the API server did not record is not detected retroactively — the
next real change picks it up.

**Every restart is a durable object.** A `KickRequest` records why a restart was
wanted, what blocked it, and what finally happened. `spec.dryRun` on a
`KickPolicy` runs the entire pipeline and stops immediately before the patch,
leaving a `DryRun` `KickRequest` you can inspect.

## When to use Reloader or Wave instead

KICK is narrower and much newer. Prefer the alternatives when:

- **Your application reads a Secret through the API** rather than through env,
  a volume, or a Secrets Store CSI mount. Reloader and Wave can both be told
  about such a dependency by name. KICK deliberately does not support this; see
  below.
- **You need OpenShift `DeploymentConfig`.** KICK has no counterpart and none
  is planned.
- **You want pods held `Pending` while a required Secret is missing.** Wave's
  mutating webhook does this. KICK never blocks scheduling.
- **You want the most proven option.** Reloader has years of production use
  behind it across far more clusters than KICK.

## Deliberate non-goals

These are not gaps waiting to be filled. They are decisions.

**Dependencies declared by name.** Reloader's `secret.reloader.stakater.com/reload`
and Wave's `wave.pusher.com/extra-configmaps` let you name a Secret the workload
does not actually reference. KICK only restarts on dependencies it can *prove*
the workload consumes, by reading the pod spec. A hand-maintained list drifts
silently from reality, and a stale entry produces restarts nobody can explain.
An application that reads a Secret through the API is better served by a client
that re-reads it, or by an explicit `KickRequest` created by whatever triggered
the change.

**Holding pods `Pending`.** Blocking scheduling requires a mutating admission
webhook in the pod path. A failure there is a cluster-wide outage of pod
creation, which is a much worse failure mode than the one it prevents. KICK has
no admission webhook at all and can never take the cluster down this way.

## One thing that is not different

KICK also modifies the pod template — that is unavoidable, since changing the
template is what starts a rollout. The difference is what gets written: the
standard `kubectl.kubernetes.io/restartedAt` annotation that `kubectl rollout
restart` uses, rather than tool-specific hashes, environment variables, or
KICK-owned fields. Argo Rollouts are the exception: KICK sets `spec.restartAt`,
which is the Rollout controller's own restart mechanism, so a configuration
change does not run the canary or blue-green strategy.

Checked against the Reloader and Wave documentation in August 2026. Both
projects move quickly; verify current behaviour before relying on this table.
