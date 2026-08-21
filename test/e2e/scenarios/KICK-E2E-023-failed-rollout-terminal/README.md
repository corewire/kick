# KICK-E2E-023 - Failed Rollout Terminal

Behavior under test: failed rollout terminal.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-023
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-023
- **Kind**: Deployment, **Name**: app-023, Namespace: kick-e2e-023
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-023

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-023
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
