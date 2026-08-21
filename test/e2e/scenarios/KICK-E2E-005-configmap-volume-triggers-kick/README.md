# KICK-E2E-005 - Configmap Volume Triggers Kick

Behavior under test: configmap volume triggers kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-005
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-005
- **Kind**: Deployment, **Name**: app-005, Namespace: kick-e2e-005
- **Kind**: ConfigMap, **Name**: app-config, Namespace: kick-e2e-005

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-005
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-003
