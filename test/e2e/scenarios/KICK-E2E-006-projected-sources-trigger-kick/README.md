# KICK-E2E-006 - Projected Sources Trigger Kick

Behavior under test: projected sources trigger kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-006
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-006
- **Kind**: Deployment, **Name**: app-006, Namespace: kick-e2e-006
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-006
- **Kind**: ConfigMap, **Name**: app-config, Namespace: kick-e2e-006

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-006
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-003
