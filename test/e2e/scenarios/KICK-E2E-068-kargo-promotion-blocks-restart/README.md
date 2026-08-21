# KICK-E2E-068 - Kargo Promotion Blocks Restart

Behavior under test: kargo promotion blocks restart.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-068
- **Kind**: Project, **Name**: kick-e2e-068
- **Kind**: ProjectConfig, **Name**: kick-e2e-068, Namespace: kick-e2e-068
- **Kind**: Secret, **Name**: gitea-credentials, Namespace: kick-e2e-068
- **Kind**: Warehouse, **Name**: repo, Namespace: kick-e2e-068
- **Kind**: Stage, **Name**: prod, Namespace: kick-e2e-068
- **Kind**: Application, **Name**: kick-e2e-068, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-068
- **Kind**: Deployment, **Name**: app-068, Namespace: kick-e2e-068
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-068

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/deployment.yaml`
- `updated/secret.yaml`
- `updated/slow.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-068
- **Provider**: kargo
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-025
