# KICK-E2E-056 - Dependency Selector Scopes Triggers

Behavior under test: dependency selector scopes triggers.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-056
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-056
- **Kind**: Secret, **Name**: watched-secret, Namespace: kick-e2e-056
- **Kind**: Secret, **Name**: ignored-secret, Namespace: kick-e2e-056
- **Kind**: Deployment, **Name**: app-watched, Namespace: kick-e2e-056
- **Kind**: Deployment, **Name**: app-ignored, Namespace: kick-e2e-056

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-056
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-020
