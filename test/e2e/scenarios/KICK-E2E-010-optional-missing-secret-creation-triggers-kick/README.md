# KICK-E2E-010 - Optional Missing Secret Creation Triggers Kick

## Behavior under test
Primary behavior: optional missing secret creation triggers kick.

This scenario exercises provider 'core' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies edge-state handling for optional or absent dependencies so controller decisions remain deterministic.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-010
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-010
- **Kind**: Deployment, **Name**: app-010, Namespace: kick-e2e-010

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-010
- **Provider**: core
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-001
