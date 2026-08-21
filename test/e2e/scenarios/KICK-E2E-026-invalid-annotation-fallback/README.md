# KICK-E2E-026 - Invalid Annotation Fallback

Behavior under test: invalid annotation fallback.

## Setup
The initial state of this scenario is defined by the following files:
- `resources.yaml`
- `manifests/deployments.yaml`
- `manifests/secret.yaml`

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-026
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-026
- **Kind**: Application, **Name**: kick-e2e-026, Namespace: argocd
- **Kind**: Deployment, **Name**: app-026-malformed, Namespace: kick-e2e-026
- **Kind**: Deployment, **Name**: app-026-mismatched, Namespace: kick-e2e-026
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-026

## Execution and assertions
The execution steps and assertions are driven by `chainsaw-test.yaml`.

This scenario also references the following update files:
- `updated/secret.yaml`

## Traceability
- [`trace.yaml`](./trace.yaml)
- **Scenario ID**: KICK-E2E-026
- **Provider**: argocd
- **Class**: behavior
- **Required**: True
- **Features**: KICK-FEAT-009
