# KICK-E2E-037 - Outofsync Waits

Behavior under test: outofsync waits.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-037
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-037
- **Kind**: Application, **Name**: kick-e2e-037, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-037
- **Kind**: Deployment, **Name**: app-037, Namespace: kick-e2e-037
- **Kind**: ConfigMap, **Name**: drift-037, Namespace: kick-e2e-037

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-037
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-008, KICK-FEAT-013
