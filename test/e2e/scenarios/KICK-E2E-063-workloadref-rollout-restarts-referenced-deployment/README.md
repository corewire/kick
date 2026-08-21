# KICK-E2E-063 - Workloadref Rollout Restarts Referenced Deployment

Behavior under test: workloadref rollout restarts referenced deployment.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-063
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-063
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-063

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-063
- **Provider**: argo-rollouts
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-024
