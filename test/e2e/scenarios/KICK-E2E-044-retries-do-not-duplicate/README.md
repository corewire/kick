# KICK-E2E-044 - Retries Do Not Duplicate

Behavior under test: retries do not duplicate.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-044
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-044
- **Kind**: Deployment, **Name**: app-044, Namespace: kick-e2e-044
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-044

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-044
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-015
