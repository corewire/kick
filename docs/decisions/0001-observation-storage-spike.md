# ADR 0001: Change Observation Storage Spike

## Status

Accepted for implementation guidance.

## Context

Task 03 requires a decision for durable Secret/ConfigMap change observations while open questions remain about reliable Kubernetes change timestamps and baseline semantics.

Constraints:

- survive controller restart;
- distinguish relevant content changes from metadata-only updates;
- avoid workload annotations;
- prevent Secret data leakage from stored state;
- define garbage collection behavior.

## Options Considered

### Option A: Infer from Kubernetes metadata only

Candidate fields: resourceVersion, generation, managedFields time.

Pros:

- no extra storage model.

Cons:

- does not reliably separate metadata-only updates from relevant content changes;
- managedFields and generation behavior is not stable enough across all writers;
- downtime gaps are ambiguous without durable content signature state.

Decision: rejected.

### Option B: Store full object snapshots

Pros:

- exact diff after restart.

Cons:

- high storage cost;
- unacceptable Secret exposure risk.

Decision: rejected.

### Option C: Store durable observation records with canonical relevant fingerprints

Pros:

- restart-safe;
- metadata-only filtering is deterministic;
- no Secret content stored;
- independent of workload annotations.

Cons:

- requires a durable store and GC policy.

Decision: selected.

## Decision

Use an observation service behind interfaces with durable records containing:

- source identity: apiVersion/kind/namespace/name;
- last seen resourceVersion;
- last relevant resourceVersion;
- last relevant change time (controller observed time);
- canonical relevant fingerprint digest.

Digest input includes only relevant fields:

- Secret: data, type, immutable;
- ConfigMap: data, binaryData, immutable.

Metadata-only changes do not modify last relevant fields.

## Initial Baseline Semantics

Default policy is conservative:

- first observation establishes baseline;
- first observation alone does not enqueue restart.

This is isolated behind BaselinePolicy so optional source creation can be treated as RelevantChange by higher-level controllers when reference-index context proves it is a dependency appearance event.

## Durable Storage Model Proposal

Production target: dedicated observation CRD (namespaced by source namespace).

Suggested object key:

- name: <kind-lower>-<sha256(namespace/name)> to avoid long names;
- namespace: source namespace.

Suggested fields:

- spec.identity
- status.lastSeenResourceVersion
- status.lastRelevantResourceVersion
- status.lastRelevantChangeTime
- status.relevantFingerprint

No Secret data or plaintext content is persisted.

## Garbage Collection

GC behavior:

- periodic sweep lists observation records by namespace;
- if source object does not exist and reverse dependency indexes show no current consumers, delete observation record;
- retain records while source is still referenced, even if source is temporarily absent.

## Downtime Behavior

After restart, observer reads durable records and compares new objects against stored fingerprint and RV state.

Result:

- metadata-only updates remain metadata-only;
- relevant changes remain detectable even after controller downtime.

## Prototype Evidence

Prototype package: internal/observation

- interface-based observation service;
- deterministic relevant fingerprint generation;
- restart recovery test by re-instantiating observer with persisted store.

This spike intentionally does not implement full source controllers.
