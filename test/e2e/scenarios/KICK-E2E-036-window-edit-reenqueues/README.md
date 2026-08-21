# KICK-E2E-036 - Window Edit Reenqueues

## Behavior under test
Primary behavior: window edit reenqueues.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies time-window enforcement so changes do not trigger restarts outside approved maintenance periods.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- windows-closed.yaml
- windows-open.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-036
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-036
- **Kind**: AppProject, **Name**: kick-e2e-036, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-036, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-036
- **Kind**: Deployment, **Name**: app-036, Namespace: kick-e2e-036

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-036
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-012
