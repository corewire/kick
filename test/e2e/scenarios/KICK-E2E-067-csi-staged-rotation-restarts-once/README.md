# KICK-E2E-067 - Csi Staged Rotation Restarts Once

## Behavior under test
Primary behavior: csi staged rotation restarts once.

This scenario exercises provider 'secrets-store-csi' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Ensures Secrets Store CSI integration behaves predictably so secret refresh events are translated into safe restart decisions.
It verifies CSI-backed dependency handling so non-standard secret delivery paths are covered.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-067
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-067
- **Kind**: SecretProviderClass, **Name**: app-067, Namespace: kick-e2e-067
- **Kind**: Deployment, **Name**: app-067, Namespace: kick-e2e-067

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-067
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-023
