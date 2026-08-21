# KICK-E2E-049 - Restart While Waiting For Sync

Behavior under test: restart while waiting for sync.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-049
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-049
- **Kind**: Deployment, **Name**: app-049, Namespace: kick-e2e-049
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-049

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-049
- **Provider**: core
- **Class**: recovery
- **Required**: True
- **Features**: KICK-FEAT-006, KICK-FEAT-014
