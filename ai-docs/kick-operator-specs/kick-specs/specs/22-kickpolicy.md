# KickPolicy Specification

## Purpose
`KickPolicy` enables KICK for selected workloads in a namespace.

KICK ignores workloads in namespaces that do not contain a matching `KickPolicy`.

The policy defines:

- how workloads and their Secret or ConfigMap dependencies are discovered;
- which workloads in the namespace are managed;
- how the GitOps owner is resolved;
- whether provider reconciliation state and schedules must be respected;
- how frequently the same workload may be kicked.
Users do not need to list individual Secrets, ConfigMaps, or workload dependencies when using automatic discovery.

---

## Minimal policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: payments
spec: {}
```
An empty spec is valid. It enables KICK for all supported workloads in the
`payments` namespace, treats every discovered dependency as a trigger, and
restarts immediately when one changes (no GitOps gate).

Every field is optional. Narrow scope or add gating as needed:

- `discovery.workloadSelector` — which workloads are managed;
- `discovery.dependencySelector` — which dependency changes trigger a restart;
- `gitOps.provider` — defer the restart decision to a GitOps tool.

---

## Complete example

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: production
  namespace: payments
spec:
  discovery:
    workloadSelector:
      matchLabels:
        kick.corewire.io/enabled: "true"

  gitOps:
    provider: ArgoCD
    requireReconciled: true
    schedule:
      source: Provider

  minInterval: 30s
```

---

# API structure

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: string
  namespace: string
spec:
  discovery:
    workloadSelector: {}
    dependencySelector: {}

  gitOps:
    provider: None
    requireReconciled: true
    schedule:
      source: Provider

  minInterval: 30s
```

---

# Namespace semantics
`KickPolicy` is namespaced.

A policy may manage only workloads in its own namespace.

It must not select or restart workloads in another namespace.

Dependencies are also namespaced. A namespaced workload may reference only Secrets and ConfigMaps in its own namespace through normal Pod references.

A namespace without a valid `KickPolicy` is outside KICK's management scope.

KICK must not:

- build dependency-index entries for unmanaged workloads;
- create kick requests for unmanaged workloads;
- restart unmanaged workloads.

---

# `spec.discovery`

## Purpose
The `discovery` section defines how KICK discovers workloads and their dependencies.

```yaml
spec:
  discovery:
    workloadSelector: {}
    dependencySelector: {}
```

---

## Discovery is always automatic
KICK always discovers, without any manual list:

1. supported workloads selected by the policy;
2. Secrets and ConfigMaps referenced by those workloads;
3. reverse relationships from each dependency to its consuming workloads.

There is no discovery `mode` field. Two optional selectors scope the policy; an
empty or omitted selector matches everything on its axis:

- `workloadSelector` — which workloads are managed (the actors that may restart);
- `dependencySelector` — which consumed Secret/ConfigMap changes trigger a restart.

A workload restarts when it consumes a changed dependency, the workload is in
`workloadSelector` scope, and the changed dependency is in `dependencySelector`
scope.

---

## Automatic workload discovery
The initial supported workload kind is:

```text
Deployment
```
Future versions may add:

```text
StatefulSet
DaemonSet
```
Support for a new workload kind must define:

- how its current rollout is identified;
- how rollout completion is detected;
- how a kick is triggered;
- all required unit and e2e tests.

---

## Automatic dependency discovery
KICK discovers Secrets and ConfigMaps referenced through the workload's Pod template.

The following references are dependencies:

```text
spec.template.spec.containers[].envFrom[].secretRef
spec.template.spec.containers[].envFrom[].configMapRef

spec.template.spec.containers[].env[].valueFrom.secretKeyRef
spec.template.spec.containers[].env[].valueFrom.configMapKeyRef

spec.template.spec.initContainers[].envFrom[].secretRef
spec.template.spec.initContainers[].envFrom[].configMapRef

spec.template.spec.initContainers[].env[].valueFrom.secretKeyRef
spec.template.spec.initContainers[].env[].valueFrom.configMapKeyRef

