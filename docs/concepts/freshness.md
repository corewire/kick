# Freshness

Freshness compares two timestamps:

- latest relevant dependency change;
- current rollout start time from the active ReplicaSet.

Decision:

- dependency newer than rollout => stale => kick required;
- dependency older or equal => fresh => no kick.

KICK always re-reads live state before restart execution.

Initial-baseline limitation:

- if no authoritative change timestamp exists for a dependency, KICK cannot infer historical drift from before observation started.