# KICK-E2E-052 - Observability Excludes Secret Data

Behavior under test: observability excludes secret data.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-052
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-052
- **Kind**: Deployment, **Name**: app-052, Namespace: kick-e2e-052
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-052

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-052
- **Provider**: core
- **Class**: security
- **Required**: True
- **Features**: KICK-FEAT-016
