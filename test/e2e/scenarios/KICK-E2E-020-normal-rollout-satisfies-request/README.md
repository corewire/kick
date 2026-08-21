# KICK-E2E-020 - Normal Rollout Satisfies Request

Behavior under test: normal rollout satisfies request.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-020
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-020
- **Kind**: Deployment, **Name**: app-020, Namespace: kick-e2e-020
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-020

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-020
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
