# KICK-E2E-002 - Secretkeyref Triggers Kick

Behavior under test: secretkeyref triggers kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-002
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-002
- **Kind**: Deployment, **Name**: app-002, Namespace: kick-e2e-002
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-002

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-002
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-002
