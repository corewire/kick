# KICK-E2E-009 - Duplicate References Create One Request

## Behavior under test
Primary behavior: duplicate references create one request.

This scenario exercises provider 'core' in class 'behavior' (required='true') and verifies that KICK's decision flow matches the expected outcome for this case.

## Why this matters
Catches restart-trigger regressions that can leave workloads running stale Secret/ConfigMap data or cause unnecessary restarts.
It verifies this concrete decision path end-to-end, reducing regression risk in dependency-to-restart flow.

## Setup
The initial state of this scenario is defined by:
- resources.yaml

### Resource inventory
- **Kind**: Namespace, **Name**: kick-e2e-009
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-009
- **Kind**: Deployment, **Name**: duplicate-api, Namespace: kick-e2e-009
- **Kind**: Secret, **Name**: shared-secret, Namespace: kick-e2e-009

## Execution and assertions
Execution and assertions are defined in chainsaw-test.yaml.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-009
- **Provider**: core
- **Class**: behavior
- **Required**: true
- **Features**: KICK-FEAT-014
