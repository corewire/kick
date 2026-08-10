# Task 21 — API redesign (finalize before first use)

## Goal

Lock the `kick.corewire.io` API into a shape we can commit to **before** anyone
runs the operator, so that promoting `v1alpha1 → v1` never requires a breaking
change. This document is the target: the YAML we *wish* to have.

> **Status: implemented.** This shape is the live `v1alpha1` API. `honorSyncWindows`
> was intentionally **not** added — Argo CD `AppProject` sync windows are always
> honored by the Argo provider gate, so no per-policy toggle is needed and restart
> timing lives purely in `spec.schedule`.

## Design principles

1. **One concern per top-level group.** Each `spec` block answers exactly one
   question: *what*, *when*, *who*, *how*.
2. **Names must not lie.** A field's path must reflect what it does. Native
   schedule windows are not GitOps, so they must not live under `gitOps`.
3. **Group knobs that will grow.** Behaviour that we expect to extend later
   (restart strategy, concurrency) goes under a group now, so new fields are
   additive — never a new top-level field or a move.
4. **Optional with safe defaults.** Every block is omittable; omission yields the
   least-surprising behaviour.
5. **Provider-neutral vocabulary.** Wording works for Argo CD *and* Flux
   (`reconciled`, not `synced`).
6. **Explicit intent for wide blast radius.** A match causes restarts, so
   "manage everything" must be typed on purpose, never the accidental default.
7. **The workload stays clean.** The API never carries state we would write into
   a workload (content hashes, env, annotations beyond the standard restart
   stamp).

## The problems we are fixing

| Current path | Problem | New path |
|--------------|---------|----------|
| `spec.gitOps.schedule` | Schedule windows are evaluated **without** a provider — they are not GitOps. | `spec.schedule` |
| `spec.gitOps.schedule.source: Provider\|None` | Enum conflates native scheduling with provider integration, and buries native windows under `gitOps`. | native `windows[]` → `spec.schedule`; source dropped (Argo sync windows are always honored by the Argo provider gate) |
| `spec.gitOps.schedule.windows[].kind` | `kind` collides with the Kubernetes `kind` concept. | `windows[].type` |
| `spec.gitOps.schedule.windows[].schedule` | A field named `schedule` nested inside a `schedule` block. | `windows[].cron` |
| `spec.minInterval` (flat) | Restart behaviour will grow (strategy, concurrency); a flat field forces future top-level additions. | `spec.restart.minInterval` |

---

## Target: `KickPolicy` (the object users write)

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: web
spec:
  suspend: false                    # pause this policy (no matches, no restarts)
                                    # without deleting it. Default: false.

  # ── WHAT ─────────────────────────────────────────────────────────────
  # Which workloads this policy manages, and which of their dependency
  # changes count as a trigger.
  discovery:
    workloadSelector:               # REQUIRED. `{}` = every supported workload
      matchLabels: { app: web }     #   in the namespace (explicit opt-in).
    dependencySelector: {}          # optional: omit = all consumed Secrets/ConfigMaps

  # ── WHEN ─────────────────────────────────────────────────────────────
  # Time gate. Pure scheduling — no GitOps here. Omit the whole block to
  # allow restarts at any time.
  schedule:
    windows:
      - type: Allow                 # Allow | Deny  (Deny always wins)
        cron: "0 2 * * *"           # 5-field cron marking each window start
        duration: 1h                # how long the window stays open
        timeZone: Europe/Berlin     # IANA zone, default UTC

  # ── WHO ──────────────────────────────────────────────────────────────
  # GitOps integration. Common gate config is flat; provider-specific knobs
  # live under a per-provider sub-block. Omit the block (or provider: None) to
  # let KICK self-gate.
  gitOps:
    provider: ArgoCD                # None (default) | Auto | ArgoCD | Flux
    requireReconciled: true         # wait until the owning app has finished
                                    # applying (Argo Synced / Flux Ready)

  # ── HOW ──────────────────────────────────────────────────────────────
  # Restart behaviour. Grouped so future knobs (strategy, maxConcurrent,
  # dryRun) are additive.
  restart:
    minInterval: 30s                # cooldown between restarts of one workload
```

### Minimal policy (self-gated, restart on any relevant change)

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: web
spec:
  discovery:
    workloadSelector:
      matchLabels: { app: web }
```

### `KickPolicy.spec` field reference

| Path | Type | Default | Description |
|------|------|---------|-------------|
| `suspend` | bool | `false` | Pause the policy: it matches nothing and issues no restarts until unset. |
| `discovery.workloadSelector` | LabelSelector | **required** | Workloads the policy manages. Explicit `{}` = every supported workload in the namespace (deliberate opt-in). |
| `discovery.dependencySelector` | LabelSelector | all | Which consumed Secret/ConfigMap changes trigger a restart. Also scopes freshness. |
| `schedule.windows[].type` | enum `Allow`/`Deny` | — | Window semantics. Deny overlapping now wins. |
| `schedule.windows[].cron` | string (5-field cron) | — | Marks each window start. |
| `schedule.windows[].duration` | duration | — | How long the window stays open from each start. |
| `schedule.windows[].timeZone` | IANA string | `UTC` | Zone used to evaluate `cron`. |
| `gitOps.provider` | enum | `None` | `None` self-gates; `Auto` detects the owning provider (falls back to self-gate); `ArgoCD`/`Flux` pin a provider. |
| `gitOps.requireReconciled` | bool | `true` | Correctness gate (not a schedule): wait until the owning application has finished applying (Argo `Synced` / Flux `Ready`) before restarting. |
| `restart.minInterval` | duration | `30s` | Minimum time between restarts of the same workload. |

