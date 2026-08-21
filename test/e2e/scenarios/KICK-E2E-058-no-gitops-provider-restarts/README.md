# KICK-E2E-058 - No Gitops Provider Restarts

Behavior under test: no gitops provider restarts.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-058
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-058
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-058
- **Kind**: Deployment, **Name**: app, Namespace: kick-e2e-058

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-058
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-022
