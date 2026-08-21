# KICK-E2E-071 - Kargo Ambiguous Stage Blocks

## Behavior under test
Primary behavior: kargo ambiguous stage blocks.

This scenario exercises provider 'kargo' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Prevents unsafe promotion-time restarts by ensuring Kargo stage gating is enforced before rollout actions.
It verifies fail-closed behavior for ambiguous or missing ownership signals, preventing automatic restarts on uncertain targets.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- manifests/secret.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-071
- **Kind**: Application, **Name**: kick-e2e-071, Namespace: argocd
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-071
- **Kind**: Deployment, **Name**: app-071, Namespace: kick-e2e-071
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-071

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/deployment.yaml
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-071
- **Provider**: kargo
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-025
