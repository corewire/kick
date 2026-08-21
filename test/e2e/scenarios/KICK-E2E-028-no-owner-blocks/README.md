# KICK-E2E-028 - No Owner Blocks

## Behavior under test
Primary behavior: no owner blocks.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies fail-closed behavior for ambiguous or missing ownership signals, preventing automatic restarts on uncertain targets.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-028
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-028
- **Kind**: Application, **Name**: kick-e2e-028-unrelated, Namespace: argocd
- **Kind**: Deployment, **Name**: app-028, Namespace: kick-e2e-028
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-028

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-028
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-010
