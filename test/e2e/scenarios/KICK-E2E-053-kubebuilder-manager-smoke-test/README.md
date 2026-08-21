# KICK-E2E-053 - Kubebuilder Manager Smoke Test

Behavior under test: kubebuilder manager smoke test.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-053
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-053
- **Kind**: Deployment, **Name**: app-053, Namespace: kick-e2e-053
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-053

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-053
- **Provider**: core
- **Class**: smoke
- **Required**: True
- **Features**: KICK-FEAT-017
