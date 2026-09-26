# Kargo verification-aware restarts

Status: implemented as [KICK-FEAT-029](../../traceability/features.yaml).
Verification waits live in
[internal/gitops/kargo/verification.go](../../internal/gitops/kargo/verification.go).
Optional reverification lives in
[internal/controller/kargo_reverify.go](../../internal/controller/kargo_reverify.go).

Target: Kargo v1.11.4
([e2e installer](../../test/e2e/setup/kargo/install-kargo.sh),
[Stage types](https://github.com/akuity/kargo/blob/v1.11.4/api/v1alpha1/stage_types.go)).

## Gap

Promotion and verification are separate. Reusable `PromotionTask`s are promotion
steps only. After a successful Promotion, configured verification starts once
referenced Argo CD Applications are Healthy, then runs as `AnalysisRun`s. A
Stage has no `status.phase` in v1.11.4
([verification guide](https://docs.kargo.io/user-guide/how-to-guides/verification/),
[Stage types](https://github.com/akuity/kargo/blob/v1.11.4/api/v1alpha1/stage_types.go)).

KICK blocks on `status.currentPromotion` and non-terminal Promotions, then
delegates to Argo CD. It ignores verification
([Kargo adapter](../../internal/gitops/kargo/provider.go),
[Argo CD gate](../../internal/gitops/argocd/provider.go)).

## Decision

Activity controls timing. Freshness controls necessity
([freshness evaluation](../../internal/controller/kickrequest_controller.go)).
Verification outcome never replaces freshness.

Active means any of:

- `status.currentPromotion.name` is set, or a Promotion with `spec.stage` has a
  phase outside `Succeeded`, `Failed`, `Errored`, and `Aborted`. Empty phase is
  active. This is the current rule.
- `status.freightHistory[0].verificationHistory` contains a phase outside
  `Successful`, `Failed`, `Error`, `Aborted`, and `Inconclusive`. `Pending`,
  `Running`, and empty are active. This matches Kargo `IsTerminal`
  ([v1.11.4 types](https://github.com/akuity/kargo/blob/v1.11.4/api/v1alpha1/stage_types.go)).
- `spec.verification` is configured, current Freight has no verification record,
  and `status.lastPromotion` for that Freight succeeded. This covers the delay
  before the first record, including the Application-health wait. A missing
  `AnalysisRun` alone does not mean settled.

`status.health` is not a restart gate. After verification is terminal, Unhealthy
can be the stale-Secret case and must remain restartable.

| State | Workload | Action |
|---|---|---|
| Promotion or verification active, or expected verification not yet recorded | Either | Wait |
| Settled; verification `Successful`, `Failed`, `Error`, `Aborted`, `Inconclusive`, or not configured | Stale | Restart if other gates permit |
| Settled; any of those outcomes | Fresh | No restart |

Other rules:

- Missing, ambiguous, or unreadable Stage, Promotion, or Freight state blocks.
- Re-read Kargo state and freshness immediately before restart.
- Keep Argo CD gates, policy windows, rate limits, and dry-run.
- Terminal failure does not permanently block a corrective restart. Success does
  not hide staleness.
- Soak time and downstream eligibility are not KICK gates
  ([soak times](https://docs.kargo.io/user-guide/how-to-guides/verification/)).
- Stages without `spec.verification` keep the current promotion-only gate.
  Implicit health verification controls downstream eligibility, not KICK waits
  ([implicit verification](https://docs.kargo.io/user-guide/how-to-guides/verification/)).

Example: a Secret rotates during promotion, but sync leaves the Pod template
unchanged. Verification may pass or fail because the Pod still has old
credentials. KICK waits until that verification is terminal, rechecks freshness,
and restarts once if the workload is still stale. No second restart when
promotion already replaced the Pods
([freshness](../../docs/content/docs/concepts/freshness.md)).

## Optional reverification

Opt-in, default off. Do not create a Promotion. Kargo reruns verification when
`kargo.akuity.io/reverify` changes to the current verification ID, or to
`VerificationRequest` JSON containing it. Kargo refuses without current Freight
and current verification info
([annotation](https://github.com/akuity/kargo/blob/v1.11.4/api/v1alpha1/annotations.go),
[ReverifyStageFreight](https://github.com/akuity/kargo/blob/v1.11.4/pkg/api/stage.go)).

1. Persist Stage, Freight-collection ID, and verification ID before restart.
2. Complete the KICK rollout and confirm freshness.
3. Re-read the Stage. Patch only when all three identifiers are unchanged,
   promotion and verification are inactive, and that verification ID has no
   recorded request.
4. Persist the requested verification ID.

No KICK restart, dry-run, missing verification info, changed Freight, or new
Kargo activity means no patch. A failed patch or failed reverification must not
repeat the restart. The same annotation ID does not retrigger while present;
setting it again after Kargo clears it does. Recovery must inspect recorded
request state and verification history before patching
([reverify predicate](https://github.com/akuity/kargo/blob/v1.11.4/pkg/kargo/kargo.go)).

Workloads sharing a Stage share one verification. Coalesce to one request per
verification ID.

## Tests

Unit tests cover the gate matrix. End-to-end coverage is
[KICK-E2E-074](../../test/e2e/scenarios/KICK-E2E-074-kargo-verification-blocks-restart),
[KICK-E2E-075](../../test/e2e/scenarios/KICK-E2E-075-kargo-failed-verification-restarts),
and [KICK-E2E-076](../../test/e2e/scenarios/KICK-E2E-076-kargo-reverify-after-restart)
([scenario matrix](../../traceability/e2e-scenarios.yaml)).
The installer enables `controller.rollouts.integrationEnabled` after Argo Rollouts CRDs exist.

- Active promotion, non-terminal verification, or configured verification not yet
  recorded: zero restarts.
- Terminal `Successful` or `Failed` verification with a stale workload: one
  restart, including failure caused by an unconsumed Secret.
- Fresh workload, including one refreshed by promotion: zero restarts.
- No `spec.verification`: current promotion gate only.
- Reverification disabled, dry-run, changed Freight, or no verification info:
  zero patches.
- Enabled path: one annotation patch after rollout, keyed by verification ID.
- Retry after Kargo clears the annotation: zero duplicate patches.
- Failed patch or failed reverification: zero additional restarts.

## Resolved

- Policy field: `spec.gitOps.reverifyAfterRestart` (default false).
- Request status: `status.kargoReverification` with `Pending`, `Requested`, or
  `Skipped`. Active verification reuses `GateOwnerReconciling`.
- RBAC: patch on Stages only. Promotions stay get/list/watch
  ([ClusterRole](../../config/rbac/role.yaml)).
- Freight history can still advance between the pre-restart read and the
  restart patch. Live reads do not make those writes atomic. A changed
  collection ID skips the annotation.