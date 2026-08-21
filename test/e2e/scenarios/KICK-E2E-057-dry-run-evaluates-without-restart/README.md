# KICK-E2E-057 - Dry Run Evaluates Without Restart

Behavior under test: dry run evaluates without restart.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-057
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-057
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-057
- **Kind**: Deployment, **Name**: app, Namespace: kick-e2e-057

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-057
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-021
