# KICK-E2E-022 - Scaled To Zero Deterministic

Behavior under test: scaled to zero deterministic.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-022
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-022
- **Kind**: Deployment, **Name**: app-022, Namespace: kick-e2e-022
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-022

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-022
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
