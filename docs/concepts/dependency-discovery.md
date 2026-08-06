# Dependency Discovery

KICK supports `apps/v1 Deployment` in the initial release.

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