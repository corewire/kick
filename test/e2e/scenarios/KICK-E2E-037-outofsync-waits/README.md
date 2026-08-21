# KICK-E2E-037 - Outofsync Waits

## Behavior under test
Primary behavior: outofsync waits.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies gate timing so restarts are deferred until controller state is stable and authoritative.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-037
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-037
- **Kind**: Application, **Name**: kick-e2e-037, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-037
- **Kind**: Deployment, **Name**: app-037, Namespace: kick-e2e-037
- **Kind**: ConfigMap, **Name**: drift-037, Namespace: kick-e2e-037

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-037
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-008, KICK-FEAT-013
