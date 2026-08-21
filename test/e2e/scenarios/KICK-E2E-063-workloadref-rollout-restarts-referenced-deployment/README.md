# KICK-E2E-063 - Workloadref Rollout Restarts Referenced Deployment

## Behavior under test
Primary behavior: workloadref rollout restarts referenced deployment.

This scenario exercises provider 'argo-rollouts' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Guards progressive-delivery semantics so restart actions do not break Rollout step behavior or ownership targeting.
It verifies target resolution so KICK restarts the correct workload owner when indirection is involved.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-063
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-063
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-063

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-063
- **Provider**: argo-rollouts
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-024
