# KICK-E2E-015 - Changes Coalesce

Behavior under test: changes coalesce.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-015
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-015
- **Kind**: Deployment, **Name**: app-015, Namespace: kick-e2e-015
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-015

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-015
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-014
