# Instructions for coding agents

## Objective

Implement KICK incrementally without widening scope or inventing unresolved behavior.

## Required workflow

1. Read `README.md`.
2. Read the assigned file under `tasks/`.
3. Read only the specification files listed in that task's `Dependencies` section.
4. Implement the smallest complete change satisfying the acceptance criteria.
5. Add or update every required unit, Envtest, and e2e test.
6. Update `traceability/features.yaml` and scenario metadata for changed behavior.
7. Run the task's verification commands and `make feature-coverage`.
8. Do not begin a later task unless explicitly assigned.

## Framework

- KICK MUST be a plain Kubebuilder Go project built directly on `controller-runtime`.
- Use `controller-gen`, Envtest, Kustomize, Kind, Chainsaw, and a separately maintained Helm chart.
- Do not add Operator SDK, OLM, ClusterServiceVersions, bundles, or catalogs in the initial implementation.

## Design constraints

- The initial workload type is `apps/v1 Deployment` only.
- Full dependency discovery is mandatory.
- Discover only Secret and ConfigMap references used through container environment variables or mounted volumes.
- `imagePullSecrets` MUST NOT trigger restarts.
- KICK MUST NOT inject dependency hashes, environment variables, or KICK-owned state annotations into workloads.
- The rollout action MAY use the standard `kubectl.kubernetes.io/restartedAt` annotation because changing the Pod template is required to start a Deployment rollout.
- The core MUST remain independent of Argo CD and Flux types.
- Argo CD is the first provider adapter.
- Before restart, always re-read live state and re-evaluate whether the restart remains necessary.
- Missing or ambiguous GitOps ownership MUST block automatic restart.
- Do not silently resolve an item listed in `specs/17-open-questions.md`. Isolate it behind an interface or configuration and document the assumption.

## Testability and traceability

- Every feature MUST be testable.
- Every feature that can meaningfully be expressed as both a unit test and an e2e test MUST have both.
- Features may omit a test level only with a concrete rationale in `traceability/features.yaml`.
- Every implementation task and test MUST reference stable `KICK-FEAT-NNN` identifiers.
- A feature is incomplete until all required test mappings and tests pass.
- Never weaken or remove a required test merely to make CI pass.

## Code quality

- Prefer small pure functions for parsing, indexing, comparison, and gate evaluation.
- Keep provider-specific code in provider packages.
- Use typed reasons and conditions rather than matching free-form strings.
- Avoid global mutable state outside informer caches and controller-runtime indexes.
- Reconciliation MUST be idempotent.
- Controller restart recovery MUST derive from durable Kubernetes objects, not lost workqueue timers.
- Never log Secret data or content digests.

## Generated files

Do not edit generated CRDs, deepcopy code, API reference pages, or generated example output manually. Change source definitions and run the generation target.

## Required local commands

The repository SHOULD expose these commands:

```text
make generate
make manifests
make fmt
make vet
make lint
make test
make test-e2e
make helm-lint
make helm-template
make generate
make docs-gen-check
make feature-coverage
```

## End-to-end tests

Before adding or changing user-visible behavior, read `specs/20-e2e-suite-conventions.md`. Do not create one giant e2e test. Add or update a stable-ID scenario directory, its `trace.yaml`, and the central matrix. Assert stable convergence and exact rollout counts; Ready alone is not sufficient. Never print Secret values in diagnostics.
