# KICK-E2E-030 - Appproject Control Plane Namespace

Behavior under test: appproject control plane namespace.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-030
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-030
- **Kind**: AppProject, **Name**: kick-e2e-030, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-030, Namespace: argocd
- **Kind**: Deployment, **Name**: app-030, Namespace: kick-e2e-030
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-030

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-030
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-011