spec.template.spec.volumes[].secret
spec.template.spec.volumes[].configMap

spec.template.spec.volumes[].projected.sources[].secret
spec.template.spec.volumes[].projected.sources[].configMap
```
The following is explicitly not a workload dependency:

```text
spec.template.spec.imagePullSecrets[]
```
Changing an image-pull Secret must not trigger a kick.

The Secret's `type` does not determine whether it is a dependency. The reference location in the Pod template determines the behavior.

---

## `spec.discovery.workloadSelector`
Type:

```text
metav1.LabelSelector
```
Required:

```text
No
```
Default:

```text
{}
```
An empty selector matches every supported workload in the policy namespace.

Example:

```yaml
spec:
  discovery:
    workloadSelector:
      matchLabels:
        kick.corewire.io/enabled: "true"
```
Example using expressions:

```yaml
spec:
  discovery:
    workloadSelector:
      matchExpressions:
        - key: environment
          operator: In
          values:
            - production
            - staging
```
The selector is applied to workload metadata labels.

The selector must be reevaluated when:

- the policy changes;
- workload labels change;
- a workload is created;
- a workload is deleted.

---

## `spec.discovery.dependencySelector`
Type:

```text
metav1.LabelSelector
```
Required:

```text
No
```
Default:

```text
{}
```
An empty selector treats every discovered dependency as a trigger. When set, only
Secret/ConfigMap changes whose object labels match count as triggers.

`dependencySelector` also scopes freshness: a dependency outside the selector
never marks a workload stale, so out-of-scope changes neither create a kick nor
keep one alive.

Example — only rotate on Secrets labelled for KICK:

```yaml
spec:
  discovery:
    dependencySelector:
      matchLabels:
        kick.corewire.io/watch: "true"
```

---

# `spec.gitOps`

## Purpose
The `gitOps` section defines whether and how KICK gates a restart on a GitOps
system. The whole section is optional; when omitted, `provider` defaults to
`None` and KICK gates only on its own (native windows if any, otherwise it
restarts as soon as a dependency is stale).

```yaml
spec:
  gitOps:
    provider: None
    requireReconciled: true
    schedule:
      source: Provider
```

---

## `spec.gitOps.provider`
Type:

```text
string
```
Required:

```text
No
```
Default:

```text
None
```
Supported values:

```text
None
Auto
ArgoCD
Flux
```

---

## `None`
KICK does not consult any GitOps system. The restart is gated only by KICK-native
schedule windows (if configured); with no windows a stale dependency restarts
immediately. This is the default, so a policy without a `gitOps` block works
standalone — no Argo CD or Flux required.

---

## `Auto`
KICK automatically determines the GitOps provider from workload ownership metadata.

Initial detection order:

1. Argo CD ownership;
2. Flux ownership;
3. no supported owner found.
Exactly one provider must own the workload.

Possible outcomes:

### One provider found
KICK uses the corresponding provider adapter.

### No provider found
The workload is blocked with:

```text
GitOpsOwnerUnknown
```

### Several providers found
The workload is blocked with:

```text
AmbiguousGitOpsOwner
```
KICK must not select a provider arbitrarily.

---

## `ArgoCD`
KICK resolves only Argo CD ownership.

The primary discovery mechanism is:

```text
argocd.argoproj.io/tracking-id
```
KICK must validate that the tracking identity matches the actual workload:

- group;
- kind;
- namespace;
- name.
If annotation-based resolution fails, KICK may use its cached Application-resource index to find an exact resource match.

Exactly one owning Application must be found.

KICK then:

1. reads `Application.spec.project`;
2. resolves the corresponding AppProject;
3. evaluates the Application's reconciliation state;
4. evaluates applicable AppProject sync windows.
Applications may exist in different namespaces.

AppProjects exist in the Argo CD control-plane namespace.

The mechanism for determining the Argo CD control-plane namespace remains subject to the corresponding research specification.

---

## `Flux`
KICK resolves only Flux ownership.

The provider adapter must eventually support ownership resolution for at least:

```text
Kustomization
HelmRelease
```
Until the Flux adapter is implemented, a policy using:

```text
provider: Flux
```
must expose:

```text
ProviderNotImplemented
```
and must not restart workloads.

Flux schedules are considered equivalent to provider-controlled execution windows at the generic gate level, even if their native semantics differ from Argo CD sync windows.

---

## `spec.gitOps.requireReconciled`
Type:

```text
boolean
```
Required:

```text
No
```
Default:

```text
true
```
Example:

```yaml
spec:
  gitOps:
    provider: Auto
    requireReconciled: true
