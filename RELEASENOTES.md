# Release Notes

## Fixes

- Argo Rollouts and other CRD-backed workload kinds were looked up through a
  controller-runtime field index. The manager does not cache unstructured
  objects, so the field selector reached the API server, which rejects it and
  made the whole `--enable-argo-rollouts` integration a no-op. Optional kinds
  are now listed per namespace and filtered in process.
- A dependency change observed while a `KickRequest` was being evaluated could
  be discarded by the terminal `NoLongerRequired` write, leaving the workload
  stale with nothing left to re-trigger it. The request now stays open and is
  re-evaluated against the newer change.
- A `KickRequest` no longer terminates as `NoLongerRequired` because of the
  restart it issued itself; it reaches `Succeeded` once that rollout completes.
- A dependency change was marked as observed before it had been turned into a
  `KickRequest`. Any error in between (a conflicting status write, an API
  hiccup) dropped the change permanently, because the retry compared the source
  against the already-updated baseline and saw no change. The observation is now
  committed only after the change has been enqueued, so retries are safe. A
  request is consequently evaluated against the change it was opened with and
  not only against the observation store, which still holds the previous
  baseline while that hand-off is in flight.
- Because that hand-off is at-least-once, the same dependency change could be
  enqueued twice. The second enqueue re-opened the already finished request and
  re-evaluated it against the restart it had just completed, so a `Succeeded`
  request flipped to `NoLongerRequired`. A request is now only re-opened by a
  change that is newer than the one it already recorded.
- A request that waited for a rollout it had not started (an Argo Rollouts step
  pause, a deployment already in flight) recorded that rollout as its own. The
  executor reads that field as "the restart was already issued", so once the
  foreign rollout finished the request reported `Succeeded` without ever
  restarting the workload, which kept running with the old dependency.
- Observation records and `KickRequest`s were read through the informer cache,
  which lags behind KICK's own writes. Both are read-modify-write cycles, and a
  stale read is not merely slow: an observation record that reads as missing
  makes KICK establish a second baseline, which anchors the change to the
  source's creation time and swallows it for good. Both kinds are now read
  straight from the API server.
- The change time handed to a request is the one the observation recorded
  instead of the wall-clock moment KICK happened to look. A dependency created
  alongside its workload no longer appears newer than the workload's rollout and
  therefore no longer triggers a restart on start-up.
- The first observation of a source is anchored to the last write the API server
  recorded for it, not to its creation. A source that was rotated before KICK
  first saw it (KICK was installed, restarted, or its cache had not synced yet)
  was dated to its creation, which put the change before the workload's rollout;
  the rotation was dismissed as "already fresh" and never reconsidered, because
  every later observation matched that baseline.
- The moment of the newest dependency change was kept in the `KickRequest`
  status as a `Time`, which Kubernetes serialises with whole-second precision.
  Rollout timestamps have the same granularity, so a change made in the same
  second as the rollout it had to supersede was truncated back to the start of
  that second, compared as not newer than that rollout, and the request settled
  as `NoLongerRequired` while the workload still ran the old content. This
  surfaced only when the request was re-read before the observation store had
  committed, which is exactly the at-least-once hand-off path. The field is a
  `MicroTime` and retains its sub-second component.

## Compatibility

- API versions: `kick.corewire.io/v1alpha1` — `KickPolicy`, `KickRequest`, `NotificationPolicy`
- Workload kinds: `Deployment`, `StatefulSet`, `DaemonSet`, and `argoproj.io/v1alpha1` `Rollout` (opt-in)
- Dependency sources: `Secret`, `ConfigMap`, and `SecretProviderClass` (opt-in)
- GitOps gating: optional (default `None`); adapters for Argo CD, Flux and Kargo

Kubernetes and Argo CD compatibility must be validated per release by CI/e2e evidence.

## Optional integrations

Both are disabled by default and are additionally skipped when the corresponding
CRD is absent, so enabling them on a cluster that lacks the dependency is safe.

- `--enable-argo-rollouts` (`integrations.argoRollouts.enabled`) — restarts Argo
  Rollouts through `spec.restartAt` rather than the pod template, so a
  configuration change does not re-run the canary or blue-green strategy.
- `--enable-csi-integration` (`integrations.secretsStoreCSI.enabled`) — observes
  `SecretProviderClassPodStatus` and restarts once a rotation has reached every
  pod of the class.

`integrations.kargo.enabled` grants the RBAC needed for `provider: Kargo`; Kargo
is never auto-detected and must be selected explicitly.

## Limitations

- when a GitOps provider is configured, ambiguous or missing ownership blocks automatic restart;
- historical dependency changes before KICK observation start are not reconstructed;
- dependencies cannot be declared by name — only references proven from the pod spec trigger restarts;
- KICK ships no admission webhook and therefore never holds pods `Pending`;
- OpenShift `DeploymentConfig` is not supported;
- `NotificationPolicy` delivery is best-effort: the queue is bounded and drops the
  oldest event under sustained backpressure, and a failed delivery never fails a restart;
- behavior remains alpha and may change before beta.

## Artifacts

- container image (signed)
- Helm chart package
- CRD bundle
- SBOM
- checksums