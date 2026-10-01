# 
# Token-efficient development workflow

## Goal

Enable a lower-cost coding agent to implement KICK in small, verifiable increments without repeatedly reading the entire specification or inventing missing behavior.

## Core rule

One agent invocation MUST address exactly one task file from `tasks/` and the smallest set of linked specification files.

Do not ask an agent to "implement KICK" or to read every document. The task file is the entry point and declares the required context.

## Context loading order

For any implementation task, read only:

1. `AGENTS.md` — global constraints.
2. The selected `tasks/NN-*.md` file.
3. Only the specification files listed under that task's `Dependencies` section.
4. The relevant feature entries from `traceability/features.yaml`.
5. Existing code in the package being changed and its direct tests.

Read other files only when the selected task or compiler output proves they are needed.

## Prompt template

Use this prompt for a coding agent:

```text
Implement task <TASK FILE> in the KICK repository.

Read in this order:
1. AGENTS.md
2. <TASK FILE>
3. only the files listed in the task's Dependencies section
4. the feature IDs referenced by the task in traceability/features.yaml
5. existing code and tests in the packages you modify

Rules:
- Do not implement later tasks.
- Do not resolve open questions by assumption.
- Keep the patch minimal.
- Add or update unit tests for every behavior changed.
- Add/update e2e scenario metadata when required by the feature matrix.
- Run the narrowest relevant checks first, then the task's full acceptance checks.
- Report changed files, tests run, unresolved blockers, and feature IDs covered.
```

## Work-unit sizing

A task should normally change:

- one production package;
- its unit tests;
- at most one API or test fixture area;
- one traceability entry when necessary.

Split a task further when it requires more than one independent reconciliation concern or more than approximately 400 lines of new production code.

## Stable interfaces first

Implement pure, provider-neutral components before controllers:

1. dependency extraction;
2. dependency identity and indexes;
3. content-change classification;
4. rollout inspection;
5. freshness evaluation;
6. GitOps provider contract;
7. Argo CD adapter;
8. KickRequest reconciliation;
9. execution and rollout observation.

Pure functions and narrow interfaces reduce the context required for later agents and make most behavior testable without Envtest.

## Test-first sequence

For each behavior:

1. identify the feature ID;
2. write or update the smallest unit test;
3. implement the pure behavior;
4. add the Envtest/controller test when API behavior is involved;
5. add or update the isolated Chainsaw scenario;
6. update traceability metadata;
7. run the coverage checker.

No feature is complete when its required test level is missing.

## Commands by feedback cost

Run checks in this order:

```text
1. go test ./path/to/changed/package
2. go test ./path/to/changed/package -run TestSpecificCase
3. go test ./...
4. make generate
5. make verify-generated
6. make test-envtest
7. make e2e-scenario SCENARIO=KICK-E2E-NNN
8. make e2e-pr
```

Do not begin with the complete e2e suite unless the task explicitly changes shared cluster setup.

## Agent handoff record

Every completed task MUST leave a short handoff in the pull request or task result:

```text
Task:
Feature IDs:
Changed packages:
Tests added/changed:
Commands executed:
Known limitations:
Next task unlocked:
```

This allows the next agent to start from repository state rather than rereading prior conversations.

## Decision records

When an implementation choice is not already specified and affects public API, persistence, ownership resolution, or compatibility:

- stop implementation;
- add the question to `specs/17-open-questions.md` or an ADR;
- implement only an interface or fake needed to continue independent work.

Do not let a coding agent silently encode a product decision.

## Generated files

Agents MUST NOT manually edit generated CRDs, deepcopy files, or generated API documentation. Modify sources and run the documented generation target.

Generated-diff checks prevent an agent from spending tokens reviewing derived output as if it were handwritten code.

## Efficient review strategy

Review by feature ID rather than by file count:

1. inspect the feature contract;
2. inspect the unit test proving local behavior;
3. inspect the e2e scenario proving cluster behavior;
4. inspect the minimal implementation path;
5. verify the traceability report.

## Parallel work

Tasks may run in parallel only when they do not modify the same API types, controller, generated manifests, or traceability entries.

Good parallel candidates:

- dependency extractor and Argo CD research spike;
- rollout inspector spike and docs tooling;
- observability after metric names are frozen, alongside Helm/RBAC.

Avoid parallel edits to:

- `api/`;
- manager setup;
- `KickRequestController`;
- generated CRDs;
- shared e2e cluster bootstrap.

## Completion definition

A task is complete only when:

- acceptance criteria pass;
- tests required by the feature matrix exist and pass;
- generated files are current;
- no unresolved assumption was hidden in code;
- the handoff record is complete.

