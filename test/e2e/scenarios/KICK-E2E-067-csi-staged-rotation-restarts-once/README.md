# KICK-E2E-067 - Csi Staged Rotation Restarts Once

Behavior under test: csi staged rotation restarts once.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-067
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-067
- **Kind**: SecretProviderClass, **Name**: app-067, Namespace: kick-e2e-067
- **Kind**: Deployment, **Name**: app-067, Namespace: kick-e2e-067

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-067
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-023