### Window evaluation

The effective restart-window set is exactly the native `windows[]`. (Argo
`AppProject` sync windows are honored separately by the Argo provider gate; they
constrain the provider's *syncs*, not KICK's restart windows.) The gate is open
at time `t` when:

```
open(t)  ⇔  ( no Allow window exists  ∨  t ∈ some Allow window )
            ∧  t ∉ any Deny window
```

- **No windows at all** ⇒ always open.
- **Only Deny windows** ⇒ open except during a Deny window.
- **Any Allow window present** ⇒ closed by default; open only inside an Allow
  window (and never inside a Deny).

This refines today's behaviour, where the absence of an Allow window closed the
gate permanently. "Only Deny" now reads as the intuitive open-except-deny.

**Restart windows are not sync windows.** Three independent things must not be
conflated:

1. *When KICK restarts* — KICK rolls pods because a consumed dependency changed;
   timing is governed **only** by `spec.schedule.windows[]`.
2. *When the GitOps controller syncs* — Argo/Flux applying Git→cluster. Argo
   `AppProject` sync windows gate **that**, i.e. the provider's own apply
   cadence — not KICK.
3. *Whether the provider has finished applying* — `requireReconciled` (Synced /
   Ready). A correctness precondition ("don't restart against half-applied
   config"), not a schedule.

KICK does **not** reuse provider sync windows as restart windows: (1) and (2)
stay separate. Argo's sync windows are honored by the Argo provider gate (they
govern *its* syncs); KICK's restart timing is expressed solely in
`spec.schedule.windows[]`. There is deliberately no per-policy toggle coupling
the two.

### Provider differences (Argo CD vs Flux)

The common gate is identical for both providers — resolve exactly one owner, then
check it has finished applying — so `provider`, `requireReconciled`, and owner
resolution stay flat.

| KICK needs | Argo CD | Flux |
|------------|---------|------|
| Resolve owner | `argocd.argoproj.io/tracking-id` annotation (or `app.kubernetes.io/instance`) → `Application`; fallback = indexed Applications | `kustomize.toolkit.fluxcd.io/{name,namespace}` (or `helm.toolkit.fluxcd.io/...`) labels → `Kustomization` / `HelmRelease` |
| `requireReconciled` maps to | `Application.status.sync.status == Synced` | owner `Ready` condition `== True` |
| Provider sync windows | `AppProject.spec.syncWindows[]`, honored by the Argo provider gate (constrains the provider's sync, not KICK's restart) | **none** — no `SyncWindow`/`AppProject` concept; `spec.interval` is a frequency and `spec.suspend` a manual boolean, neither a schedule |

Because Flux has no time-window concept at all, Flux users rely on KICK's native
`schedule.windows[]` for any time gating — there is nothing provider-side to
honor. This is exactly why native windows live under `spec.schedule` and are
independent of `gitOps`.

### `KickPolicy.status` (controller-managed)

```yaml
status:
  observedGeneration: 4
  matchedWorkloads: 3
  blockedWorkloads: 1              # matched but currently gate-blocked
  conditions:
    - type: Ready
      status: "True"
      reason: Reconciled
```

---

## Target: `KickRequest` (controller-managed, one per target workload)

Users normally do **not** author `KickRequest`. KICK creates exactly one per
target workload and coalesces events into it. `spec` is intentionally tiny;
`status` is diagnostic — live cluster state is always authoritative.

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickRequest
metadata:
  name: deployment-web            # <lowercase-kind>-<name>
  namespace: shop
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment              # Deployment | StatefulSet | DaemonSet
    name: web
  policyRef:
    name: web                     # which KickPolicy created this request (audit)
status:
  phase: Succeeded                # see phase set below
  latestObservedDependencyChange: "2026-08-09T02:00:00Z"
  owner:                          # resolved GitOps owner, if any
    provider: ArgoCD
    kind: Application
    namespace: argocd
    name: web
    project: default
  gate:
    reason: Allowed               # Allowed | OutsideSchedule | OwnerUnknown |
    message: allowed by schedule and provider   # MultipleOwners | OutOfSync | SyncInProgress
    requeueAt: "2026-08-09T02:05:00Z"
  currentRollout:
    replicaSet: web-6f9c            # empty for StatefulSet/DaemonSet
    startedAt: "2026-08-09T02:00:03Z"
  conditions:
    - type: Progressing
      status: "False"
      reason: RestartCompleted
