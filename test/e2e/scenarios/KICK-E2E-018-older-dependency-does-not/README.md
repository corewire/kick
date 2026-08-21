# KICK-E2E-018 - Older Dependency Does Not

Behavior under test: older dependency does not.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-018
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-018
- **Kind**: Deployment, **Name**: app-018, Namespace: kick-e2e-018
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-018

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-018
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-007
