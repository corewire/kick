# Gap 5: Adoption without an initial restart

> **Outcome: closed.** Adoption already restarts nothing, which is now
> documented rather than fixed. `spec.dryRun` was added so the decision can be
> previewed (`KICK-FEAT-021`). Holding pods `Pending` is declined: it needs an
> admission webhook in the pod path.

> **Comparison bullet.** *You want to avoid the initial restart when adopting
> the tool. Wave's mutating webhook sets the first hash without a rollout, and
> holds pods `Pending` while a required Secret is missing.*

## Status

This bullet bundles two unrelated Wave features, and KICK's position differs
sharply between them:

| Half of the bullet | KICK today |
|---|---|
| Avoid the initial restart | **Already solved**, by design, without a webhook |
| Hold pods `Pending` while a source is missing | **Not supported**, and would require an admission webhook |

The first half of the bullet is misleading and should be corrected. The second
half is real and needs a build-or-decline decision.

## Proof

### KICK already avoids the initial restart

Wave and Reloader need a webhook or an event to avoid a first-adoption rollout
because their trigger is *state written into the workload* (a hash) or *an
observed event*. KICK's trigger is a **time comparison**, so a baseline
observation simply lands in the past:

```go
// A first observation establishes a baseline; anchor it to the source's
// own creation time (deterministic) rather than the wall-clock moment KICK
// happened to see it. A dependency created alongside its workload is then
// never "newer" than the workload's rollout, so baselines do not trigger a
// spurious restart, while a later-created (previously missing) source is.
baselineTime := observedAt
if !createdAt.IsZero() {
	baselineTime = createdAt.UTC()
}
```