```

### `KickRequest` phase set

`Pending → WaitingForGate → WaitingForOwner → WaitingForApplicationSync →
WaitingForRollout → Executing → Succeeded`
plus terminal `NoLongerRequired` and `Failed`.

### `KickRequest.status.gate.reason` set

`Allowed`, `OutsideSchedule`, `OwnerUnknown`, `MultipleOwners`, `OutOfSync`,
`SyncInProgress`.

### Change vs current `KickRequest`

- **Add** `spec.policyRef.name` — records the owning policy for audit and for
  cheap reverse lookup (currently unresolvable without re-matching selectors).
- Everything else is unchanged and already stable.

---

## Rename / migration map (old → new)

```
spec.gitOps.schedule                         →  spec.schedule
spec.gitOps.schedule.source                  →  removed (Argo AppProject sync windows are always honored by the Argo provider gate)
spec.gitOps.schedule.windows[].kind          →  spec.schedule.windows[].type
spec.gitOps.schedule.windows[].schedule      →  spec.schedule.windows[].cron
spec.gitOps.schedule.windows[].duration      →  (unchanged) spec.schedule.windows[].duration
spec.gitOps.schedule.windows[].timeZone      →  (unchanged) spec.schedule.windows[].timeZone
spec.gitOps.provider                         →  (unchanged)
spec.gitOps.requireReconciled                →  (unchanged)
spec.minInterval                             →  spec.restart.minInterval
```

## Defaults & validation summary

- `suspend` defaults `false`.
- `discovery` is required and **must** set `workloadSelector`; an explicit empty
  `{}` opts in to "all supported workloads in the namespace". Omitting
  `workloadSelector` is a validation error — wide scope is never accidental.
- `discovery.dependencySelector` omitted ⇒ every discovered dependency triggers.
- `schedule` omitted ⇒ no time gate (always open).
- Argo CD `AppProject` sync windows are always honored by the Argo provider gate
  when the resolved owner is Argo CD; there is no per-policy toggle.
- `gitOps` omitted ⇒ `provider: None` ⇒ KICK self-gates. `Auto` detects the
  owner from the workload (Argo tracking annotation, Flux labels) and falls back
  to self-gate when no provider owns it.
- `restart.minInterval` defaults `30s`; pattern `^([0-9]+(ns|us|µs|ms|s|m|h))+$`.
- `windows[].cron` validated as a 5-field cron; `duration` as a Go duration.

## Invariants (must hold in every version)

- KICK writes exactly one thing to a workload: the standard
  `kubectl.kubernetes.io/restartedAt` annotation. The API never carries content
  hashes, injected env, or KICK-owned workload state — do not add such a field.
- `KickRequest` is diagnostic; live cluster state is authoritative. The API must
  never require a client to trust `status` over the live object.
- `imagePullSecrets` are never dependencies and never appear in the API.
- All CRDs stay cluster-agnostic and namespaced per target; no cluster-scoped
  policy is introduced without a new, explicitly-scoped kind.

## Versioning recommendation

- Make **all** of the above the shape of `v1alpha1` now (pre-release, so the
  break is free).
- Ship the first tagged release as `v1alpha1` with this shape frozen.
- Promote to `v1` at GA with **no field changes**: serve `v1alpha1` and `v1`
  side by side with `v1` as the storage version. Because the schemas are
  identical the conversion is the identity function — no conversion webhook
  logic is required.

## Open questions

1. `gitOps.provider: Flux` is reserved but unimplemented — keep in the enum now
   (so adding Flux later is additive) or omit until built? *Recommend: keep.*
2. Should `restart` grow a `strategy` (e.g. `RolloutRestart` vs future
   `Recreate`)? Not now — but the group exists so it is additive.
3. Do we ever need `schedule` on a `KickRequest` override? Assumed no; policy is
   the only source of scheduling.
4. `restart.dryRun` (report what would restart without acting) is high value for
   adoption. It is additive under `restart`, so it can land later without a
   break — build it when there is demand.
5. Should `Auto` that finds *no* owner self-gate (proposed) or block? Proposed
   self-gate keeps unmanaged workloads working; revisit if it surprises users.
6. ~~Is `gitOps.argoCD.honorSyncWindows` worth the conceptual coupling?~~
   **Resolved: dropped.** A restart is not a sync; restart timing lives purely
   in `spec.schedule`. Argo `AppProject` sync windows remain honored by the Argo
   provider gate (they govern the provider's own sync), with no per-policy knob.

## Follow-up implementation tasks (not this doc)

1. Edit `api/v1alpha1/kickpolicy_types.go`: add `spec.suspend`; introduce
   `ScheduleSpec` + `RestartSpec` at `spec` level; drop `Schedule` from
   `KickPolicyGitOpsSpec`; rename window fields (`Kind→Type`, `Schedule→Cron`);
   add provider sub-block `gitOps.argoCD.honorSyncWindows`; require
   `discovery.workloadSelector`.
2. Add `PolicyRef` to `KickRequestSpec`.
3. `make codegen`; sync CRDs into `charts/kick/crds/`.
4. Update controller reads (`internal/controller`, `internal/schedule`,
   `internal/policy`), envtest, all e2e scenario YAMLs, samples, traceability.
5. Regenerate AI docs (`make docs-gen`); run `make test`, `make lint`,
   `make test-e2e`.
