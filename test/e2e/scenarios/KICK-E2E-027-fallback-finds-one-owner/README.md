# KICK-E2E-027 - Fallback Finds One Owner

Behavior under test: fallback finds one owner.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-027
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-027
- **Kind**: Application, **Name**: kick-e2e-027, Namespace: argocd
- **Kind**: Deployment, **Name**: app-027, Namespace: kick-e2e-027
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-027

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-027
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-010
