# KICK-E2E-024 - Annotation Owner In Argo Cd Namespace

## Behavior under test
Primary behavior: annotation owner in argo cd namespace.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml
- manifests/deployment.yaml
- manifests/secret.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-024
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-024
- **Kind**: Application, **Name**: kick-e2e-024, Namespace: argocd
- **Kind**: Application, **Name**: kick-e2e-024-annotated, Namespace: argocd
- **Kind**: Deployment, **Name**: app-024, Namespace: kick-e2e-024
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-024

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-024
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-009
