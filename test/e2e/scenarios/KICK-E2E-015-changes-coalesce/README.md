# KICK-E2E-015 - Changes Coalesce

## Behavior under test
Primary behavior: changes coalesce.

This scenario exercises provider 'core' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-015
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-015
- **Kind**: Deployment, **Name**: app-015, Namespace: kick-e2e-015
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-015

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-015
- **Provider**: core
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-014
