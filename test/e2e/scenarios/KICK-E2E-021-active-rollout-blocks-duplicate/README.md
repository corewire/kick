# KICK-E2E-021 - Active Rollout Blocks Duplicate

Behavior under test: active rollout blocks duplicate.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-021
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-021
- **Kind**: Deployment, **Name**: app-021, Namespace: kick-e2e-021
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-021

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-021
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
