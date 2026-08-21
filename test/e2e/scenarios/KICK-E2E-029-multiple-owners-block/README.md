# KICK-E2E-029 - Multiple Owners Block

Behavior under test: multiple owners block.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-029
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-029
- **Kind**: Application, **Name**: kick-e2e-029-a, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-029-b, Namespace: argocd
- **Kind**: Deployment, **Name**: app-029, Namespace: kick-e2e-029
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-029

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-029
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-010
