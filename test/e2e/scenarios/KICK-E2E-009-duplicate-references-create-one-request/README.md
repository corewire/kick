# KICK-E2E-009 - Duplicate References Create One Request

Behavior under test: duplicate references create one request.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-009
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-009
- **Kind**: Deployment, **Name**: duplicate-api, Namespace: kick-e2e-009
- **Kind**: Secret, **Name**: shared-secret, Namespace: kick-e2e-009

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-009
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-014
