# KICK-E2E-040 - Unrelated Argo Sync Still Requires Kick

## Behavior under test
Primary behavior: unrelated argo sync still requires kick.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies gate timing so restarts are deferred until controller state is stable and authoritative.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/app.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-040
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-040
- **Kind**: Application, **Name**: kick-e2e-040, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-040
- **Kind**: ConfigMap, **Name**: unrelated-040, Namespace: kick-e2e-040
- **Kind**: Deployment, **Name**: app-040, Namespace: kick-e2e-040

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/app.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-040
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-013
