---
title: Freshness
weight: 30
description: How KICK decides a running rollout is stale and needs a restart.
---

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

## Initial-baseline limitation

Freshness needs an authoritative change timestamp. The first time KICK observes
a dependency it records a baseline anchored to the source's creation time — it
cannot infer drift that happened *before* observation began. From that baseline
onward, every relevant change advances Λ and is compared normally.