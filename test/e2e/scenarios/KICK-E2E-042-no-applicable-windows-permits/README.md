# KICK-E2E-042 - No Applicable Windows Permits

Behavior under test: no applicable windows permits.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `windows.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-042
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-042
- **Kind**: AppProject, **Name**: kick-e2e-042, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-042, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-042
- **Kind**: Deployment, **Name**: app-042, Namespace: kick-e2e-042
- **Kind**: AppProject, **Name**: kick-e2e-042, Namespace: argocd

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-042
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-012
