---
title: Argo CD
weight: 41
description: Configure Argo CD gating with concrete, scenario-backed examples.
---

KICK can gate restarts on Argo CD ownership, sync state, and AppProject sync windows.

![Argo CD gate flow](/images/gitops-gate.drawio.svg)

## Minimal policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: team-a
spec:
  discovery:
    workloadSelector: {}
  gitOps:
    provider: ArgoCD
```

Proof: provider contract and fields in [KickPolicy reference](../reference/kickpolicy/).

## Ownership resolution order

1. Tracking annotation on workload: `argocd.argoproj.io/tracking-id`.
2. Fallback owner search when annotation is missing or invalid.
3. Block if owner is missing or ambiguous.

Proof scenarios:

- Annotation owner in Argo CD namespace: [KICK-E2E-024](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-024-annotation-owner-in-argo-cd-namespace)
- Annotation owner in other namespace: [KICK-E2E-025](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-025-annotation-owner-in-other-namespace)
- Invalid annotation fallback: [KICK-E2E-026](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-026-invalid-annotation-fallback)
- No owner blocks: [KICK-E2E-028](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-028-no-owner-blocks)
- Multiple owners block: [KICK-E2E-029](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-029-multiple-owners-block)

## Gate checks

- `Application` sync state: waits while OutOfSync or actively syncing.
- `AppProject` sync windows: restart only when window allows.

Proof scenarios:

- Open window allows restart: [KICK-E2E-032](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-032-open-window-and-synced-allows-kick)
- Closed/deny windows wait: [KICK-E2E-033](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-033-closed-allow-window-waits), [KICK-E2E-034](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-034-deny-window-waits)
- OutOfSync/syncing waits: [KICK-E2E-037](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-037-outofsync-waits), [KICK-E2E-038](https://github.com/corewire/kick/tree/main/test/e2e/scenarios/KICK-E2E-038-active-sync-waits)

## Feature mapping

- Provider-neutral gate: [KICK-FEAT-008](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
- Argo owner + fallback + ambiguity: [KICK-FEAT-009](https://github.com/corewire/kick/blob/main/traceability/features.yaml), [KICK-FEAT-010](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
- AppProject windows + sync wait: [KICK-FEAT-012](https://github.com/corewire/kick/blob/main/traceability/features.yaml), [KICK-FEAT-013](https://github.com/corewire/kick/blob/main/traceability/features.yaml)