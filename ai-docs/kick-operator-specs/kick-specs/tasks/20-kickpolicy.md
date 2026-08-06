# Task 20: KickPolicy API and policy-driven scope

## Goal

Introduce the namespaced `KickPolicy` API and enforce policy-driven workload management scope.

## Dependencies

- `specs/22-kickpolicy.md`
- `specs/02-dependency-discovery.md`
- `specs/09-restart-request-api.md`
- `specs/11-controller-architecture.md`

## Deliverables

- `KickPolicy` API types and CRD validation.
- Policy matching logic (no policy, one policy, conflicting policies).
- Namespace-scoped workload discovery with selector support.
- Policy-driven request gating and cancellation reasons.
- Policy status with conflict and provider availability conditions.
- Unit and e2e coverage for required policy scenarios.

## Acceptance criteria

- Workloads in namespaces without matching `KickPolicy` are unmanaged.
- Exactly one matching policy manages a workload.
- Multiple matches block the workload with `ConflictingPolicies`.
- `imagePullSecrets` changes never trigger kicks.
- Policy updates re-evaluate affected requests immediately.
- Policy deletion cancels pending requests with `PolicyDeleted`.
- Required unit tests and e2e scenarios in `specs/22-kickpolicy.md` are mapped and passing.
