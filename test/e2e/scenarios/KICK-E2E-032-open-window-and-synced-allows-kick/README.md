# KICK-E2E-032 - Open Window And Synced Allows Kick

## Behavior under test
Primary behavior: open window and synced allows kick.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies time-window enforcement so changes do not trigger restarts outside approved maintenance periods.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- windows.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-032
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-032
- **Kind**: AppProject, **Name**: kick-e2e-032, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-032, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-032
- **Kind**: Deployment, **Name**: app-032, Namespace: kick-e2e-032

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-032
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-008, KICK-FEAT-012
