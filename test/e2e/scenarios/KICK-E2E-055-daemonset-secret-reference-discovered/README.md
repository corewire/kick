# KICK-E2E-055 - Daemonset Secret Reference Discovered

Behavior under test: daemonset secret reference discovered.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-055
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-055
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-055
- **Kind**: DaemonSet, **Name**: app-055, Namespace: kick-e2e-055

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-055
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-007
