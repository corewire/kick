---
title: Dependency discovery
weight: 20
description: How KICK finds the Secrets and ConfigMaps a workload consumes.
---

KICK supports `Deployment`, `StatefulSet`, and `DaemonSet` workloads and
discovers their dependencies automatically — you never maintain a dependency
list by hand.

For each workload it walks the Pod template and collects every `Secret` and
`ConfigMap` reached through:

- container and init-container `envFrom` references;
- container and init-container `valueFrom.secretKeyRef` / `valueFrom.configMapKeyRef`;
- `volumes[].secret.secretName` and `volumes[].configMap.name`;
- `volumes[].projected.sources[].secret.name` / `...configMap.name`.

`imagePullSecrets` are **excluded by design** and never trigger a restart — they
authenticate image pulls, not application config.

Each discovered source is identified by `apiVersion` + `kind` + `namespace` +
`name`, so the same underlying object referenced twice is one dependency.

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