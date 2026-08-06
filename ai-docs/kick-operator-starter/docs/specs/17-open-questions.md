# Open questions

These items MUST be researched and decided explicitly. Implementation should isolate them behind interfaces or configuration until resolved.

## 1. Argo CD control-plane namespace discovery

Verify whether the Argo CD Application object reliably exposes the Argo CD main/control-plane namespace across supported versions.

Investigate fields such as `status.controllerNamespace` and determine:

- whether the field exists in released CRDs;
- whether it is always populated;
- whether it works with Applications-in-any-namespace;
- which Argo CD versions support it.

Fallback: require `argocd.namespace` configuration.

## 2. Secret and ConfigMap change metadata

Verify which fields or annotations can reliably tell whether relevant content changed and when it changed.

Evaluate:

- `metadata.resourceVersion`;
- `metadata.generation`;
- `metadata.managedFields[].time`;
- audit events;
- watch receive time;
- External Secrets Operator annotations/status;
- Secret Store CSI status objects where applicable.

Required conclusion:

- reliable content-change detection source;
- reliable timestamp source, or confirmation that KICK must persist observed time;
- behavior across operator downtime;
- metadata-only update handling.

## 3. Current ReplicaSet selection

Verify the most robust way to identify the ReplicaSet representing the Deployment's current Pod template during normal operation, rollout, rollback, pause, and history cleanup.

## 4. Application ownership fallback

Verify which Application status field or Argo CD API endpoint reliably exposes exact managed-resource membership across supported versions.

Determine whether a cache built from Application CR status is enough or whether KICK needs the Argo CD resource-tree API.

## 5. Tracking annotation format

Verify exact tracking-ID formats for:

- annotation tracking;
- annotation plus label tracking;
- Applications in the control-plane namespace;
- Applications in other namespaces;
- installation IDs;
- long Application names.

Prefer reuse of Argo CD parsing code where stable.

## 6. Sync-window compatibility

Verify all semantics required for exact compatibility:

- allow and deny precedence;
- selectors;
- cluster names versus server URLs;
- time zones;
- AND/OR matching;
- `manualSync`;
- sync overrun;
- behavior when windows change during an operation.

## 7. Restart annotation ownership

Verify Argo CD behavior for the standard rollout annotation under:

- normal apply and self-heal;
- server-side apply;
- replace sync;
- force sync;
- charts that declare the same annotation.

## 8. Durable observation storage

Select the simplest durable storage model that:

- survives controller restart;
- distinguishes content from metadata changes;
- avoids workload annotations;
- scales to the target cluster size;
- has clear garbage collection.

## 9. Initial baseline semantics

Decide whether the conservative baseline behavior is acceptable: pre-existing sources do not trigger a kick until KICK observes a later relevant change.

## 10. Naming of API resources

Decide final names:

- `KickPolicy` versus a configuration-only model;
- `KickRequest` versus `RestartRequest`;
- observation CRD name if one is required.
