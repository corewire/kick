# KICK-E2E-025 - Annotation Owner In Other Namespace

## Behavior under test
Primary behavior: annotation owner in other namespace.

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
- **Kind**: Namespace, **Name**: kick-e2e-025
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-025
- **Kind**: Application, **Name**: kick-e2e-025, Namespace: kick-e2e-025
- **Kind**: Application, **Name**: kick-e2e-025-annotated, Namespace: kick-e2e-025
- **Kind**: Deployment, **Name**: app-025, Namespace: kick-e2e-025
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-025

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-025
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-009
