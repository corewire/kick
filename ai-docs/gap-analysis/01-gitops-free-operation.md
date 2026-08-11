# Gap 1: Running KICK without Argo CD or Flux

> **Outcome: closed.** No code change was needed. The comparison page and the
> new [Running without GitOps](../../docs/content/docs/guides/without-gitops.md)
> guide were corrected, and `KICK-E2E-058` proves it. Tracked as
> `KICK-FEAT-022`.

> **Comparison bullet.** *You do not run Argo CD or Flux. Without GitOps gating,
> KICK's main advantage does not apply, and Reloader is far better proven.*

## Status

**The functional half of this bullet is already false.** KICK runs fine without
any GitOps tool. What remains true is the positioning half: without a GitOps
gate, KICK's headline differentiator is gone, and Reloader has far more
production mileage.

This is therefore mostly a documentation and proof gap, not a feature gap.

## Proof

**KICK does not require a GitOps provider.** `KickPolicy.spec.gitOps.provider`
defaults to `None`:

```go
// Provider selects the GitOps gate. "None" (the default) restarts without
// consulting any GitOps tool, gated only by any native schedule windows.
// +kubebuilder:default:=None
// +kubebuilder:validation:Enum=None;Auto;ArgoCD;Flux
Provider KickPolicyProvider `json:"provider,omitempty"`
```

[api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go#L61-L73)

The reconciler treats an unset or `None` provider as "KICK gates on its own",
skipping ownership resolution entirely:

```go
// policyGitOpsGated reports whether restart decisions defer to a GitOps
// provider. An unset or "None" provider means KICK gates on its own.
```

[internal/controller/kickrequest_controller.go](internal/controller/kickrequest_controller.go#L442-L455)

**KICK-native gating exists and is provider-independent.** `spec.schedule.windows`
are cron-based allow/deny windows evaluated by
[internal/schedule](internal/schedule), with no GitOps owner involved:

[api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go#L40-L59),
[internal/controller/kickrequest_controller.go](internal/controller/kickrequest_controller.go#L255-L290)

**What KICK still offers with `provider: None`:**

| Capability | Reloader | KICK without GitOps |
|---|---|---|
| Opt-in without touching the workload | No (annotation required) | Yes (`KickPolicy` selectors) |
| Restart suppressed if something else already restarted | No (event-driven) | Yes (freshness comparison) |
| Maintenance windows | No | Yes (cron allow/deny) |
| Per-workload minimum interval | No — upstream lists "no per-resource reload rate limiting" as a known gap in `CLAUDE.md` | Yes (`spec.restart.minInterval`) |
| Durable, inspectable restart requests | No | Yes (`KickRequest`) |

Reloader's own `CLAUDE.md` documents the rate-limiting gap:

> **No per-resource reload rate limiting**: A rapid-fire ConfigMap update (e.g.,
> from a CI pipeline) can trigger many restarts. A cooldown window per resource
> would help.

## Impact

The current wording sends away users who would be well served by KICK: teams
with no GitOps tool who still want maintenance windows, no workload annotations,
and idempotent restarts. It also under-sells `spec.schedule`, which is a genuine
Reloader/Wave gap in the other direction.

## Plan

### A. Correct the comparison page (required)

Rewrite the bullet to separate the two claims:

- Keep: *"Reloader is far better proven"* — a maturity statement, honest and
  unchanged.
- Remove: the implication that KICK needs Argo CD or Flux.
- Add: a short "KICK without GitOps" note pointing at `provider: None` plus
  `spec.schedule.windows`.

Add a row to the "At a glance" table for **Restart rate limiting**
(Reloader: none, Wave: rate limit, KICK: `minInterval` + cron windows).

### B. Document the standalone mode (required)

New page `docs/content/docs/guides/without-gitops.md`:

- Minimal `KickPolicy` with `provider: None`.
- Cron window example (nightly 02:00–04:00 allow window).
- `minInterval` example for CI-driven ConfigMap churn.
- Explicit statement that no Argo CD or Flux CRDs need to exist.

Feed into `make docs-gen` so `llms.txt`/`llms-full.txt` pick it up.

### C. Prove it in e2e (required)

The claim "works without GitOps" must not rest on a code reading. Add e2e
scenarios that run in a cluster with **no Argo CD and no Flux CRDs installed**:

| ID | Scenario |
|---|---|
| KICK-E2E-057 | `provider: None`, ConfigMap changes → exactly one rollout, no gate wait |
| KICK-E2E-058 | `provider: None` + deny window active → no rollout; window closes → exactly one rollout |
| KICK-E2E-059 | `provider: Auto` with no GitOps CRDs present → documented, deterministic behaviour (see open question) |

Follow [specs/20-e2e-suite-conventions.md](ai-docs/kick-operator-specs/kick-specs/specs/20-e2e-suite-conventions.md):
stable-ID scenario directory, `trace.yaml`, central matrix entry, assert exact
rollout counts.

### D. Verify `provider: Auto` degradation (required)

`Auto` currently resolves through the provider adapters. In a cluster with no
Argo CD or Flux CRDs, the adapters return `GateProviderUnavailable`
([internal/gitops/argocd/provider.go](internal/gitops/argocd/provider.go#L62),
[internal/gitops/flux/provider.go](internal/gitops/flux/provider.go#L59)), which
the reconciler treats as a blocking gate
([internal/controller/kickrequest_controller.go](internal/controller/kickrequest_controller.go#L717)).

Decide and document: does `Auto` in a GitOps-free cluster mean **block forever**
(safe, current behaviour) or **fall back to `None`** (convenient, but silently
removes a safety gate if a provider is merely temporarily unreachable)?

Recommendation: keep blocking, but surface it as a clear
`ProviderUnavailable` condition on the `KickPolicy` status rather than only on
the `KickRequest`, so the misconfiguration is visible without inspecting
requests.

## Work breakdown

| Item | Type | Notes |
|---|---|---|
| Rewrite comparison bullet + table row | Docs | No code |
| `without-gitops.md` guide | Docs | No code |
| E2E 057–059 | Test | New chainsaw scenarios, GitOps-free kind cluster |
| `Auto` degradation decision + policy condition | Code | Small; touches policy status only |

## Traceability

| ID | Name | Unit | Envtest | E2E |
|---|---|---|---|---|
| KICK-FEAT-021 | GitOps-free operation with `provider: None` | required | required | required |
| KICK-FEAT-022 | Deterministic `provider: Auto` behaviour without GitOps CRDs | required | required | required |

E2E: KICK-E2E-057, KICK-E2E-058, KICK-E2E-059.

## Risks

- A GitOps-free e2e lane means a second kind cluster profile in CI. Check
  whether the existing suite installs Argo CD CRDs unconditionally before
  estimating this.
- Loosening `Auto` to fall back to `None` would be a silent safety regression.
  Do not do it without an explicit opt-in field.

## Open questions

1. Should `spec.gitOps.provider: None` be renamed to something less
   accidental-looking, given it is the default? (e.g. `Disabled`)
2. Does the current e2e harness install Argo CD CRDs globally? If so, the
   GitOps-free scenarios need their own cluster profile.
