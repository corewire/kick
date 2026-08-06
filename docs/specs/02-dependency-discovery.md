# Dependency discovery

## Goal

Build and maintain a complete reverse graph from supported Secrets and ConfigMaps to every in-scope Deployment that consumes them.

## Supported references

KICK MUST inspect regular and init containers for:

```text
containers[].envFrom[].secretRef
containers[].envFrom[].configMapRef
containers[].env[].valueFrom.secretKeyRef
containers[].env[].valueFrom.configMapKeyRef
initContainers[].envFrom[].secretRef
initContainers[].envFrom[].configMapRef
initContainers[].env[].valueFrom.secretKeyRef
initContainers[].env[].valueFrom.configMapKeyRef
volumes[].secret
volumes[].configMap
volumes[].projected.sources[].secret
volumes[].projected.sources[].configMap
```

## Excluded references

KICK MUST ignore:

```text
imagePullSecrets[]
```

The Secret type MUST NOT decide whether a reference is eligible. Reference location defines eligibility.

## Namespace behavior

Pod-template Secret and ConfigMap references are namespaced. KICK MUST resolve them in the Deployment's namespace.

## Optional references

References marked optional MUST still be indexed even if the object does not exist. Creation of the object later is a relevant event.

## Deduplication

If the same dependency is referenced multiple times by one Deployment, the discovery result MUST contain one dependency identity.

Identity is:

```text
apiVersion + kind + namespace + name
```

## Indexes

The controller MUST provide reverse indexes equivalent to:

```text
Secret/<namespace>/<name> -> Deployment keys
ConfigMap/<namespace>/<name> -> Deployment keys
```

Indexes MUST update when a Deployment is created, updated, or deleted.

## Pure extraction function

Implement a pure function conceptually equivalent to:

```go
func ExtractDependencies(deployment *appsv1.Deployment) []DependencyRef
```

The function MUST:

- return deterministic sorted output;
- remove duplicates;
- include optional missing references;
- exclude imagePullSecrets;
- perform no API reads.

## Acceptance criteria

- Unit tests cover every supported reference path.
- Unit tests prove imagePullSecrets are excluded.
- Unit tests cover duplicate references and optional missing references.
- Envtest proves an updated Deployment changes reverse-index lookup results without restarting the manager.
