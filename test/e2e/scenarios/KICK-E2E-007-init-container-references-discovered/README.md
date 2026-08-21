# KICK-E2E-007 - Init Container References Discovered

Behavior under test: init container references discovered.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-007
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-007
- **Kind**: Deployment, **Name**: app-007, Namespace: kick-e2e-007
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-007

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-007
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-002
