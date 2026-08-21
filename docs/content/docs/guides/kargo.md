---
title: Kargo
weight: 43
description: Configure KICK Kargo gating with Stage-annotation and promotion-state examples.
---

KICK can gate restarts on Kargo Stage promotion state when `provider: Kargo` is set.

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

- If promotion is active for the resolved Stage, KICK blocks restart with gate reason.
- After promotion completes, KICK proceeds to Argo CD gate checks.

Proof scenarios:

- Promotion blocks restart: [KICK-E2E-068](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-068-kargo-promotion-blocks-restart)
- Restart after promotion: [KICK-E2E-069](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-069-kargo-restart-after-promotion)

## Safety cases

- Missing annotation blocks: [KICK-E2E-070](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-070-kargo-unannotated-application-blocks)
- Ambiguous stage list blocks: [KICK-E2E-071](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-071-kargo-ambiguous-stage-blocks)

## Feature mapping

- Kargo stage promotion gate: [KICK-FEAT-025](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
