# KICK-E2E-017 - Newer Dependency Triggers Rollout

Behavior under test: newer dependency triggers rollout.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-017
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-017
- **Kind**: Deployment, **Name**: app-017, Namespace: kick-e2e-017
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-017

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-017
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
