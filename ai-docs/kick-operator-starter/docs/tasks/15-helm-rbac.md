# Task 15: Helm chart, configuration, and RBAC

## Goal

Provide a production-installable chart with least-privilege configuration.

## Dependencies

- `specs/12-configuration-and-rbac.md`
- final API and controller watches

## Deliverables

- Helm values and templates.
- ClusterRole/Role split for Applications and AppProjects.
- leader election, probes, resources, security context, and PodDisruptionBudget.
- generated CRD installation.

## Acceptance criteria

- `helm lint` and template tests pass.
- RBAC contains no Pod delete permission.
- AppProject permissions are namespace-restricted where possible.
