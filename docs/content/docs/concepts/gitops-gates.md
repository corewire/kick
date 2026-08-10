---
title: GitOps gates
weight: 40
description: How KICK gates a restart behind native schedule windows and a GitOps provider.
---

A **gate** decides whether a restart that is *needed* (the workload is
[stale](../freshness/)) may actually run *right now*. The gate never invents work:
it only ever delays or permits a restart that freshness already justified.

![If the GitOps owner is unknown or the sync window is closed, KICK stays blocked and re-checks later; only a clear owner with an open window lets the restart run.](/images/gitops-gate.drawio.svg)

Gating is **opt-in**. With no `gitOps` block — the default, `provider: None` —
KICK self-gates and restarts as soon as a dependency is stale. You add a gate
when a restart must wait for a maintenance window, or until your GitOps tool has
finished reconciling the workload.

## Two stages

The gate is evaluated in order. A restart runs only if **both** stages allow it.

```
stale workload ──▶ [ 1. schedule windows ] ──▶ [ 2. GitOps provider ] ──▶ restart
                          │                            │
                     OutsideSchedule            OwnerUnknown / OutOfSync / …
                          ▼                            ▼
                     wait & re-check              wait & re-check
```

### Stage 1 — native schedule windows

`spec.gitOps.schedule.windows[]` are KICK-native cron windows evaluated without
any provider. Each window is either `Allow` or `Deny`:

- The current time is **open** when it falls inside at least one `Allow` window
  and inside no `Deny` window.
- **Deny always wins.** A `Deny` window overlapping the current time blocks the
  restart even if an `Allow` window also matches.
- With no windows configured, this stage is always open.

When the clock is outside the allowed schedule the gate blocks with reason
`OutsideSchedule` and re-checks when the next window boundary arrives.

`schedule.source` selects where windows come from: `Provider` (default) lets a
provider contribute windows (for example Argo CD `AppProject` sync windows);
`None` uses only the KICK-native `windows[]`.

### Stage 2 — GitOps provider

`spec.gitOps.provider` selects who owns the restart decision:

| Provider | Behaviour |
|----------|-----------|
| `None` *(default)* | KICK self-gates; this stage always allows. |
| `ArgoCD` | Defer to Argo CD ownership and sync state. |
| `Auto` | Detect the managing provider automatically. |
| `Flux` | Reserved for Flux ownership (roadmap). |

With a real provider, KICK must resolve **exactly one** owner for the workload
and confirm that owner is reconciled before it restarts.

**Owner resolution** (Argo CD):

1. **Primary** — the workload's Argo CD tracking annotation.
2. **Fallback** — an indexed lookup of `Application`s that manage the workload.
3. **Zero or ambiguous** — no owner (`OwnerUnknown`) or more than one
   (`MultipleOwners`) blocks the automatic restart. KICK never guesses.

**Sync check** — when `requireReconciled` is `true` (the default), the owning
`Application` must report `Synced`. An out-of-sync or still-reconciling owner
blocks with `OutOfSync` / `SyncInProgress`, so KICK never restarts against
config the GitOps tool is mid-flight on.

## Blocking reasons

The gate surfaces why a restart is waiting on `KickRequest.status.gate.reason`:

| Reason | Stage | Meaning |
|--------|-------|---------|
| `Allowed` | — | Both stages passed; the restart may run. |
| `OutsideSchedule` | Windows | Current time is outside the allowed schedule. |
| `OwnerUnknown` | Provider | No GitOps owner could be resolved. |
| `MultipleOwners` | Provider | More than one owner matched; ambiguous. |
| `OutOfSync` | Provider | The owning application is not `Synced`. |
| `SyncInProgress` | Provider | The owning application is still reconciling. |

## Waiting behaviour

A blocked restart is not dropped. KICK persists the waiting phase and reason on
the `KickRequest` and re-evaluates the gate when either:

- a timer fires (to catch schedule-window boundaries), or
- a relevant provider object changes (for example the `Application` becomes
  `Synced`).

The moment the gate flips to `Allowed`, KICK re-reads live state and — if the
workload is still stale — issues the restart.

## Example

Restart only inside a nightly window **and** only once Argo CD has synced the
owner:

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: gated
spec:
  discovery:
    workloadSelector: {}
  schedule:
    windows:
      - type: Allow
        cron: "0 2 * * *"   # 02:00 daily
        duration: 1h
  gitOps:
    provider: ArgoCD
    requireReconciled: true
```

Native `schedule.windows` gate *when* KICK restarts; they are independent of the
provider. Argo CD `AppProject` sync windows still constrain when Argo itself
syncs, and the Argo provider honors them as part of its own gate.

## See also

- [Freshness](../freshness/) — what makes a restart *needed* before the gate runs.
- [KickPolicy reference](../../reference/kickpolicy/) — every `gitOps` field.
- [Operator model §6](../../theory/operator-model/#6-policy-scope-and-the-gate) —
  the gate in formal notation.