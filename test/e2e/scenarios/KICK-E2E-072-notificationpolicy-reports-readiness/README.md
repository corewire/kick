# KICK-E2E-072 - Notificationpolicy Reports Readiness

Behavior under test: notificationpolicy reports readiness.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-072
- **Kind**: NotificationPolicy, **Name**: default, Namespace: kick-e2e-072

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-072
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-027
