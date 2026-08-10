# Gap 3: Dependencies not referenced by the pod template

> **Comparison bullet.** *Your application reads a Secret through the API rather
> than through env or a volume. Reloader and Wave can both be told about such a
> dependency by name; KICK only discovers env and volume references.*

## Status

**Accurate.** KICK's discovery is purely structural: it reads the pod template.
An application that calls the Kubernetes API to read a `Secret` leaves no trace
in the pod template, so KICK has nothing to index.

## Proof

### KICK discovers only pod-template references

```go
func extractDependenciesFromPodSpec(namespace string, pod corev1.PodSpec) []DependencyRef {
	refs := make([]DependencyRef, 0)
	refs = append(refs, containerRefs(namespace, pod.Containers)...)
	refs = append(refs, containerRefs(namespace, pod.InitContainers)...)
	refs = append(refs, volumeRefs(namespace, pod.Volumes)...)
	return Normalize(refs)
}
```

[internal/dependency/extractor.go](internal/dependency/extractor.go#L48-L54)

`containerRefs` covers `envFrom` and `env`; `volumeRefs` covers `secret`,
`configMap`, and `projected` sources. There is no other input:

[internal/dependency/extractor.go](internal/dependency/extractor.go#L64-L103)

The `KickPolicy` API has no field for naming additional dependencies —
`discovery` only has `workloadSelector` and `dependencySelector`, both of which
*filter* discovered references rather than adding new ones:

[api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go#L17-L31)

### Wave's mechanism

```go
// ConfigMaps which Wave should watch
ExtraConfigMapsAnnotation = "wave.pusher.com/extra-configmaps"
// Secrets which Wave should watch
ExtraSecretsAnnotation = "wave.pusher.com/extra-secrets"
```

`wave-k8s/wave`, `pkg/core/types.go`

From the Wave README, "Advanced Features":

> If your Pod is reading some ConfigMap or Secret using the API and you want it
> to be restarted on change you can tell Wave in an annotation
> […] Wave will watch those ConfigMap or Secret and behave just like if they
> were mounted.

Values are comma-separated and may be cross-namespace
(`some-namespace/my-secret`).

### Reloader's mechanism

```go
ConfigmapUpdateOnChangeAnnotation = "configmap.reloader.stakater.com/reload"
SecretUpdateOnChangeAnnotation    = "secret.reloader.stakater.com/reload"
```

`stakater/Reloader`, `internal/pkg/options/flags.go`

Reloader also has a label-driven mode — `reloader.stakater.com/search: "true"`
on the workload plus `reloader.stakater.com/match: "true"` on the ConfigMap or
Secret — which is closer to KICK's selector philosophy than the by-name
annotations are.

## Impact

Realistic cases: applications using the Kubernetes API directly for
feature-flag ConfigMaps; sidecar-free operators reading their own config;
workloads whose `Secret` is consumed by an init container that writes it
somewhere unrelated; anything using a client-go informer over a ConfigMap.

Small share of workloads, but when it applies KICK silently does nothing, which
is the worst failure mode for a restart tool.

## Options

### Option A — Policy-level extra dependencies (recommended)

Add to `KickPolicySpec.Discovery`:

```go
// ExtraDependencies are sources that are not referenced by the pod template
// but should still trigger a restart of the matched workloads.
// +optional
ExtraDependencies []ExtraDependencyRef `json:"extraDependencies,omitempty"`
```

```go
type ExtraDependencyRef struct {
	// +kubebuilder:validation:Enum=Secret;ConfigMap
	Kind string `json:"kind"`
	// Name of the source in the policy's namespace.
	Name string `json:"name"`
}
```

Semantics: every workload matched by the policy also depends on every listed
source. Same namespace only.

Consistent with KICK's "no annotations on your workloads" positioning, which is
the differentiator called out on the comparison page. The cost is bluntness: the
extra dependency applies to all matched workloads, so a policy that selects ten
Deployments would restart all ten.

Mitigation: allow a per-entry `workloadSelector` that narrows the entry to a
subset of the policy's workloads. Adds a nested selector but keeps everything in
one object.

### Option B — Workload annotation

`kick.corewire.io/extra-secrets` / `kick.corewire.io/extra-configmaps` on the
workload, Wave-style.

Precise and familiar, but it breaks the "no annotations on your workloads" claim
that the comparison page leads with, and reintroduces the "you must own the
rendering" problem for third-party charts. Also conflicts with the AGENTS.md
constraint that KICK must not inject KICK-owned state into workloads — reading
an annotation is not injecting, so this is legal, but it muddies the story.

### Option C — Source-side opt-in (Reloader "search" style)

Label the `Secret`/`ConfigMap` (e.g. `kick.corewire.io/notify: "true"`) and let
a policy declare that labelled sources in its namespace trigger all matched
workloads.

Fully selector-based and consistent with KICK's philosophy, but it inverts
ownership: the config author decides who restarts, which is usually the wrong
person. Also it is very close to what `dependencySelector` already does, so it
risks two overlapping concepts.

**Recommendation: Option A with the optional per-entry selector.** Revisit
Option B only if real users report that policy-level granularity is unworkable.

## Plan (Option A)

### Phase 1 — API

- `ExtraDependencyRef` type, `Discovery.ExtraDependencies` field, CRD
  validation (`Kind` enum, `Name` DNS subdomain, max list length).
- Deepcopy and CRD regeneration via `make codegen`.
- API field coverage entry in
  [traceability/api-field-coverage.yaml](traceability/api-field-coverage.yaml).

### Phase 2 — Reverse lookup

The field index in [internal/dependency/index.go](internal/dependency/index.go#L36-L66)
is built from workload objects and cannot see policies. Extra dependencies must
therefore be resolved as a **second lookup path**, not folded into the index:

1. Existing path: index lookup → consumers.
2. New path: list `KickPolicy` in the changed source's namespace, keep policies
   whose `extraDependencies` name the source, expand each to its matched
   workloads via `workloadSelector` (intersected with the entry selector if
   present).
3. Union the two consumer sets, de-duplicated.

Put this in a new function beside `LookupConsumingWorkloads` so the pure
extractor stays free of client and policy concerns, per the AGENTS.md rule about
small pure functions and provider-free core packages.

### Phase 3 — Observation

No change needed. `SourceObservationReconciler` already observes every `Secret`
and `ConfigMap` in scope and only stops early when the consumer list is empty
([internal/controller/source_observer_controller.go](internal/controller/source_observer_controller.go#L72-L100)).
Feeding it a larger consumer list is sufficient.

### Phase 4 — Policy re-evaluation

Changing `extraDependencies` must re-evaluate affected workloads, matching the
existing acceptance criterion "policy updates re-evaluate affected requests
immediately" from
[tasks/20-kickpolicy.md](ai-docs/kick-operator-specs/kick-specs/tasks/20-kickpolicy.md).

### Phase 5 — Status and observability

Surface the count of extra dependencies in `KickPolicyStatus`, and set a
condition when a listed source does not exist. A typo in a name is otherwise
invisible — the whole point of this feature is dependencies that leave no trace
elsewhere.

## Traceability

| ID | Name | Unit | Envtest | E2E |
|---|---|---|---|---|
| KICK-FEAT-026 | Policy-declared extra dependencies trigger restarts | required | required | required |
| KICK-FEAT-027 | Missing extra dependency is reported on policy status | required | required | optional |

| ID | Scenario |
|---|---|
| KICK-E2E-064 | Extra `Secret` (unreferenced) changes → exactly one rollout of matched workloads |
| KICK-E2E-065 | Per-entry selector narrows the blast radius to a subset |
| KICK-E2E-066 | Removing an entry stops future restarts, cancels nothing already succeeded |
| KICK-E2E-067 | Non-existent extra dependency → policy condition set, no rollout, no error loop |

## Risks

- Blast radius: a policy-level extra dependency restarts everything the policy
  matches. The per-entry selector is a mitigation, not a fix; document it
  loudly.
- Two consumer-resolution paths must stay consistent, especially around
  `dependencySelector` filtering. Decide once: does `dependencySelector` apply
  to extra dependencies? (Proposal: no — naming a source explicitly is already
  an explicit decision.)
- Listing policies on every source change adds API load. It is a namespace-scoped
  cached list, so cost should be low, but confirm against the informer cache
  before accepting.

## Open questions

1. Cross-namespace extra dependencies — Wave allows them. KICK's policy is
   namespaced; allowing cross-namespace references would need RBAC thought.
   Proposal: same namespace only, at least initially.
2. Should key-level granularity be supported (`Secret` `foo`, key `bar` only)?
   The existing observation hashing works at object level
   ([internal/observation/hash.go](internal/observation/hash.go)); check before
   promising.
