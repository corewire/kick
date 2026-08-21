# KICK-E2E-024 - Annotation Owner In Argo Cd Namespace

Behavior under test: annotation owner in argo cd namespace.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployment.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-024
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-024
- **Kind**: Application, **Name**: kick-e2e-024, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-024-annotated, Namespace: argocd
- **Kind**: Deployment, **Name**: app-024, Namespace: kick-e2e-024
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-024

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-024
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-009
