# KICK-E2E-060 - Canary Restart Does Not Rerun Steps

## Behavior under test
Primary behavior: canary restart does not rerun steps.

This scenario exercises provider 'argo-rollouts' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Guards progressive-delivery semantics so restart actions do not break Rollout step behavior or ownership targeting.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-060
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-060
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-060

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

Scenario update inputs:
- updated/secret.yaml

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-060
- **Provider**: argo-rollouts
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-024
