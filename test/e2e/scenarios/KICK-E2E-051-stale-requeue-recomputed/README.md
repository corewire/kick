# KICK-E2E-051 - Stale Requeue Recomputed

## Behavior under test
Primary behavior: stale requeue recomputed.

This scenario exercises provider 'core' in class 'recovery' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-051
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-051
- **Kind**: Deployment, **Name**: app-051, Namespace: kick-e2e-051
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-051

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-051
- **Provider**: core
- **Class**: recovery
- **Required**: true
- **Features**: KICK-FEAT-006
