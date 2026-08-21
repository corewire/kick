# KICK-E2E-051 - Stale Requeue Recomputed

Behavior under test: stale requeue recomputed.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-051
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-051
- **Kind**: Deployment, **Name**: app-051, Namespace: kick-e2e-051
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-051

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-051
- **Provider**: core
- **Class**: recovery
- **Required**: True
- **Features**: KICK-FEAT-006