```
When enabled, KICK waits until the resolved GitOps owner has fully completed its reconciliation before determining whether a kick is still required.

### Argo CD interpretation
The Application must satisfy:

```text
Application.status.sync.status == Synced
```
and no sync operation may currently be active.

Application health is not required by default.

A `Degraded` Application may still require a kick to receive updated credentials or configuration.

### Flux interpretation
The Flux owner must eventually satisfy provider-specific conditions equivalent to:

- reconciliation is not running;
- the resource is ready;
- `status.observedGeneration` matches the current generation.

### Recalculation requirement
When the GitOps owner becomes reconciled, KICK must not immediately restart the workload.

It must first re-read:

- the workload;
- its current rollout;
- all currently referenced dependencies.
If the GitOps reconciliation already created a newer rollout, the kick may no longer be required.

---

## `spec.gitOps.schedule.source`
Type:

```text
string
```
Required:

```text
No
```
Default:

```text
Provider
```
Supported values:

```text
Provider
None
```

### `Provider`
KICK obtains scheduling or window information from the resolved GitOps provider.

For Argo CD:

```text
Application
→ Application.spec.project
→ AppProject
→ AppProject.spec.syncWindows
```
For Flux:

```text
Flux owner
→ supported Flux scheduling mechanism
```
The provider adapter translates its native behavior into a generic gate decision.

### `None`
Provider schedule or window information is ignored.

The `requireReconciled` setting still applies independently.

Example:

```yaml
spec:
  gitOps:
    provider: ArgoCD
    requireReconciled: true
    schedule:
      source: None
```
This configuration waits for the Application to be reconciled but does not enforce AppProject sync windows.

---

# `spec.minInterval`
Type:

```text
duration
```
Required:

```text
No
```
Default:

```text
30s
```
Minimum:

```text
0s
```
Example:

```yaml
spec:
  minInterval: 1m
```
The interval has two purposes:

1. collapse several dependency changes into one kick;
2. prevent repeated kicks of the same workload within a short period.
Example:

```text
10:00:00 Secret A changes
10:00:05 ConfigMap B changes
10:00:12 Secret C changes
```
With:

```yaml
minInterval: 30s
```
KICK creates or updates one pending request for the workload.

The request is evaluated after the interval.

The interval is advisory. KICK must always recalculate current workload freshness before performing the kick.

A value of:

```yaml
minInterval: 0s
```
disables interval-based coalescing.

---

# Workload freshness
The policy does not store or configure dependency versions.

KICK determines whether a workload requires a kick using current and observed cluster state.

A workload requires a kick when at least one currently consumed Secret or ConfigMap changed after the workload's current rollout.

Conceptually:

```text
kickRequired =
    latestRelevantDependencyChange > currentRolloutCreationTime
