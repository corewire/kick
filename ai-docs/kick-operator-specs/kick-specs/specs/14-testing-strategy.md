# Testing and CI strategy

## Principles

- Every feature MUST be designed for deterministic testing.
- Every feature that can meaningfully be tested at both unit and e2e level MUST have both.
- Feature-to-test traceability is machine-enforced through stable feature IDs.

- Pure logic is covered by table-driven unit tests.
- Kubernetes API behavior is covered by Envtest.
- Real controller interactions are covered by Kind and Chainsaw.
- Argo CD compatibility is tested with a real supported Argo CD installation.
- Generated artifacts and documentation freshness are enforced in CI.

## Required developer commands

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

Pin tools locally in `bin/`, following the practices used by the DROP operator.

## Pull-request checks

Every PR MUST run:

```text
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
make generate
make manifests
git diff --exit-code
helm lint
helm template
docs-gen-check
govulncheck ./...
make feature-coverage
```

## Unit-test areas

- dependency extraction;
- relevant-content comparison;
- freshness comparison;
- ReplicaSet selection helpers;
- provider detection;
- tracking-ID parsing;
- gate-reason mapping;
- sync-window evaluation;
- request state transitions.

## Envtest areas

- CRD defaults and validation;
- status patching;
- indexes;
- request coalescing;
- source observation persistence;
- watch-driven reenqueues;
- recovery of non-terminal requests;
- conflict retries.

## E2e

Use Kind plus Chainsaw scenarios in one directory per behavior. See:

- `15-e2e-scenarios.md` for the mandatory behavior inventory;
- `20-e2e-suite-conventions.md` for structure, lifecycle stability, diagnostics, CI partitioning, and flake policy.

The suite MUST verify stable convergence. A transient Ready condition alone is insufficient. Scenarios that exercise stateful behavior MUST include a post-condition stability assertion, and recovery scenarios MUST restart the controller and prove idempotent continuation.

## Matrices

CI SHOULD test:

- oldest supported Kubernetes minor;
- newest supported Kubernetes minor;
- every declared supported Argo CD minor family.

## Supply chain

Release CI SHOULD:

- build multi-architecture images;
- generate an SBOM;
- scan images;
- sign images and Helm artifacts;
- publish checksums;
- pin external CI actions by commit SHA.

## Feature-to-test comparison

The repository MUST maintain `traceability/features.yaml` and `traceability/e2e-scenarios.yaml` as specified in `19-framework-and-test-traceability.md`.

CI MUST generate a report comparing each feature against required and actual unit, Envtest, and e2e coverage. Missing mandatory coverage MUST fail the pull request.

Traceability supplements but does not replace statement, branch, race, or mutation testing.
