# 23. Unified generation entry point

## Purpose

`make generate` MUST be the single command that produces every generated artifact in the
repository. No other invocation order, no partial target chains, no direct tool calls.

## Normative rules

1. `make generate` MUST regenerate all generated artifacts:
   - deepcopy code (`zz_generated.deepcopy.go`)
   - CRD and RBAC manifests under `config/crd/bases` and `config/rbac`
   - generated traceability data (`traceability/api-field-coverage.generated.yaml`)
   - generated agent/docs output (`llms.txt`, `llms-full.txt`, `docs/static/llms-full.txt`, `AGENTS.md`)
2. `make generate` MUST be idempotent. A second consecutive run MUST leave the working tree clean.
3. Contributors and agents MUST NOT invoke `controller-gen`, `gen-docs.sh`, or the coverage
   generators directly, and MUST NOT hand-edit generated files.
4. Narrow targets (`manifests`, `docs-gen`, `generate-deepcopy`, `api-field-coverage-gen`) MAY
   remain as internal building blocks and as dependencies of other targets (e.g. `e2e-install`),
   but MUST NOT be documented as the way to regenerate code.
5. `codegen` MUST remain as an alias of `generate` for backwards compatibility.
6. CI MUST run `make generate` followed by `git diff --exit-code` to gate generated drift.

## Test gate (non-negotiable)

Any change to generation wiring MUST keep the full test surface green:

- `make test` (unit + envtest) MUST pass.
- `make test-e2e` (Chainsaw) MUST pass on a Kind cluster.
- `make verify` MUST pass.

A generation change is not complete while any test or e2e scenario fails, is skipped, or is
weakened. Tests MUST NOT be relaxed to accommodate a generation refactor.

## Documentation rule

Every human- and agent-facing document MUST describe `make generate` as the only generation
command. References to `make codegen`, `make manifests generate`, or "deepcopy only" MUST be
removed from source documents; generated documents inherit the fix via `make generate`.
