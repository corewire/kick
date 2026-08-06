# Future Flux provider

## Status

Planned, not part of v1.

## Goal

Implement the generic GitOps provider contract for Flux without changing KICK's core dependency, freshness, request, or restart logic.

## Potential owners

- `kustomize.toolkit.fluxcd.io/Kustomization`
- `helm.toolkit.fluxcd.io/HelmRelease`

## Provider responsibilities

The Flux adapter will need to:

- resolve the exact owning Flux object;
- distinguish Kustomization and HelmRelease ownership;
- require current observed generation and Ready state;
- detect active reconciliation or upgrade;
- normalize Flux schedules into `GateDecision`;
- return `OutsideSchedule` and `RequeueAt` like the Argo CD adapter.

## Scheduling

Flux schedule support should be modeled as a provider-specific schedule that maps to the generic gate. It does not need to share Argo CD's native allow/deny representation.

## Core invariance

Adding Flux MUST NOT change:

- dependency discovery;
- change observations;
- freshness evaluation;
- KickRequest state machine;
- restart execution.

## Detection ambiguity

If both Argo CD and Flux confidently claim one workload, KICK MUST block with `AmbiguousOwner` unless a future explicit provider-selection policy is introduced.

## Deferred research

- exact ownership metadata;
- HelmRelease-to-generated-Deployment relationship;
- Kustomization inventory fields;
- Flux Operator scheduling APIs;
- field-manager behavior for rollout annotations;
- compatibility matrices.
