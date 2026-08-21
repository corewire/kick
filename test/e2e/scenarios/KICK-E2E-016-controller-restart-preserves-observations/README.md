# KICK-E2E-016 - Controller Restart Preserves Observations

## Behavior under test
Primary behavior: controller restart preserves observations.

This scenario exercises provider 'core' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-016
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-016
- **Kind**: Deployment, **Name**: app-016, Namespace: kick-e2e-016
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-016

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-016
- **Provider**: core
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-006
