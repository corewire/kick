# Freshness

A changed dependency is not enough on its own — the running rollout might
already predate the change *or* already include it. **Freshness** is the check
that decides. It compares two timestamps:

- **Λ** — the latest *relevant* dependency change (see
  [dependency discovery](../dependency-discovery/) for what counts).
- **σ** — the current rollout's start time (the `restartedAt` stamp if present,
  otherwise the active revision's creation time), computed per workload kind.

The decision is a strict comparison:

| Condition | Verdict |
|-----------|---------|
| dependency newer than rollout (Λ > σ) | **stale** → restart required |
| dependency older than or equal to rollout (Λ ≤ σ) | **fresh** → no restart |

KICK always **re-reads live state** immediately before restarting, so a rollout
that started in the meantime is never restarted twice.

## Initial baseline

Freshness needs an authoritative change timestamp, and the first observation of
a dependency never witnessed the change that produced the content it finds. KICK
therefore dates that content from what the API server recorded: the source's
creation time, or the last write any field manager made to it, whichever is
later.

- A source untouched since creation carries exactly what the workload picked up
  when it rolled out, so it is fresh and adopting a workload never restarts it.
- A source written after its creation is dated to that write, exactly as the API
  server recorded it. Kubernetes' own timestamps are second-granular, so KICK
  keeps its change times at sub-second precision: a change made in the same
  second as a rollout is still recognised as newer.

Drift that the API server did not record - a write by a client that does not use
server-side field management - still cannot be inferred. From the baseline
onward, every relevant change advances Λ and is compared normally.

Evaluation is driven by observed changes, including the first observation of a
source. A workload created after all of its dependencies already exist is fresh
by construction — its Pods started from the current content of every source — so
no `KickRequest` is created for it. The next relevant change to one of its
sources triggers evaluation as usual.
