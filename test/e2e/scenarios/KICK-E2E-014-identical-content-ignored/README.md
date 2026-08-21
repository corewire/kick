# KICK-E2E-014 - Identical Content Ignored

Behavior under test: identical content ignored.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-014
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-014
- **Kind**: Deployment, **Name**: app-014, Namespace: kick-e2e-014
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-014

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-014
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-005
