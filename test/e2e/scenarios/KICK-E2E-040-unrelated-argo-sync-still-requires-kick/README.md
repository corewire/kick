# KICK-E2E-040 - Unrelated Argo Sync Still Requires Kick

Behavior under test: unrelated argo sync still requires kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/app.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-040
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-040
- **Kind**: Application, **Name**: kick-e2e-040, Namespace: argocd
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-040
- **Kind**: ConfigMap, **Name**: unrelated-040, Namespace: kick-e2e-040
- **Kind**: Deployment, **Name**: app-040, Namespace: kick-e2e-040

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/app.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-040
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-013
