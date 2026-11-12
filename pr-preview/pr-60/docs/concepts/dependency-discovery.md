# Dependency discovery

KICK supports `Deployment`, `StatefulSet`, and `DaemonSet` workloads, and
optionally `argoproj.io/v1alpha1` `Rollout` when the controller runs with
`--enable-argo-rollouts`. It discovers their dependencies automatically — you
never maintain a dependency list by hand.

For each workload it walks the Pod template and collects every `Secret` and
`ConfigMap` reached through:

- container and init-container `envFrom` references;
- container and init-container `valueFrom.secretKeyRef` / `valueFrom.configMapKeyRef`;
- `volumes[].secret.secretName` and `volumes[].configMap.name`;
- `volumes[].projected.sources[].secret.name` / `...configMap.name`;
- `volumes[].csi` with driver `secrets-store.csi.k8s.io`, which yields a
  `SecretProviderClass` dependency (see below).

`imagePullSecrets` are **excluded by design** and never trigger a restart — they
authenticate image pulls, not application config.

Each discovered source is identified by `apiVersion` + `kind` + `namespace` +
`name`, so the same underlying object referenced twice is one dependency.

An Argo `Rollout` that uses `spec.workloadRef` instead of an inline
`spec.template` has no pod spec of its own, so KICK discovers no dependencies
for it. Select the referenced workload instead.

## Only proven references

KICK restarts only on dependencies it can prove the workload consumes by reading
the pod spec. There is deliberately no way to declare an extra dependency by
name: a hand-maintained list drifts from reality, and a stale entry causes
restarts nobody can explain. If an application reads a Secret through the API,
create a `KickRequest` from whatever triggers that change instead.

## Secrets Store CSI

With `--enable-csi-integration`, KICK watches `SecretProviderClassPodStatus` and
derives a content fingerprint from the mounted object versions reported by every
pod of the class. Because a rotation reaches pods one at a time, KICK only acts
once every pod reports the same versions; while they disagree it re-checks
shortly instead of restarting mid-rotation. The first observation is anchored to
an epoch baseline, so enabling the integration never restarts anything by
itself.

## Scoping with selectors

Discovery is always automatic — KICK never needs a hand-maintained dependency
list. Two optional selectors on `spec.discovery` narrow what a policy acts on:

- `workloadSelector` picks the workloads the policy manages.
- `dependencySelector` picks which consumed Secret/ConfigMap changes trigger a
  restart (by the Secret/ConfigMap's own labels).

A workload restarts only when it consumes a changed dependency that is in both
scopes. An empty or omitted selector matches everything on its axis, so a policy
with no selectors watches every workload and every dependency. When set,
`dependencySelector` also scopes freshness: out-of-scope dependencies never mark
a workload stale.
