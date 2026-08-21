# KICK-E2E-012 - Configmap Data Update Relevant

Behavior under test: configmap data update relevant.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-012
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-012
- **Kind**: Deployment, **Name**: config-api, Namespace: kick-e2e-012
- **Kind**: ConfigMap, **Name**: app-config, Namespace: kick-e2e-012

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-012
- **Provider**: core
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-005
