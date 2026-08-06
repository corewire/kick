# Domain model

## Terms

### Dependency

A Secret or ConfigMap currently referenced by a Deployment Pod template through a supported environment-variable or volume field.

### Relevant change

A change to persisted Secret or ConfigMap content that can alter what the workload consumes. Metadata-only changes are not relevant.

### Change observation

Durable KICK state recording that a relevant dependency change was observed at a particular time and resource version.

### Current rollout

The ReplicaSet corresponding to the Deployment's current Pod template.

### Fresh Deployment

A Deployment for which every current dependency's last relevant change is older than or equal to the current rollout time.

### Stale Deployment

A Deployment for which at least one current dependency's last relevant change is newer than the current rollout time.

### GitOps owner

The provider-specific object responsible for reconciling the workload, normalized into a provider-neutral owner reference.

### Gate

A provider decision indicating whether KICK may act now. A gate includes reconciliation state and provider-specific scheduling constraints.

### Kick

The controlled action of triggering a new Deployment rollout so newly created Pods consume current dependency values.

## Core invariant

```text
latestDependencyChange = max(change time of every current dependency)
latestRollout = creation time of the ReplicaSet for the current Pod template
restartRequired = latestDependencyChange > latestRollout
```

A restart is not required if and only if all current dependency changes are older than or equal to the latest rollout.

## Important semantics

- KICK compares current dependencies, not dependencies that were previously referenced.
- Every decision MUST be recomputed immediately before execution.
- A later normal rollout can make a pending kick unnecessary.
- A dependency removed before execution no longer participates in the decision.
- A reverted content update still counts as an update under timestamp semantics.
- Kubernetes does not necessarily expose a reliable content-update timestamp; durable observation is therefore part of the design unless research proves otherwise.

## Safety defaults

- Unknown owner: block.
- Ambiguous owner: block.
- Unknown provider state: block.
- Missing AppProject: block.
- Active GitOps reconciliation: wait.
- Application OutOfSync: wait.
- Active Deployment rollout: wait.
