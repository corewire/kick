# KICK-E2E-066 - Csi Volume Reverse Index Scopes Restart

Behavior under test: csi volume reverse index scopes restart.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-066
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-066
- **Kind**: SecretProviderClass, **Name**: rotated-066, Namespace: kick-e2e-066
- **Kind**: SecretProviderClass, **Name**: untouched-066, Namespace: kick-e2e-066
- **Kind**: Deployment, **Name**: consumer-066, Namespace: kick-e2e-066
- **Kind**: Deployment, **Name**: bystander-066, Namespace: kick-e2e-066

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-066
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-023
