# KICK-E2E-039 - Argo Sync Rollout Satisfies Request

Behavior under test: argo sync rollout satisfies request.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/app.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-039
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-039
- **Kind**: Application, **Name**: kick-e2e-039, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-039
- **Kind**: Deployment, **Name**: app-039, Namespace: kick-e2e-039

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/app.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-039
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-013
