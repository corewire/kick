# Argo CD ownership discovery

## Goal

Resolve an affected Deployment to exactly one Argo CD Application and its AppProject.

## Primary path

Use the workload annotation:

```text
argocd.argoproj.io/tracking-id
```

The adapter MUST parse and validate the complete resource identity represented by the annotation:

- group;
- kind;
- namespace;
- resource name;
- Application identity.

A copied, stale, or mismatching annotation MUST NOT be accepted without validation.

## Application location

Applications may exist in any namespace supported by the Argo CD installation. The adapter MUST watch or index all configured Application namespaces, or cluster-wide Applications when configured.

## AppProject location

AppProjects are read from the Argo CD control-plane namespace. The Application's `.spec.project` selects the AppProject name.

The control-plane namespace source remains an open question:

- derive from a reliable Application field if verified;
- otherwise use operator configuration.

## Fallback path

If the tracking annotation is absent or invalid:

1. Query an Application-resource reverse index using exact identity:
   `group/kind/namespace/name`.
2. If exactly one Application contains the Deployment, accept it.
3. If none match, report `OwnerUnknown`.
4. If multiple match, report `AmbiguousOwner`.

The adapter MUST NOT infer ownership from destination namespace or Git source alone.

## Application-resource index

The adapter SHOULD derive the reverse index from a supported Application status/resource representation or Argo CD resource-tree API. Which source is reliable across supported versions is an open question.

Per-event scanning of all Applications is acceptable only as an initial correctness implementation and MUST be replaced or cached before production readiness.

## Required watches

Changes to these objects MUST re-enqueue affected restart requests:

- Application ownership/resource membership;
- Application `.spec.project`;
- Application deletion;
- AppProject deletion or recreation.

## Acceptance criteria

- Valid tracking annotation resolves an Application in a non-control-plane namespace.
- Mismatching tracking annotation is rejected.
- Fallback exact membership finds one owner.
- Zero and multiple matches block action with typed reasons.
- AppProject is read from the control-plane namespace, not the Application namespace.
