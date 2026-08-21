# KICK-E2E-041 - Degraded But Synced Allowed

Behavior under test: degraded but synced allowed.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/app.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-041
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-041
- **Kind**: Application, **Name**: kick-e2e-041, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-041
- **Kind**: Deployment, **Name**: app-041, Namespace: kick-e2e-041
- **Kind**: Deployment, **Name**: broken-041, Namespace: kick-e2e-041

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-041
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-013
