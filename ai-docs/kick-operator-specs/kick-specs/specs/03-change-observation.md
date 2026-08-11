# Secret and ConfigMap change observation

## Goal

Detect relevant content changes, ignore metadata-only updates, and preserve enough durable information to compare dependency changes with workload rollouts after controller restarts.

## Relevant Secret fields

A Secret observation MUST consider:

- `.data`
- `.type`
- `.immutable`

`stringData` is not persisted and is therefore not read from stored objects.

## Relevant ConfigMap fields

A ConfigMap observation MUST consider:

- `.data`
- `.binaryData`
- `.immutable`

## Irrelevant changes

Changes only to these fields MUST NOT create a new content-change observation:

- labels
- annotations
- owner references
- finalizers
- unrelated managed fields

## Observation model

Until the open research questions are resolved, implement behind an interface:

```go
type ChangeObserver interface {
    ObserveSecret(oldObj, newObj *corev1.Secret, observedAt time.Time) ObservationResult
    ObserveConfigMap(oldObj, newObj *corev1.ConfigMap, observedAt time.Time) ObservationResult
}
```

A result should distinguish:

```text
BaselineEstablished
RelevantChange
MetadataOnlyChange
NoChange
```

## Durable state

The implementation MUST persist, directly or indirectly:

- source identity;
- last observed relevant resource version;
- last observed relevant change time;
- enough state to distinguish a future content change from a metadata-only update.

The exact API object is intentionally not fixed until Task 03 evaluates storage options. Acceptable options include a dedicated observation CRD or status attached to an internal durable object. Workload annotations are forbidden.

## Initial baseline

When a source is first observed without previous durable state:

1. Record a baseline.
2. Do not trigger restarts solely because the source is new to KICK.
3. Creation of an optional source after its reference was already indexed is not merely a baseline; it is a relevant dependency appearance and MUST enqueue consumers.

## Change time

Let `γ(s)` be the creation timestamp of a source and `τ(m)` the time the API
server recorded for the last write of field manager `m`. The **last recorded
write** of a source is

```text
λ(s) = max( γ(s), max{ τ(m) : m is a field manager of s } )
```

The change time an observation records is exactly:

| classification | recorded change time |
| --- | --- |
| `BaselineEstablished` | `λ(s)` |
| `RelevantChange` | the wall-clock instant of the observation |
| `MetadataOnlyChange`, `NoChange` | unchanged |

`λ(s)` is the tightest evidence the API server offers for when the observed
content came to be. The wall-clock instant MUST NOT be used for a baseline: it
would make freshness depend on the race between KICK starting up and the
workload rolling out. `γ(s)` alone MUST NOT be used either: a rotation that
happened before KICK first saw the source would be dated before the workload's
rollout, dismissed as fresh, and never reconsidered, because every later
observation matches that baseline.

Writes by clients that do not use server-side field management are not recorded
and therefore cannot be dated. They are picked up by the next relevant change.

## Timestamp precision

Rollout anchors are API server timestamps and are therefore second-granular.
Change times MUST be carried with sub-second precision:

- the durable observation record MUST store the change time in RFC 3339 with
  nanosecond precision;
- every API field carrying a change time — in particular
  `KickRequest.status.latestObservedDependencyChange` — MUST use a type that
  preserves sub-second precision (`metav1.MicroTime`), never `metav1.Time`;
- no code on the path from the observation to the freshness comparison may
  truncate.

Truncation to whole seconds inverts the freshness comparison for every change
that happens in the same second as the rollout it must supersede: the change is
dated back to the start of that second, compares as not newer than the rollout,
and the workload is wrongly declared fresh.

## Evaluation trigger

A workload is evaluated only when KICK observes a `BaselineEstablished` or
`RelevantChange` event for one of its in-scope sources. There is no
workload-driven evaluation. A workload created after all of its sources have
been observed therefore has no `KickRequest` at all, which is correct: its Pods
started from the current content of every source (see `04-deployment-freshness`,
"Adoption").

## Security

- Never log Secret data.
- Never expose Secret content hashes in workload annotations, events, labels, metrics, or status visible to unrelated workload readers.
- Metrics MUST avoid Secret names unless explicitly documented and bounded.

## Acceptance criteria

- Secret and ConfigMap content updates produce relevant observations.
- Metadata-only updates do not.
- Reapplying identical content does not create a relevant observation.
- Controller restart does not lose previously persisted observations.
- Optional source creation enqueues current consumers.
- A source rotated before KICK first observed it is dated to that rotation, not to its creation.
- A change recorded in the same second as the rollout it supersedes survives every persistence hop with its sub-second component intact.
