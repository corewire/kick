# KICK Operator Specifications

KICK detects Kubernetes workloads that silently require a restart because a consumed Secret or ConfigMap changed after the workload's latest rollout. KICK waits until the workload's GitOps owner permits the action, then triggers a normal Kubernetes rollout.

This repository is intentionally split into small specifications. A coding agent should read only `AGENTS.md`, this file, the task file it is implementing, and the task's listed dependencies.

KICK is a plain **Kubebuilder/controller-runtime** Go operator. Operator SDK and OLM packaging are outside the initial scope.

## Design summary

1. Discover Secret and ConfigMap references from Deployment Pod templates.
2. Observe relevant data changes and maintain durable change observations.
3. Determine whether a Deployment is stale relative to its current ReplicaSet.
4. Resolve the workload's GitOps owner through a provider adapter.
5. Wait while the provider is reconciling, out of sync, or outside its schedule/window.
6. Re-evaluate live state and restart only when still required.

## Initial scope

- Kubernetes `apps/v1` Deployments
- Secret and ConfigMap references in environment variables and volumes
- Full automatic dependency discovery
- Argo CD provider
- Automatic Application discovery
- AppProject sync-window evaluation
- Durable restart requests
- Standard rollout restart

Flux support is planned through the generic provider interface, but is not part of the initial implementation.

## Specification map

| File | Purpose |
|---|---|
| `specs/00-product-and-scope.md` | Product definition, goals, and boundaries |
| `specs/01-domain-model.md` | Shared terminology and invariants |
| `specs/02-dependency-discovery.md` | Pod-template reference discovery and indexes |
| `specs/03-change-observation.md` | Relevant Secret/ConfigMap changes and durable observations |
| `specs/04-deployment-freshness.md` | Determine whether a Deployment needs a restart |
| `specs/05-gitops-provider-contract.md` | Provider-neutral ownership and gate interface |
| `specs/06-argocd-ownership.md` | Application and AppProject discovery |
| `specs/07-argocd-sync-windows.md` | AppProject sync-window interpretation |
| `specs/08-argocd-application-gate.md` | Synced/reconciling gate behavior |
| `specs/09-restart-request-api.md` | Durable request API and state machine |
| `specs/10-restart-execution.md` | Standard rollout restart and completion behavior |
| `specs/11-controller-architecture.md` | Reconcilers, indexes, watches, and recovery |
| `specs/12-configuration-and-rbac.md` | Operator configuration, RBAC, and tenancy |
| `specs/13-observability.md` | Conditions, events, metrics, and logs |
| `specs/14-testing-strategy.md` | Unit, Envtest, integration, and CI requirements |
| `specs/15-e2e-scenarios.md` | Required behavioral scenarios |
| `specs/16-documentation-and-release.md` | Documentation, generation, packaging, and release |
| `specs/17-open-questions.md` | Explicitly unresolved research questions |
| `specs/18-future-flux-provider.md` | Planned Flux adapter boundaries |
| `specs/19-framework-and-test-traceability.md`
- `specs/20-e2e-suite-conventions.md` | Kubebuilder decision and feature-to-test enforcement |
| `specs/22-kickpolicy.md` | Namespaced policy model, selection scope, and policy-driven gates |
| `specs/23-unified-generation-entrypoint.md` | `make generate` as the single generation command |
| `traceability/` | Machine-readable feature and e2e coverage matrices |
| `tasks/` | Ordered, independently implementable work packages |
| `examples/` | Example resources and flows |

Example highlights:

- `examples/argocd-autodiscovery.md`
- `examples/full-flow.md`
- `examples/kickpolicy-autodiscovery.md`

## Reading order for implementers

Start with:

1. `AGENTS.md`
2. `specs/00-product-and-scope.md`
3. `specs/01-domain-model.md`
4. the assigned task file
5. only the dependencies listed by that task

Before completing a feature, run the feature coverage check and ensure its feature IDs are mapped to all required test levels.

## Status language

Normative requirements use **MUST**, **MUST NOT**, **SHOULD**, and **MAY**.

## Efficient coding-agent workflow

Use [`specs/21-token-efficient-development.md`](specs/21-token-efficient-development.md) as the mandatory workflow for low-cost coding agents. Agents should enter through one task file and load only its declared dependencies.
