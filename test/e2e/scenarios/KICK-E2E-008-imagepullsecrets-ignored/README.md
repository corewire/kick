# KICK-E2E-008 - Imagepullsecrets Ignored

Behavior under test: imagepullsecrets ignored.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-008
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-008
- **Kind**: Deployment, **Name**: app-008, Namespace: kick-e2e-008
- **Kind**: Secret, **Name**: pull-creds, Namespace: kick-e2e-008

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-008
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-004
