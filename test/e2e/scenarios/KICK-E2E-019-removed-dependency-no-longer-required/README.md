# KICK-E2E-019 - Removed Dependency No Longer Required

Behavior under test: removed dependency no longer required.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-019
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-019
- **Kind**: Deployment, **Name**: app-019, Namespace: kick-e2e-019
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-019

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-019
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
