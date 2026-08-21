# KICK-E2E-046 - Pod Deletion Never Used

Behavior under test: pod deletion never used.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-046
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-046
- **Kind**: Deployment, **Name**: app-046, Namespace: kick-e2e-046
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-046

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-046
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-015
