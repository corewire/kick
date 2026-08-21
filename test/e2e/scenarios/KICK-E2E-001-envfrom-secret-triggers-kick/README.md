# KICK-E2E-001 - Envfrom Secret Triggers Kick

Behavior under test: envfrom secret triggers kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-001
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-001
- **Kind**: Deployment, **Name**: app-001, Namespace: kick-e2e-001
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-001

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-001
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-002
