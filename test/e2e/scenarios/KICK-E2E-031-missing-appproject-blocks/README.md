# KICK-E2E-031 - Missing Appproject Blocks

## Behavior under test
Primary behavior: missing appproject blocks.

This scenario exercises provider 'argocd' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Protects GitOps safety by ensuring restarts happen only when Argo CD ownership and sync/window conditions are correctly evaluated.
It verifies edge-state handling for optional or absent dependencies so controller decisions remain deterministic.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-031
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-031
- **Kind**: Application, **Name**: kick-e2e-031, Namespace: argocd
- **Kind**: Deployment, **Name**: app-031, Namespace: kick-e2e-031
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-031

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-031
- **Provider**: argocd
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-011
