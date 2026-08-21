# KICK-E2E-061 - Blue Green Restart Keeps Active Service

Behavior under test: blue green restart keeps active service.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-061
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-061
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-061
- **Kind**: Service, **Name**: app-061-active, Namespace: kick-e2e-061
- **Kind**: Service, **Name**: app-061-preview, Namespace: kick-e2e-061

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-061
- **Provider**: argo-rollouts
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-024
