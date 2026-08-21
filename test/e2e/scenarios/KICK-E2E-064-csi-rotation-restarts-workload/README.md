# KICK-E2E-064 - Csi Rotation Restarts Workload

Behavior under test: csi rotation restarts workload.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-064
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-064
- **Kind**: SecretProviderClass, **Name**: app-064, Namespace: kick-e2e-064
- **Kind**: Deployment, **Name**: app-064, Namespace: kick-e2e-064

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-064
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-023
