# KICK-E2E-043 - Exactly One Replicaset

## Behavior under test
Primary behavior: exactly one replicaset.

This scenario exercises provider 'core' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-043
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-043
- **Kind**: Deployment, **Name**: app-043, Namespace: kick-e2e-043
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-043

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-043
- **Provider**: core
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-015
