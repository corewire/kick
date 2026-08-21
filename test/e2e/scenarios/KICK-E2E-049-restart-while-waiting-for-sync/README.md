# KICK-E2E-049 - Restart While Waiting For Sync

## Behavior under test
Primary behavior: restart while waiting for sync.

This scenario exercises provider 'core' in class 'recovery' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies gate timing so restarts are deferred until controller state is stable and authoritative.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-049
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-049
- **Kind**: Deployment, **Name**: app-049, Namespace: kick-e2e-049
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-049

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-049
- **Provider**: core
- **Class**: recovery
- **Required**: true
- **Features**: KICK-FEAT-006, KICK-FEAT-014
