# KICK-E2E-003 - Configmap Environment Reference Triggers Kick

Behavior under test: configmap environment reference triggers kick.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-003
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-003
- **Kind**: Deployment, **Name**: app-003, Namespace: kick-e2e-003
- **Kind**: ConfigMap, **Name**: app-config, Namespace: kick-e2e-003

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-003
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-001, KICK-FEAT-002
