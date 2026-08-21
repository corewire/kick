# KICK-E2E-068 - Kargo Promotion Blocks Restart

## Behavior under test
Primary behavior: kargo promotion blocks restart.

This scenario exercises provider 'kargo' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Prevents unsafe promotion-time restarts by ensuring Kargo stage gating is enforced before rollout actions.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- manifests/secret.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-068
- **Kind**: Project, **Name**: kick-e2e-068
- **Kind**: ProjectConfig, **Name**: kick-e2e-068, Namespace: kick-e2e-068
- **Kind**: Secret, **Name**: gitea-credentials, Namespace: kick-e2e-068
- **Kind**: Warehouse, **Name**: repo, Namespace: kick-e2e-068
- **Kind**: Stage, **Name**: prod, Namespace: kick-e2e-068
- **Kind**: Application, **Name**: kick-e2e-068, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-068
- **Kind**: Deployment, **Name**: app-068, Namespace: kick-e2e-068
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-068

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/deployment.yaml
- updated/secret.yaml
- updated/slow.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-068
- **Provider**: kargo
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-025
