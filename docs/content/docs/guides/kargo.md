---
title: Kargo
weight: 43
description: Configure KICK Kargo gating with Stage, promotion, and verification examples.
---

KICK can gate restarts on Kargo Stage promotion and verification activity when `provider: Kargo` is set.

![Kargo promotion gate](/images/kargo-promotion-gate.drawio.svg)

## Enable the integration

Set one of the supported toggles:

- Helm value: `integrations.kargo.enabled: true`
- Manager flag: `--enable-kargo=true`

Proof: [Configuration reference](../reference/configuration/).

## Minimal policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: kick-e2e-068
spec:
  discovery:
    workloadSelector: {}
  gitOps:
    provider: Kargo
```

Proof example: [KICK-E2E-068 resources](https://github.com/corewire/kick/blob/main/test/e2e/scenarios/KICK-E2E-068-kargo-promotion-blocks-restart/resources.yaml).

## Required Argo CD Application annotation

KICK resolves the authorized Stage from this annotation on the Argo CD Application:

```yaml
metadata:
  annotations:
    kargo.akuity.io/authorized-stage: kick-e2e-068:prod
```

Proof example: [KICK-E2E-068 Application](https://github.com/corewire/kick/blob/main/test/e2e/scenarios/KICK-E2E-068-kargo-promotion-blocks-restart/resources.yaml).

## Gate behavior

- Active Promotion or verification blocks the restart. Empty and non-terminal verification phases wait. `Successful`, `Failed`, `Error`, `Aborted`, and `Inconclusive` do not.
- A configured verification with no record yet also waits when `status.lastPromotion` for the current Freight succeeded.
- `status.health` is not a gate. A failed verification can still be the stale-Secret case.
- After that activity settles, KICK proceeds to the Argo CD gate. Freshness still decides whether a restart is necessary.
- Stages without `spec.verification` keep the promotion-only gate.

Proof: [verification gate](https://github.com/corewire/kick/blob/main/internal/gitops/kargo/verification.go), [unit tests](https://github.com/corewire/kick/blob/main/internal/gitops/kargo/verification_test.go).

Proof scenarios:

- Promotion blocks restart: [KICK-E2E-068](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-068-kargo-promotion-blocks-restart)
- Restart after promotion: [KICK-E2E-069](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-069-kargo-restart-after-promotion)
- Running verification blocks, then one restart: [KICK-E2E-074](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-074-kargo-verification-blocks-restart)
- Failed verification still restarts a stale workload: [KICK-E2E-075](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-075-kargo-failed-verification-restarts)

The e2e installer enables Kargo's Rollouts integration and installs Argo Rollouts first. Kargo checks for AnalysisRun CRDs once at startup and otherwise runs as if verification were disabled ([install-kargo.sh](https://github.com/corewire/kick/blob/main/test/e2e/setup/kargo/install-kargo.sh)).

## Optional reverification

`spec.gitOps.reverifyAfterRestart` defaults to false. When true, KICK stores the Stage, Freight-collection ID, and verification ID before the restart, then patches `kargo.akuity.io/reverify` with `{"id":"<verification id>"}` only after the rollout is fresh and those IDs are unchanged. It does not create a Promotion. A failed patch does not restart again.

The pre-restart read and the restart patch are not atomic. Freight history can advance in between; KICK then skips the annotation.

```yaml
spec:
  gitOps:
    provider: Kargo
    reverifyAfterRestart: true
```

Proof: [follow-up](https://github.com/corewire/kick/blob/main/internal/controller/kargo_reverify.go), [KICK-E2E-076](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-076-kargo-reverify-after-restart).

## Safety cases

- Missing annotation blocks: [KICK-E2E-070](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-070-kargo-unannotated-application-blocks)
- Ambiguous stage list blocks: [KICK-E2E-071](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-071-kargo-ambiguous-stage-blocks)

## Feature mapping

- Kargo stage promotion gate: [KICK-FEAT-025](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
- Kargo verification-aware restarts: [KICK-FEAT-029](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
