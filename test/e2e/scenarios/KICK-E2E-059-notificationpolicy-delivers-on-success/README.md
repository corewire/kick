# KICK-E2E-059 - Notificationpolicy Delivers On Success

Behavior under test: notificationpolicy delivers on success.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-059
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-059
- **Kind**: NotificationPolicy, **Name**: default, Namespace: kick-e2e-059
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-059
- **Kind**: Deployment, **Name**: app, Namespace: kick-e2e-059

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-059
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-026
