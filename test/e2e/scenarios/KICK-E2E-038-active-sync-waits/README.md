# KICK-E2E-038 - Active Sync Waits

Behavior under test: active sync waits.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/app.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-038
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-038
- **Kind**: Application, **Name**: kick-e2e-038, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-038
- **Kind**: Deployment, **Name**: app-038, Namespace: kick-e2e-038

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-038
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-008, KICK-FEAT-013
