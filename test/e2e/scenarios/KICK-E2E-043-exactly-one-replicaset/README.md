# KICK-E2E-043 - Exactly One Replicaset

Behavior under test: exactly one replicaset.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-043
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-043
- **Kind**: Deployment, **Name**: app-043, Namespace: kick-e2e-043
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-043

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-043
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-015
