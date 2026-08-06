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
