---
name: project-management
description: Use before creating, updating, starting, or closing any work item in KICK (corewire/kick): issues, bugs, features, enhancements, security risks, chores, upstream reports, sprints. Defines issue types, required fields, GitHub Project fields, and the GOOD/DONE/PROOF/REVIEW workflow.
---

# Project management for KICK

Source of truth: GitHub issues in `corewire/kick` + GitHub Project **kick**
(`gh project … 1 --owner corewire`, https://github.com/orgs/corewire/projects/1).
Milestones are not used. Planning files under `ai-docs/` are history, not a tracker.

## Issue types (one `type:` label each, template in `.github/ISSUE_TEMPLATE/`)

| Type | Use when | Must contain |
|---|---|---|
| `type:feature` | new capability | what/why, `KICK-FEAT-NNN` (new or existing), spec link |
| `type:enhancement` | change existing behaviour | current vs wanted |
| `type:bug` | KICK behaves wrong | repro, actual vs expected, versions |
| `type:security` | vulnerability or weakened safety invariant | attack path or advisory, severity, handling: mitigate / accept (ADR) / defer |
| `type:chore` | CI, release, dependencies, docs, process | current vs wanted |
| `type:upstream` | bug/limit in a dependency (Argo CD, Kargo, CSI, …) | upstream link, impact on KICK, local handling |

Every issue body has **GOOD** (quality bar), **DONE** (checkable criteria), **PROOF
plan** (commands that prove DONE). No GOOD/DONE → not ready, Status stays Backlog.

Title: imperative, specific (`Timeline shows Rollout restart time`), no ticket
prefixes except `upstream:`.

## Project fields

| Field | Values | Rule |
|---|---|---|
| Status | Backlog → Todo → In Progress → Review → Done | exactly one item In Progress per agent |
| Sprint | iteration (2 weeks) | set when pulled into a sprint; unset = backlog |
| Severity | high / med / low | required for bug, security |
| Feature | `KICK-FEAT-NNN` | required for feature; set whenever a mapped feature changes |

Blocked: label `status:blocked` + comment with blocker and link.

## Workflow

1. **Create** with `gh issue create -R corewire/kick --template <type>` or a body
   following the template; `gh project item-add 1 --owner corewire --url <issue>`;
   set Status (+ Severity, Feature).
2. **Start**: Sprint set, Status In Progress.
3. **Commit**: `<conventional-type>: <what> (#N)`, small steps. The conventional
   prefix drives the release bump ([tools/release_plan.py](../../../tools/release_plan.py)).
4. **Review**: Status Review. Spawn a subagent with the issue body + diff/commits
   + proof; it checks every DONE item and GOOD. Fix findings.
5. **Close**: comment with
   ```
   PROOF
   commits: abc123, def456
   $ <command>
   <decisive output>
   REVIEW
   subagent: <findings> → <resolution>
   ```
   then `gh issue close`, Status Done.
6. **Change of plan** (postpone, not planned, blocked): update field/label + comment
   with reason immediately. Decisions → ADR in `docs/content/docs/decisions/`.

## Useful commands

```
gh project item-list 1 --owner corewire --format json -L 300
gh project field-list 1 --owner corewire --format json
gh issue list -R corewire/kick -l type:bug
```
Field values are set via `gh project item-edit --id <item> --project-id <PVT_…>
--field-id <…> --single-select-option-id <…>` (or `--iteration-id`, `--text`).
