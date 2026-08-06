# Task 19: Token-efficient workflow and starter repository

## Goal

Provide a low-context implementation workflow and a conservative Kubebuilder-compatible starter repository.

## Dependencies

- `specs/11-controller-architecture.md`
- `specs/14-testing-strategy.md`
- `specs/19-framework-and-test-traceability.md`
- `specs/21-token-efficient-development.md`

## Deliverables

- documented context-loading and prompt strategy;
- task-sized development sequence;
- starter Go module and manager entry point;
- `KickRequest` API skeleton;
- provider-neutral GitOps contract;
- dependency extractor with unit tests;
- controller skeleton without guessed Argo CD behavior;
- traceability checker and scenario template.

## Non-goals

- no production-ready controller;
- no guessed Secret update timestamp implementation;
- no guessed Argo CD tracking or window semantics;
- no release-ready image or chart.

## Acceptance criteria

- all Go files are formatted;
- dependency extraction unit tests are included;
- TODOs reference a specification or task rather than vague future work;
- the starter repository clearly states which parts are placeholders;
- the workflow tells agents to load only task-linked context.
