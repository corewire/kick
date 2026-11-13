# KICK-E2E-081 - Rotation While Controller Down Restarts Once

## Behavior under test
A Secret rotated while KICK is not running, after its consumer's Pods started, restarts the consumer exactly once when KICK comes back.

KICK never observes the rotation live, so it dates it from `managedFields`, which the API server records in whole seconds. The rotation lands either in the same second as the rollout anchor or a later one; both must restart once (spec 03, issue #61).

## Setup
- **Kind**: Namespace, **Name**: kick-e2e-081
- **Kind**: KickPolicy, **Name**: default, Namespace: kick-e2e-081 (`gitOps.provider: None`)
- **Kind**: Secret, **Name**: app-secret, Namespace: kick-e2e-081
- **Kind**: Deployment, **Name**: app-081, Namespace: kick-e2e-081

## Execution and assertions
[chainsaw-test.yaml](./chainsaw-test.yaml): scale the manager to zero, create the fixtures, rotate the Secret, start the manager, assert the KickRequest `Succeeded`, generation 2 and two ReplicaSets, and that nothing changes 30 s later. Runs non-concurrently; the manager is scaled back up in step cleanup.

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-081
- **Provider**: core
- **Class**: recovery
- **Required**: true
