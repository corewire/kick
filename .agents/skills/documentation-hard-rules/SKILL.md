# Documentation Hard Rules Skill

Use this skill for every documentation change in this repository.

## Hard rules

1. Images
- Every image referenced in docs or README MUST be editable draw.io SVG.
- File name pattern: `*.drawio.svg`.
- Reference model: `docs/static/images/how-kick-works.drawio.svg`.

2. Freshness
- Docs change in the same PR as behavior change.
- No known stale statements allowed.

3. Concision
- Keep text short.
- Remove filler and repeated explanations.

4. Proofs for claims
- Every non-trivial claim must include a link to proof.
- Valid proof links: spec file, code path, test scenario directory, traceability mapping, generated artifact, or external authoritative source.

5. Feature coverage in docs
- Every feature must have docs and an example.
- Preferred example proof: mapped e2e scenario.

## Required proof targets

- Feature mapping: `traceability/features.yaml`
- Scenario mapping: `traceability/e2e-scenarios.yaml`
- E2E scenario source: `test/e2e/scenarios/`
- API surface: `api/v1alpha1/`
- Spec source: `ai-docs/kick-operator-specs/kick-specs/`

## Docs change checklist

- [ ] All image links are `*.drawio.svg`.
- [ ] Changed behavior has matching docs update.
- [ ] Each claim has a proof link.
- [ ] Feature points to at least one example.
- [ ] `make docs-gen-check` passes.
