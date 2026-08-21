# KICK-E2E-065 - Csi Identical Write Does Not Restart

Behavior under test: csi identical write does not restart.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-065
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-065
- **Kind**: SecretProviderClass, **Name**: app-065, Namespace: kick-e2e-065
- **Kind**: Deployment, **Name**: app-065, Namespace: kick-e2e-065

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-065
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-023
