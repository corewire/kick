# KICK-E2E-069 - Kargo Restart After Promotion

Behavior under test: kargo restart after promotion.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-069
- **Kind**: Project, **Name**: kick-e2e-069
- **Kind**: ProjectConfig, **Name**: kick-e2e-069, Namespace: kick-e2e-069
- **Kind**: Secret, **Name**: gitea-credentials, Namespace: kick-e2e-069
- **Kind**: Warehouse, **Name**: repo, Namespace: kick-e2e-069
- **Kind**: Stage, **Name**: prod, Namespace: kick-e2e-069
- **Kind**: Application, **Name**: kick-e2e-069, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-069
- **Kind**: Deployment, **Name**: app-069, Namespace: kick-e2e-069
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-069

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/deployment.yaml`
- `updated/secret.yaml`
- `updated/slow.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-069
- **Provider**: kargo
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-025
