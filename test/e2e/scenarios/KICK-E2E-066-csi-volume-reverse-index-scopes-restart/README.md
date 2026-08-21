# KICK-E2E-066 - Csi Volume Reverse Index Scopes Restart

## Behavior under test
Primary behavior: csi volume reverse index scopes restart.

This scenario exercises provider 'secrets-store-csi' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Ensures Secrets Store CSI integration behaves predictably so secret refresh events are translated into safe restart decisions.
It verifies CSI-backed dependency handling so non-standard secret delivery paths are covered.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-066
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-066
- **Kind**: SecretProviderClass, **Name**: rotated-066, Namespace: kick-e2e-066
- **Kind**: SecretProviderClass, **Name**: untouched-066, Namespace: kick-e2e-066
- **Kind**: Deployment, **Name**: consumer-066, Namespace: kick-e2e-066
- **Kind**: Deployment, **Name**: bystander-066, Namespace: kick-e2e-066

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-066
- **Provider**: secrets-store-csi
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-023