```
A kick is not required when every currently consumed dependency is older than or equal to the current rollout.

The mechanism for determining dependency content-change time remains defined by the dedicated source-observation specification and research task.

---

# Policy matching
For each supported workload, KICK evaluates all `KickPolicy` resources in the workload's namespace.

## No matching policy
The workload is unmanaged.

KICK must not create a request or restart it.

## Exactly one matching policy
The policy controls the workload.

## Multiple matching policies
Overlapping policies are not supported in `v1alpha1`.

The workload must be blocked with:

```text
ConflictingPolicies
```
KICK must not merge:

- provider configuration;
- schedule behavior;
- selectors;
- minimum intervals.
The policy status must identify conflicting policies and affected workloads.

---

# Policy updates
Updating a policy must immediately affect future reconciliation decisions.

## Selector changes
When a workload begins matching a policy:

- it becomes managed;
- its dependencies are indexed;
- no kick is triggered solely because it was newly discovered;
- initial dependency observations follow the baseline rules.
When a workload stops matching a policy:

- it becomes unmanaged;
- active dependency-index relationships are removed;
- pending requests controlled by that policy are cancelled with:

```text
PolicyNoLongerMatches
```

## Provider changes
Changing:

```text
provider: Auto
```
to:

```text
provider: ArgoCD
```
must re-enqueue affected pending requests.

## Reconciliation-gate changes
Changing:

```text
requireReconciled: true
```
to:

```text
requireReconciled: false
```
must immediately re-evaluate requests blocked only by GitOps reconciliation state.

## Schedule changes
Changing:

```yaml
schedule:
  source: Provider
```
to:

```yaml
schedule:
  source: None
```
must immediately re-evaluate requests blocked by a provider schedule.

## Minimum-interval changes
Changing `minInterval` must cause pending request timings to be recalculated.

Previously calculated wake-up times are advisory and must not remain authoritative after a policy update.

---

# Policy deletion
Deleting a policy must not directly modify workloads.

When the last matching policy is deleted:

- the workload becomes unmanaged;
- no future kick is created;
- existing pending requests are cancelled with:

```text
PolicyDeleted
```
A completed kick is not rolled back.

No finalizer is required unless future versions introduce external resources that require cleanup.

---

# Status
Recommended status shape:

```yaml
status:
  observedGeneration: 4

  matchedWorkloads: 12
  blockedWorkloads: 1

  conditions:
    - type: Ready
      status: "True"
      reason: Accepted
      message: Policy is valid and active.

    - type: ConflictFree
      status: "True"
      reason: NoConflicts
      message: No selected workload is matched by another policy.

    - type: ProviderAvailable
      status: "True"
      reason: Available
      message: All selected provider adapters are available.
```

---

## `Ready`
Indicates whether the policy is valid and usable.

Possible reasons:

```text
Accepted
InvalidConfiguration
InvalidSelector
UnsupportedProvider
```

---

## `ConflictFree`
Indicates whether selected workloads overlap with another policy.

Possible reasons:

```text
NoConflicts
ConflictingPolicies
```

---

## `ProviderAvailable`
Indicates whether the configured provider adapter is available.

Possible reasons:

```text
Available
ProviderNotImplemented
ProviderConfigurationMissing
```

---

# Validation
The CRD must reject:

- unsupported provider values;
- unsupported schedule sources;
- invalid label selectors;
- negative `minInterval` values.

`discovery` and `gitOps` are optional; an omitted `gitOps` defaults `provider` to
`None`.

Valid minimal configuration:

```yaml
spec: {}
```
Also valid — scope by workload and/or dependency, and optionally gate on a provider:

```yaml
spec:
  discovery:
    workloadSelector:
      matchLabels:
        app: web
    dependencySelector:
      matchLabels:
        kick.corewire.io/watch: "true"
  gitOps:
    provider: Auto
```
Invalid configuration — an unsupported provider value:

```yaml
spec:
  gitOps:
    provider: Full
