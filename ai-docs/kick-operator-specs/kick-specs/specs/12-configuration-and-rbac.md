# Configuration, RBAC, and tenancy

## Operator configuration

Initial configuration should be minimal:

```yaml
argocd:
  enabled: true
  namespace: argocd
  applicationNamespaces:
    - "*"
requestRetention: 24h
rolloutTimeout: 15m
```

The exact configuration API may be command-line flags, Helm values, or a configuration object, but there must be one documented source of truth.

## Scope

KICK v1 supports namespaced workload and dependency behavior. Cluster-wide manager deployment is acceptable, but Secret and ConfigMap reads must be limited to namespaces in which KICK operates.

A future namespace selector MAY limit managed namespaces.

## Core RBAC

KICK requires:

```text
Deployments: get, list, watch, patch
ReplicaSets: get, list, watch
Secrets: get, list, watch
ConfigMaps: get, list, watch
KickRequests and observation resources: get, list, watch, create, update, patch, delete
Events: create, patch
```

It does not require Pod deletion.

## Argo CD RBAC

Applications may be in multiple namespaces:

```text
Applications: get, list, watch
```

AppProjects exist in the Argo CD control-plane namespace and SHOULD use a namespaced Role and RoleBinding where deployment topology allows it:

```text
AppProjects: get, list, watch
```

## Secret safety

- Never include Secret values in logs, events, metrics, labels, or status.
- Avoid storing plain content hashes visible to workload readers.
- Use resource names in status only when required for audit and document visibility implications.
- The controller ServiceAccount is sensitive because it reads Secrets. Installation docs MUST state this explicitly.

## Network access

The initial Argo CD adapter SHOULD use Kubernetes CRs and caches only. If the fallback later uses the Argo CD API, authentication and network policy become separate requirements.

## Acceptance criteria

- Helm chart renders least-privilege RBAC matching enabled features.
- No verbs permit Pod deletion.
- AppProject access is restricted to the configured control-plane namespace where feasible.
- Security documentation explains Secret-read privilege and data-handling rules.
