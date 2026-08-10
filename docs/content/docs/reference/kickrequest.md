# KickRequest API Reference

Group/version: `kick.corewire.io/v1alpha1`

Kind: `KickRequest`

## spec

- `targetRef.apiVersion` default: `apps/v1`
- `targetRef.kind` enum: `Deployment`
- `targetRef.name` required

## status

- `phase` enum:
  - `Pending`
  - `WaitingForGate`
  - `WaitingForOwner`
  - `WaitingForApplicationSync`
  - `WaitingForRollout`
  - `Executing`
  - `Succeeded`
  - `NoLongerRequired`
  - `Failed`
- `owner`: resolved GitOps owner details
- `gate`: last gate decision (`reason`, `message`, `requeueAt`)
- `latestObservedDependencyChange`
- `currentRollout` (`replicaSet`, `startedAt`)
- `conditions`

## Mutated fields

KICK mutates:

- `status.*` on KickRequest;
- `metadata.annotations["kubectl.kubernetes.io/restartedAt"]` on target Deployment PodTemplate.