# KICK-E2E-013 - Metadata Only Update Ignored

Behavior under test: metadata only update ignored.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-013
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-013
- **Kind**: Deployment, **Name**: app-013, Namespace: kick-e2e-013
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-013

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-013
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-005
