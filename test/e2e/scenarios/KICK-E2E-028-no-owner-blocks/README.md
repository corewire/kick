# KICK-E2E-028 - No Owner Blocks

Behavior under test: no owner blocks.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-028
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-028
- **Kind**: Application, **Name**: kick-e2e-028-unrelated, Namespace: argocd
- **Kind**: Deployment, **Name**: app-028, Namespace: kick-e2e-028
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-028

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-028
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-010
