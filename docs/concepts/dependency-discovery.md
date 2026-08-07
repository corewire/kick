# Dependency Discovery

KICK supports `Deployment`, `StatefulSet`, and `DaemonSet` workloads.

Dependencies are discovered from:

- container and init-container `envFrom` refs;
- container and init-container `valueFrom.secretKeyRef` and `valueFrom.configMapKeyRef`;
- `volumes[].secret.secretName`;
- `volumes[].configMap.name`;
- `volumes[].projected.sources[].secret.name`;
- `volumes[].projected.sources[].configMap.name`.

`imagePullSecrets` are excluded by design and never trigger kicks.

Identity model for sources:

- `apiVersion` + `kind` + `namespace` + `name`.

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