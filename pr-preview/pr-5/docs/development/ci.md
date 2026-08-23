# Continuous integration

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
| `lint` | `make fmt` + `git diff --exit-code`, `make lint`, `make static-check`, `make vulncheck` |
| `unit-tests` | `make test` (unit + envtest) |
| `race-tests` | `make test-race` |
| `generated` | `make generate` + `git diff --exit-code`, `make helm-lint`, `make helm-template`, `make tools-test`, `make feature-coverage` |
| `e2e` | `make test-e2e-<suite>` |

`make vet` is not a CI step: golangci-lint runs `govet` as part of `make lint`,
so a separate pass only compiles everything twice.

## Caching

- Go modules and build cache: `actions/setup-go` with `cache: true`.
- Tool binaries and envtest assets: keyed on
  [`hack/tool-versions.mk`](https://github.com/corewire/kick/blob/main/hack/tool-versions.mk)
  by [`.github/actions/setup-build`](https://github.com/corewire/kick/blob/main/.github/actions/setup-build/action.yml).
  Every pinned version lives in that file, so an unrelated `Makefile` edit no
  longer rebuilds every tool.
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

