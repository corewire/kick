# KICK-E2E-048 - Restart While Waiting For Window

Behavior under test: restart while waiting for window.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-048
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-048
- **Kind**: Deployment, **Name**: app-048, Namespace: kick-e2e-048
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-048

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-048
- **Provider**: core
- **Class**: recovery
- **Required**: True
- **Features**: KICK-FEAT-006, KICK-FEAT-014
