# KICK-E2E-070 - Kargo Unannotated Application Blocks

Behavior under test: kargo unannotated application blocks.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-070
- **Kind**: Application, **Name**: kick-e2e-070, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-070
- **Kind**: Deployment, **Name**: app-070, Namespace: kick-e2e-070
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-070

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/deployment.yaml`
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-070
- **Provider**: kargo
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-025
