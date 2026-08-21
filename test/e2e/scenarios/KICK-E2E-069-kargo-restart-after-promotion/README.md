# KICK-E2E-069 - Kargo Restart After Promotion

## Behavior under test
Primary behavior: kargo restart after promotion.

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
- **Kind**: Namespace, **Name**: kick-e2e-069
- **Kind**: Project, **Name**: kick-e2e-069
- **Kind**: ProjectConfig, **Name**: kick-e2e-069, Namespace: kick-e2e-069
- **Kind**: Secret, **Name**: gitea-credentials, Namespace: kick-e2e-069
- **Kind**: Warehouse, **Name**: repo, Namespace: kick-e2e-069
- **Kind**: Stage, **Name**: prod, Namespace: kick-e2e-069
- **Kind**: Application, **Name**: kick-e2e-069, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-069
- **Kind**: Deployment, **Name**: app-069, Namespace: kick-e2e-069
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-069

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/deployment.yaml
- updated/secret.yaml
- updated/slow.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-069
- **Provider**: kargo
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-025
