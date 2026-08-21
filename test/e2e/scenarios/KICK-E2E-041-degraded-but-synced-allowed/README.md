# KICK-E2E-041 - Degraded But Synced Allowed

## Behavior under test
Primary behavior: degraded but synced allowed.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies gate timing so restarts are deferred until controller state is stable and authoritative.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/app.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-041
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-041
- **Kind**: Application, **Name**: kick-e2e-041, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-041
- **Kind**: Deployment, **Name**: app-041, Namespace: kick-e2e-041
- **Kind**: Deployment, **Name**: broken-041, Namespace: kick-e2e-041

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-041
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-013
