# ADR 0003: Argo CD Compatibility Research Baseline

## Status

Accepted as implementation baseline for Task 09.

## Sources consulted

- Argo CD resource tracking docs (stable): annotation tracking-id format, installation-id, non self-referencing behavior.
- Argo CD sync windows docs (stable): selector semantics, deny precedence, timezone, OR/AND matching, manualSync, syncOverrun.
- Argo CD ApplicationSet integration docs (stable): default namespace assumptions around Argo CD control plane resources.

## Supported version matrix

| Argo CD minor | Status | Notes |
|---|---|---|
| 2.10.x | target | adapter fixtures and gate semantics expected to apply |
| 2.11.x | target | adapter fixtures and gate semantics expected to apply |
| 2.12.x | target | adapter fixtures and gate semantics expected to apply |
| <2.10 | unsupported | no compatibility guarantee in KICK v1 |
| >2.12 | best effort | must pass fixture contract before being declared supported |

## Control-plane namespace discovery decision (open question 1)

Decision:

- do not rely on a single Application status field for controller namespace discovery;
- require explicit config `argocd.namespace` for v1;
- allow future optional auto-discovery only after per-version verification.

Rationale:

- stable docs do not provide a universal guaranteed field for all modes;
- explicit config keeps owner/project lookups deterministic.

## Tracking-id parser decision (open question 5)

Decision:

- primary ownership path uses `argocd.argoproj.io/tracking-id` in annotation or annotation+label mode;
- parser requires self-reference validation against runtime workload identity;
- non self-referencing tracking-id is rejected for ownership resolution.

Canonical fixture format:

`<appName>:<group>/<kind>:<namespace>/<name>`

Example:

`my-app:apps/Deployment:default/my-deployment`

Installation ID:

- if `argocd.argoproj.io/installation-id` is configured, it must be included in ownership checks to avoid cross-instance collisions.

## Ownership fallback data source (open question 4)

Decision:

- v1 fallback source is Application managed-resource membership from Application status/resource summary data where available;
- if a cluster/operator mode omits reliable membership data, KICK returns `AmbiguousOwner` or `OwnerUnknown` instead of guessing.

Rationale:

- correctness over convenience;
- no full-cluster arbitrary inference by destination namespace or repo URL.

## Sync-window semantics decision (open question 6)

Decision:

- deny overrides allow;
- no matching windows => allow;
- if any allow windows match, sync allowed only during active allow windows;
- selector matching supports applications, namespaces, clusters with wildcard;
- default selector composition is OR, with explicit support for AND mode when `useAndOperator` is set;
- timezone honored from window config;
- `manualSync` does not grant KICK bypass in v1 automation flow.

## Rollout annotation self-heal findings (open question 7)

Decision:

- restart annotation ownership is treated as Argo-managed desired-state interaction.
- if Argo sync removes or rewrites restart timestamp, KICK reevaluates freshness and only reissues when still required.

Evidence status:

- documented expected outcomes in reusable fixture file;
- e2e verification remains required before final support declaration.

## Unsupported modes (explicit)

- label-only tracking mode without tracking-id parser support.
- ambiguous multi-instance ownership without installation-id disambiguation.
- controller namespace auto-discovery without explicit config.
- ownership inference by destination namespace/repo URL alone.

## Reusable fixture sets

- `internal/gitops/argocd/fixtures/tracking_id_cases.yaml`
- `internal/gitops/argocd/fixtures/ownership_fallback_cases.yaml`
- `internal/gitops/argocd/fixtures/sync_window_cases.yaml`
- `internal/gitops/argocd/fixtures/rollout_annotation_selfheal_cases.yaml`
