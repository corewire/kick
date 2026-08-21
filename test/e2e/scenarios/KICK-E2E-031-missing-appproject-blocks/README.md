# KICK-E2E-031 - Missing Appproject Blocks

Behavior under test: missing appproject blocks.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-031
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-031
- **Kind**: Application, **Name**: kick-e2e-031, Namespace: argocd
- **Kind**: Deployment, **Name**: app-031, Namespace: kick-e2e-031
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-031

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-031
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-011
