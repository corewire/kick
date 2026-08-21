# KICK-E2E-050 - Restart After Patch Before Completion

Behavior under test: restart after patch before completion.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-050
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-050
- **Kind**: Deployment, **Name**: app-050, Namespace: kick-e2e-050
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-050

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-050
- **Provider**: core
- **Class**: recovery
- **Required**: True
- **Features**: KICK-FEAT-006, KICK-FEAT-014
