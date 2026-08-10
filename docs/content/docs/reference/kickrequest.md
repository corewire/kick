# KickRequest API Reference

Group/version: `kick.corewire.io/v1alpha1`

Kind: `KickRequest`

## spec

- `targetRef.apiVersion` default: `apps/v1`. Use `argoproj.io/v1alpha1` for an
  Argo Rollout.
- `targetRef.kind` enum: `Deployment`, `StatefulSet`, `DaemonSet`, `Rollout`
- `targetRef.name` required

`Rollout` is only accepted with `apiVersion: argoproj.io/v1alpha1`, and requires
the controller to run with `--enable-argo-rollouts` and the CRD to be present.

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
  - `DryRun` — terminal. The policy had `spec.dryRun: true`, so everything was
    evaluated but no workload was patched.
- `owner`: resolved GitOps owner details
- `gate`: last gate decision (`reason`, `message`, `requeueAt`)
- `latestObservedDependencyChange`
- `currentRollout` (`replicaSet`, `startedAt`)
- `conditions`

## Mutated fields

KICK mutates:

- `status.*` on KickRequest;
- `metadata.annotations["kubectl.kubernetes.io/restartedAt"]` on the target
  Deployment, StatefulSet or DaemonSet PodTemplate;
- `spec.restartAt` on a target Argo `Rollout`. The pod template is deliberately
  left untouched so the canary or blue-green strategy is not re-run for a
  configuration change.

KICK never writes dependency hashes, environment variables, or KICK-owned state
annotations into a workload.