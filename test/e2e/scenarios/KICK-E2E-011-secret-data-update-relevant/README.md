# KICK-E2E-011 - Secret Data Update Relevant

Behavior under test: secret data update relevant.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-011
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-011
- **Kind**: Deployment, **Name**: secret-api, Namespace: kick-e2e-011
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-011

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-011
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-005
