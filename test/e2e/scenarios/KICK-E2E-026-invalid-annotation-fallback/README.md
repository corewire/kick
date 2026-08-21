# KICK-E2E-026 - Invalid Annotation Fallback

## Behavior under test
Primary behavior: invalid annotation fallback.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployments.yaml
- manifests/secret.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-026
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-026
- **Kind**: Application, **Name**: kick-e2e-026, Namespace: argocd
- **Kind**: Deployment, **Name**: app-026-malformed, Namespace: kick-e2e-026
- **Kind**: Deployment, **Name**: app-026-mismatched, Namespace: kick-e2e-026
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-026

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-026
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-009
