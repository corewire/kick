# Deployment restart execution

## Goal

Trigger exactly one normal Deployment rollout when live state proves a kick is still required.

## Pre-execution gate

Immediately before mutation, the controller MUST re-read:

- KickRequest;
- Deployment;
- current ReplicaSets;
- current dependency set;
- latest dependency observations;
- GitOps owner and gate state.

If any gate closes or the Deployment is now fresh, do not mutate.

## Mutation

KICK MUST NOT delete Pods.

KICK MUST NOT inject dependency hashes or environment variables.

To trigger a rollout, KICK MAY patch the standard annotation:

```text
spec.template.metadata.annotations["kubectl.kubernetes.io/restartedAt"]
```

The value is the current RFC3339 timestamp.

The patch MUST touch only the Pod-template annotation required for the restart.

## Idempotency

The controller MUST prevent duplicate restarts caused by retries:

- transition request state before or atomically with mutation where practical;
- after a conflict, re-read live state;
- recognize that a new matching ReplicaSet may already exist;
- never blindly apply a new timestamp on every reconcile.

## Completion

For v1, completion SHOULD mean the new Deployment rollout completes, not merely that the patch was accepted.

A rollout is complete when Deployment status indicates the desired replicas are updated and available and the observed generation is current. Exact zero-replica handling must be tested.

A rollout failure or timeout marks the request Failed with a typed reason. It MUST NOT continuously create more rollouts.

## GitOps compatibility

The Argo CD e2e suite MUST verify that ordinary self-heal does not cause a second rollout by removing the restart annotation.

Configurations such as replace/force sync that may contest the field MUST be documented as supported or unsupported based on tests.

## Acceptance criteria

- One required kick creates exactly one new ReplicaSet.
- Reconcile retries do not create repeated ReplicaSets.
- Pod deletion is never used.
- A completed unrelated rollout can satisfy the request without mutation.
- Failed rollout does not enter an infinite restart loop.
