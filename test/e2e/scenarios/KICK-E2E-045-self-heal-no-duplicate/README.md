# KICK-E2E-045 - Self Heal No Duplicate

Behavior under test: self heal no duplicate.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-045
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-045
- **Kind**: Deployment, **Name**: app-045, Namespace: kick-e2e-045
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-045

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-045
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-015
