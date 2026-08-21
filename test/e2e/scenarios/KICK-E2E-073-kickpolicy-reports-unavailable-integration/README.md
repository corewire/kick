# KICK-E2E-073 - Kickpolicy Reports Unavailable Integration

Behavior under test: kickpolicy reports unavailable integration.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-073
- **Kind**: KickPolicy, **Name**: flux-gated, Namespace: kick-e2e-073
- **Kind**: KickPolicy, **Name**: ungated, Namespace: kick-e2e-073

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-073
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-028