[internal/observation/observer.go](internal/observation/observer.go#L74-L98)

The freshness evaluator only requires a restart when the latest relevant change
is strictly after the running rollout's start:

```go
decision.RestartRequired = latest.After(state.StartedAt)
```

[internal/freshness/evaluator.go](internal/freshness/evaluator.go#L66)

For an existing cluster, every `Secret`/`ConfigMap` has a `creationTimestamp`
older than the current rollout in the normal case, so installing KICK restarts
nothing. No webhook, no annotation, no hash.

Compare Wave, which needs an admission webhook to reach the same outcome:

> Wave can update Deployments on creation/update using Mutating Webhooks. This
> will prevent triggering restarts when adding the hash annotation initially.

`wave-k8s/wave`, `README.md` — "Webhooks"

### The `Pending` half is genuinely Wave-only

> Additionally, Wave will prevent scheduling of pods which lack any of their
> required Secrets or ConfigMaps to reduce stress on the cluster. Pods will stay
> in state `Pending` instead of `ContainerCreating`. When required
> Secrets/ConfigMaps have been created Wave will restore the scheduler and add
> the config hash without requiring any restarts.

`wave-k8s/wave`, `README.md` — "Webhooks"

KICK ships no admission webhook at all: there are no webhook manifests under
[config/](config) and no cert-manager wiring.

## Known adoption edge case

The baseline design has a deliberate trade-off that is currently undocumented.

A workload that was **already stale before KICK was installed** — its `Secret`
was updated after the last rollout, but before KICK started observing — is *not*
detected. The baseline anchors on `creationTimestamp` (old), not on the last
update, so the comparison against rollout start reports "fresh".

This is the correct default (no adoption stampede), but it is a silent gap that
users should be told about, along with the manual remedy
(`kubectl rollout restart`).

`metadata.managedFields[].time` could in principle recover the real last-write
time, which is exactly what
[specs/17-open-questions.md](ai-docs/kick-operator-specs/kick-specs/specs/17-open-questions.md)
question 2 is about. Any change here would turn a quiet adoption into a noisy
one, so it must be opt-in if it is ever built.

## Plan A — Correct and document (required, no code)

1. Rewrite the comparison bullet. Keep only the `Pending` half as a reason to
   choose Wave. Move "no initial restart" to the **"What is different about
   KICK"** section: KICK achieves it *without* an admission webhook, which is a
   meaningful operational advantage — Wave's approach means an outage of the
   webhook can block pod creation.
2. Add an "Adopting KICK" section to the docs covering: nothing restarts on
   install; already-stale workloads are not retroactively detected; use
   `kubectl rollout restart` once, or a `suspend: true` policy plus manual
   verification, to establish a clean starting point.
3. Add an e2e scenario that proves the claim rather than asserting it.

## Plan B — Adoption dry-run mode (recommended, small)

The real adoption anxiety is "what will this thing do to my cluster?" — which
`suspend` only half answers, because a suspended policy matches nothing and so
reports nothing.

Proposal: `KickPolicy.spec.dryRun` (bool). When true, the policy matches
workloads and creates `KickRequest`s that evaluate freshness and gates
completely, but the executor stops before patching, parking the request in a
terminal `DryRun` phase with the full decision recorded.

This gives operators an exact preview list. It is cheap: one field, one branch
in [internal/executor/restart.go](internal/executor/restart.go), one phase.

It also composes with Plan A's edge case — a dry-run policy surfaces nothing for
pre-existing staleness, which makes the trade-off concrete instead of abstract.

## Plan C — Scheduler gate for missing sources (build-or-decline)

Wave's `Pending` behaviour requires a mutating admission webhook on Pods.

What it would take:

1. A `MutatingWebhookConfiguration` on Pod `CREATE`, namespace-selected to
   policy-managed namespaces.
2. Certificate management (cert-manager, or self-signed rotation in-operator).
3. On admission: resolve the pod's `Secret`/`ConfigMap` references, and if any
   required (non-optional) source is missing, set an unsatisfiable
   `spec.schedulerName` or a blocking node selector so the pod stays `Pending`.
4. A controller that removes the gate once the sources exist.

Reasons to decline:

- **Blast radius.** A Pod-`CREATE` webhook is the single highest-risk extension
  point in Kubernetes. `failurePolicy: Ignore` makes the feature unreliable;
  `Fail` makes a KICK outage a cluster-wide pod-creation outage. For a young
  project this is a bad trade.
- **It is not KICK's problem.** This is a scheduling-efficiency feature, not a
  configuration-freshness feature. It shares no machinery with the rest of KICK.
- **Kubernetes has a native answer.** Pod scheduling gates
  (`spec.schedulingGates`, GA since 1.30) are the supported mechanism and would
  still need a webhook to apply them — but the correct owner of that webhook is
  a scheduling tool, not a restart tool.
- **AGENTS.md constraint.** "KICK MUST NOT inject dependency hashes,
  environment variables, or KICK-owned state annotations into workloads." A
  scheduling gate is exactly that kind of injection into the pod spec.

**Recommendation: decline, and say so explicitly on the comparison page.** If a
webhook is ever added for another reason, revisit.

## Traceability

| ID | Name | Unit | Envtest | E2E |
|---|---|---|---|---|
| KICK-FEAT-031 | Installing KICK on an existing cluster restarts nothing | required | required | required |
| KICK-FEAT-032 | Dry-run policies evaluate fully without patching workloads | required | required | required |

| ID | Scenario |
|---|---|
| KICK-E2E-072 | Pre-existing workloads + sources, KICK installed → zero rollouts over a stable observation period |
| KICK-E2E-073 | Operator restarted with a warm store → still zero rollouts |
| KICK-E2E-074 | `dryRun: true`, source changes → request reaches `DryRun`, workload untouched |

KICK-E2E-072 must assert an **exact rollout count of zero**, per
[specs/20-e2e-suite-conventions.md](ai-docs/kick-operator-specs/kick-specs/specs/20-e2e-suite-conventions.md).
"Ready" is not sufficient evidence.

## Risks

- The zero-restart property depends on `creationTimestamp` being older than the
  rollout. A source recreated (deleted and re-applied) by a GitOps sync gets a
  fresh `creationTimestamp` and *will* look new. Confirm whether Argo CD prune +
  re-create can produce this, since it would turn a resync into a restart
  stampede. This deserves its own e2e scenario regardless of what else is built.
- `dryRun` adds a terminal phase to the `KickRequest` state machine; check it
  against [specs/09-restart-request-api.md](ai-docs/kick-operator-specs/kick-specs/specs/09-restart-request-api.md)
  before implementing.

## Open questions

1. Should the baseline optionally use `managedFields[].time` to catch
   pre-existing staleness on adoption? (Ties into open question 2 in
   `specs/17-open-questions.md`.)
2. Does a GitOps delete-and-recreate of a `Secret` reset `creationTimestamp` in
   practice, and if so, is that a bug or acceptable?
3. Is `dryRun` a policy field or a whole-operator mode? Policy field is more
   useful; operator mode is easier to reason about during a first install.
