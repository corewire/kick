# KICK-E2E-016 - Controller Restart Preserves Observations

Behavior under test: controller restart preserves observations.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-016
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-016
- **Kind**: Deployment, **Name**: app-016, Namespace: kick-e2e-016
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-016

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-016
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-006
