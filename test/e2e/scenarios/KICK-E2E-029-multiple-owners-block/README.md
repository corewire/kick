# KICK-E2E-029 - Multiple Owners Block

## Behavior under test
Primary behavior: multiple owners block.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies fail-closed behavior for ambiguous or missing ownership signals, preventing automatic restarts on uncertain targets.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- manifests/secret.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-029
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-029
- **Kind**: Application, **Name**: kick-e2e-029-a, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-029-b, Namespace: argocd
- **Kind**: Deployment, **Name**: app-029, Namespace: kick-e2e-029
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-029

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-029
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-010
