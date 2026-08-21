# KICK-E2E-048 - Restart While Waiting For Window

## Behavior under test
Primary behavior: restart while waiting for window.

This scenario exercises provider 'core' in class 'recovery' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies time-window enforcement so changes do not trigger restarts outside approved maintenance periods.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-048
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-048
- **Kind**: Deployment, **Name**: app-048, Namespace: kick-e2e-048
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-048

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-048
- **Provider**: core
- **Class**: recovery
- **Required**: true
- **Features**: KICK-FEAT-006, KICK-FEAT-014