```
`Full` is not a supported provider; supported values are `None`, `Auto`,
`ArgoCD`, and `Flux`.

---

# API conventions
KICK follows standard Kubernetes API naming conventions.

## Field names
Lower camel case:

```text
workloadSelector
requireReconciled
minInterval
```

## Enum values
Upper camel case:

```text
Auto
ArgoCD
Flux
Provider
None
```

## Resource names
Lowercase DNS-compatible names:

```text
default
production-applications
payments
```

---

# Required unit tests
Feature ID:

```text
KICK-FEAT-POLICY
```
Required cases:

1. an empty spec is accepted (discovery and gitOps are optional);
2. missing `gitOps` defaults `provider` to `None`;
3. `provider: None` is accepted;
4. `provider: Auto` is accepted;
5. `provider: ArgoCD` is accepted;
6. `provider: Flux` is accepted;
7. unsupported provider is rejected;
8. default workload selector matches all workloads;
9. configured workload selector matches eligible workloads;
10. configured workload selector excludes non-matching workloads;
11. an empty `dependencySelector` treats every dependency as a trigger;
12. a configured `dependencySelector` restarts only consumers of matching dependencies;
13. a change to an out-of-scope dependency does not create a kick;
14. a policy cannot select a workload in another namespace;
15. `requireReconciled` defaults to true;
16. schedule source defaults to `Provider`;
17. `minInterval` defaults to `30s`;
18. negative `minInterval` is rejected;
19. one matching policy is accepted;
20. several matching policies produce `ConflictingPolicies`;
21. selector updates recalculate managed workloads;
22. provider updates re-enqueue pending requests;
23. reconciliation-gate updates re-enqueue pending requests;
24. schedule-source updates re-enqueue pending requests;
25. interval updates recalculate request timing;
26. policy deletion cancels pending requests.

---

# Required e2e scenarios

## `KICK-E2E-POLICY-001`
A namespace without a `KickPolicy` is ignored.

## `KICK-E2E-POLICY-002`
A policy with an empty spec enables all matching workloads and restarts them on a
dependency change without a GitOps provider.

## `KICK-E2E-POLICY-003`
An empty workload selector manages all supported workloads in the namespace.

## `KICK-E2E-POLICY-004`
A workload selector restricts management to matching workloads.

## `KICK-E2E-POLICY-005`
The policy does not affect another namespace.

## `KICK-E2E-POLICY-006`
Unsupported field values (e.g. an unknown provider) are rejected by the API server.

## `KICK-E2E-POLICY-007`
Two policies matching the same workload block the workload.

## `KICK-E2E-POLICY-008`
A conflicting workload is not restarted.

## `KICK-E2E-POLICY-009`
Deleting the only matching policy prevents future kicks.

## `KICK-E2E-POLICY-010`
Deleting a policy cancels its pending requests.

## `KICK-E2E-POLICY-011`
Changing the selector activates a previously unmanaged workload.

## `KICK-E2E-POLICY-012`
Changing the selector removes a previously managed workload.

## `KICK-E2E-POLICY-013`
Changing the provider causes blocked requests to be re-evaluated.

## `KICK-E2E-POLICY-014`
Changing `requireReconciled` causes blocked requests to be re-evaluated.

## `KICK-E2E-POLICY-015`
Changing the schedule source causes blocked requests to be re-evaluated.

## `KICK-E2E-POLICY-016`
Changing `minInterval` recalculates pending request timing.

## `KICK-E2E-POLICY-017`
Policy state and workload matching are reconstructed after controller restart.

## `KICK-E2E-POLICY-018`
Policy status reports matched and conflicting workloads.

---

# Acceptance criteria
The `KickPolicy` feature is complete when:

- `discovery` and `gitOps` are optional; an empty spec is valid;
- `gitOps.provider` defaults to `None`, gating on KICK's own logic;
- namespaces without a policy are ignored;
- automatic dependency discovery works for selected workloads;
- `imagePullSecrets` are excluded;
- `workloadSelector` correctly limits which workloads are managed;
- `dependencySelector` correctly limits which dependency changes trigger a restart;
- cross-namespace workload selection is impossible;
- exactly one policy controls each managed workload;
- overlapping policies block rather than merge behavior;
- provider reconciliation and provider schedules are independently configurable;
- policy updates immediately re-evaluate affected requests;
- policy deletion safely disables future processing;
- status exposes validity and conflicts;
- every specified behavior has unit tests where applicable;
- every externally observable behavior has e2e coverage;
- the feature-to-test traceability matrix contains all policy scenarios.
