---
title: Continuous integration
weight: 45
---

The [`ci` workflow](https://github.com/corewire/kick/blob/main/.github/workflows/ci.yml)
runs on every pull request and on `main`. Everything it does is a make target,
so the same gates run locally.

## Job graph

```
changes ─▶ image ─▶ e2e (core │ argocd │ rollouts │ csi │ kargo)
lint
unit-tests                                                       ─▶ ci
race-tests
generated
```

`lint`, `unit-tests`, `race-tests` and `generated` start immediately and in
parallel; only the e2e suites wait, and only for the operator image. `ci` is the
single required status check: it fails if any job did not succeed or was skipped
unintentionally.

Each e2e suite gets its own kind cluster because the suites install different
control planes (see [End-to-end tests](../e2e-testing/)). The `core` job also
runs the `recovery` suite, which needs no extra control plane.

## What each job proves

| Job | Command |
|---|---|
| `lint` | `make workflow-lint`, `make fmt` + `git diff --exit-code`, `make lint`, `make static-check`, `make vulncheck` |
| `unit-tests` | `make test` (unit + envtest) |
| `race-tests` | `make test-race` |
| `generated` | `make generate` + `git diff --exit-code`, `make helm-lint`, `make helm-template`, `make release-chart VERSION=0.0.0-ci`, `make tools-test`, `make feature-coverage` |
| `e2e` | `make test-e2e-<suite>` |

`make vet` is not a CI step: golangci-lint runs `govet` as part of `make lint`,
so a separate pass only compiles everything twice.

## Caching

- Go modules and build cache: `actions/setup-go` with `cache: true`.
- Tool binaries and envtest assets: keyed on
  [`hack/tool-versions.mk`](https://github.com/corewire/kick/blob/main/hack/tool-versions.mk)
  plus the resolved Go version and runner architecture by
  [`.github/actions/setup-build`](https://github.com/corewire/kick/blob/main/.github/actions/setup-build/action.yml).
- Operator image: built once with Buildx layer caching, uploaded as an artifact
  and loaded into each kind cluster with `make kind-load-archive`.

## Selective execution

The `changes` job compares the pull request against its base. If only `docs/`,
`ai-docs/`, top-level Markdown or the generated `llms*.txt` files changed, the
image build and every e2e suite are skipped; the other gates always run.

## Scenario timings

Every e2e job writes a JUnit report per suite
(`make test-e2e-core E2E_REPORT_DIR=...`) and renders the slowest scenarios into
the job summary with
[`tools/e2e_timing_summary.py`](https://github.com/corewire/kick/blob/main/tools/e2e_timing_summary.py):

```bash
make test-e2e-core E2E_REPORT_DIR=dist/e2e-reports
make e2e-timing-summary E2E_REPORT_DIR=dist/e2e-reports
```

The reports are also uploaded as `e2e-timings-<suite>` artifacts.

## Reproducing CI locally

```bash
make ci-verify-local   # every non-cluster gate
make ci-e2e-local      # kind cluster, image, full e2e suite
```

## Releases

The [release workflow](https://github.com/corewire/kick/blob/main/.github/workflows/release.yml)
runs Mondays at 06:00 UTC, on stable `vMAJOR.MINOR.PATCH` tags, or manually from
`main`. Unchanged weeks are skipped; the first release uses the chart version.
Subsequent releases inspect commits and merged PRs since the previous stable tag:

| Signal | Bump | Example |
|---|---|---|
| Default | Patch | `fix: retry observation` |
| PR label `feature` or conventional `feat` | Minor | `feat: add provider` |
| PR label `breaking`, conventional `!`, or `BREAKING CHANGE:` footer | Major | `feat(api)!: remove field` |

The highest bump wins, including before 1.0. Labels belong to the merged PR,
not the branch. See the [planner](https://github.com/corewire/kick/blob/main/tools/release_plan.py)
and its [tests](https://github.com/corewire/kick/blob/main/tools/release_plan_test.py).

After all CI gates pass, the workflow publishes and signs
`ghcr.io/corewire/kick:vVERSION`, publishes `oci://ghcr.io/corewire/charts/kick`
with the same version, then creates the GitHub release and tag. The release
contains the chart, CRDs, SBOM, checksums, and generated notes. Publishing uses
`GITHUB_TOKEN` and OIDC; GHCR package write access must be allowed for this repository.
Proof: [publishing jobs](https://github.com/corewire/kick/blob/main/.github/workflows/release.yml).

Test packaging locally without publishing:

```bash
make release-chart VERSION=0.1.0
```

The [packaging target](https://github.com/corewire/kick/blob/main/Makefile)
sets the packaged image tag to `v0.1.0` without changing development defaults;
[automation tests](https://github.com/corewire/kick/blob/main/tools/automation_test.py)
verify that contract.

## Dependency updates

[Renovate](https://github.com/corewire/kick/blob/main/renovate.json) manages direct
and indirect Go modules, the docs theme, actions, container images (including
test fixtures and examples), Helm/control-plane versions, Go tools, Envtest,
Hugo, and [Python tooling](https://github.com/corewire/kick/blob/main/tools/requirements.txt).
Both Go module directives and the Docker builder update together in one
`go-version` group. Kubernetes and OpenTelemetry modules retain compatibility
groups, including their indirect entries.

Generated build output and archived specifications are excluded; KICK's own
image is a release output, not an external dependency. Updates use PRs and the
same CI gates; automerge is not enabled. Configuration:
[Renovate rules](https://github.com/corewire/kick/blob/main/renovate.json),
[tool pins](https://github.com/corewire/kick/blob/main/hack/tool-versions.mk).
