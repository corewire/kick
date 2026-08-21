---
title: Documentation Standards
weight: 35
description: "Hard documentation rules: editable images, freshness, concision, proof links, and feature-to-example coverage."
---

Hard rules for KICK documentation.

## Rules

1. Images
- Every image referenced from docs or README must be editable draw.io SVG named `*.drawio.svg`.
- Reference format: [how-kick-works.drawio.svg](/images/how-kick-works.drawio.svg).

2. Freshness
- Docs must change with behavior changes in the same PR.
- No known stale statements.

3. Concision
- Keep pages short.
- Remove filler and duplicates.

4. Proofs for claims
- Every non-trivial claim must link to proof.
- Proof can be spec, code, tests, traceability mapping, or generated artifact.

5. Feature coverage
- Every feature must be documented.
- Every feature must have at least one example, usually an e2e scenario.

## Proof sources

- Feature registry: [traceability/features.yaml](https://github.com/corewire/kick/blob/main/traceability/features.yaml)
- Scenario registry: [traceability/e2e-scenarios.yaml](https://github.com/corewire/kick/blob/main/traceability/e2e-scenarios.yaml)
- Scenario examples: [test/e2e/scenarios/](https://github.com/corewire/kick/tree/main/test/e2e/scenarios)
- API surface: [api/v1alpha1/](https://github.com/corewire/kick/tree/main/api/v1alpha1)
- Specs: [ai-docs/kick-operator-specs/kick-specs/](https://github.com/corewire/kick/tree/main/ai-docs/kick-operator-specs/kick-specs)

For agent workflows, use the dedicated skill at `.agents/skills/documentation-hard-rules/SKILL.md` and the repository rules in `AGENTS.md`.