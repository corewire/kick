---
title: Argo Rollouts
weight: 42
description: Configure KICK for Argo Rollouts with scenario-backed restart examples.
---

KICK can restart Argo Rollouts workloads when the Rollouts integration is enabled.

![Argo Rollouts restart path](/images/argo-rollouts-restart.drawio.svg)

## Enable the integration

Set one of the supported toggles:

- Helm value: `integrations.argoRollouts.enabled: true`
- Manager flag: `--enable-argo-rollouts=true`

Proof: [Configuration reference](../reference/configuration/).

## Minimal policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: kick-e2e-060
spec:
  discovery:
    workloadSelector: {}
```

Proof scenario: [KICK-E2E-060](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-060-canary-restart-does-not-rerun-steps).

Raw test manifest (implementation detail): [resources.yaml](https://github.com/corewire/kick/blob/main/test/e2e/scenarios/KICK-E2E-060-canary-restart-does-not-rerun-steps/resources.yaml).

## Restart behavior that matters

KICK restarts Rollouts through `spec.restartAt`, not by patching pod-template annotations.
That avoids re-running canary/blue-green steps from step zero.

Proof scenario:

- [KICK-E2E-060](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-060-canary-restart-does-not-rerun-steps)

## WorkloadRef behavior

For a Rollout with `workloadRef`, KICK targets the referenced Deployment when that
Deployment owns the dependency.

Proof scenario:

- [KICK-E2E-063](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-063-workloadref-rollout-restarts-referenced-deployment)

## More proven examples

- Canary restart keeps step index: [KICK-E2E-060](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-060-canary-restart-does-not-rerun-steps)
- Blue/green active service remains stable: [KICK-E2E-061](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-061-blue-green-restart-keeps-active-service)
- Rollout completion gates KickRequest: [KICK-E2E-062](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-062-rollout-completion-gates-request)

## Feature mapping

- Argo Rollout workload restarts: [KICK-FEAT-024](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
