# Gap 2: Secrets Store CSI driver

> **Outcome: implemented.** `SecretProviderClass` is discovered from CSI volumes
> and `SecretProviderClassPodStatus` is observed behind
> `--enable-csi-integration`. Tracked as `KICK-FEAT-023`.

> **Comparison bullet.** *Your secrets are mounted through the Secrets Store CSI
> driver. Reloader watches `SecretProviderClassPodStatus`; KICK has no
> equivalent.*

## Status

**Accurate.** KICK observes only `Secret` and `ConfigMap` objects. A secret
rotated in Vault / AWS Secrets Manager / Azure Key Vault and delivered through
the CSI driver's volume mount produces no `Secret` write at all, so KICK never
sees a change and never kicks.

## Proof

### KICK sees only Secrets and ConfigMaps

The source observer reconciler dispatches over exactly two types:

```go
func (r *SourceObservationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if result, err := r.reconcileSecret(ctx, req); err != nil || result {
		return ctrl.Result{}, err
	}
	if result, err := r.reconcileConfigMap(ctx, req); err != nil || result {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
```

[internal/controller/source_observer_controller.go](internal/controller/source_observer_controller.go#L41-L48)

Dependency extraction reads `volume.Secret`, `volume.ConfigMap`, and projected
sources only — `volume.CSI` is not handled:

[internal/dependency/extractor.go](internal/dependency/extractor.go#L81-L103)

The reverse index therefore contains no entry linking a workload to a
`SecretProviderClass`:

[internal/dependency/index.go](internal/dependency/index.go#L36-L66)

### Reloader's mechanism

`SecretProviderClassPodStatus` is a first-class watched resource:

```go
// ResourceMap are resources from where changes are going to be detected
var ResourceMap = map[string]runtime.Object{
	"configmaps":                     &v1.ConfigMap{},
	"secrets":                        &v1.Secret{},
	"namespaces":                     &v1.Namespace{},
	"secretproviderclasspodstatuses": &csiv1.SecretProviderClassPodStatus{},
}
```

`stakater/Reloader`, `pkg/kube/resourcemapper.go`

Change detection hashes the provider-side object IDs and versions:

```go
func GetSHAfromSecretProviderClassPodStatus(data csiv1.SecretProviderClassPodStatusStatus) string {
	values := []string{}
	for _, v := range data.Objects {
		values = append(values, v.ID+"="+v.Version)
	...
```

`stakater/Reloader`, `internal/pkg/util/util.go`

Support is opt-in (`--enable-csi-integration`) and additionally skipped when the
CRDs are absent:

```go
if !options.EnableCSIIntegration {
	logrus.Info("Skipping secretproviderclasspodstatuses controller: EnableCSIIntegration is disabled")
	return false
```
```go
if !kube.IsCSIInstalled {
	logrus.Info("Skipping secretproviderclasspodstatuses controller: CSI CRDs not installed")
	return false
```

`stakater/Reloader`, `internal/pkg/cmd/reloader.go`

RBAC is `list` on `secretproviderclasspodstatuses` and `secretproviderclasses`:

`stakater/Reloader`, `deployments/kubernetes/chart/reloader/templates/clusterrole.yaml`

### Reloader's own caveat

> **CSI integration gap**: CSI volumes are watched at the
> `SecretProviderClassPodStatus` level, but the link back to the workload is
> indirect and may miss edge cases. Needs a direct map from SecretProviderClass
> → workloads that mount it.

`stakater/Reloader`, `CLAUDE.md`

This is the hard part, and it is where KICK can do better: `SecretProviderClassPodStatus`
is per-Pod, so the workload link must be reconstructed. KICK already has a
workload-centric reverse index, so it can build the direct map Reloader wishes
it had.

## Design notes

`SecretProviderClassPodStatus` (`secrets-store.csi.x-k8s.io/v1`) relevant
fields:

| Field | Use |
|---|---|
| `spec.podName` | Pod that triggered the mount |
| `spec.secretProviderClassName` | The `SecretProviderClass` in the same namespace |
| `status.mounted` | Whether the mount succeeded |
| `status.objects[].id` | Provider-side object identifier |
| `status.objects[].version` | Provider-side version — the change signal |

**Verify these field names with `kubectl explain secretproviderclasspodstatus`
against an installed driver before writing the spec.** Per AGENTS.md, do not
take them from memory.

Two possible workload links:

- **Via Pod owner chain** (Reloader's approach, indirect): SPCPS → Pod →
  ReplicaSet → Deployment. Fragile during rollouts, and a Pod that has not
  started yet has no status object.
- **Via workload pod template** (proposed, direct): index each workload's
  `spec.template.spec.volumes[].csi` entries where
  `driver == secrets-store.csi.x-k8s.io`, reading
  `volumeAttributes.secretProviderClass`. This gives a namespace-scoped
  `SecretProviderClass` → workloads map with no Pod involvement, exactly the
  "direct map" Reloader's notes call for.

The proposed approach fits KICK's existing structure: it is another
`DependencyRef` kind and another index, nothing more.

## Plan

### Phase 1 — Discovery

- New `dependency.Kind` value `SecretProviderClass` with
  `APIVersion: secrets-store.csi.x-k8s.io/v1`.
- Extend `volumeRefs` in [internal/dependency/extractor.go](internal/dependency/extractor.go#L81-L103)
  to emit a ref for each `volume.CSI` whose `Driver` is
  `secrets-store.csi.x-k8s.io`, using
  `volume.CSI.VolumeAttributes["secretProviderClass"]` as the name.
- New field index `kick.corewire.io/secretProviderClassReferences` in
  [internal/dependency/index.go](internal/dependency/index.go#L11-L14), and a
  `LookupConsumingWorkloads` branch for the new kind.
- Guard: the ref is only emitted when the attribute is non-empty, matching the
  existing `appendNamed` behaviour.

### Phase 2 — Observation

- New reconciler `SecretProviderClassPodStatusReconciler`, registered **only**
  when CSI integration is enabled *and* the CRDs are established. Mirror
  Reloader's two-level guard so a missing CRD never crashes the manager.
- Change signal: hash of the sorted `status.objects[].id + "=" + version` pairs,
  stored through the existing [internal/observation](internal/observation) store
  keyed by `namespace/secretProviderClassName` — **not** by SPCPS name, so the
  many Pod-level status objects of one workload collapse into one observation.
- Ignore updates where `status.mounted` is false, and ignore SPCPS deletions
  (Pod churn is not a config change).
- Reuse `enqueueDependencyChange` unchanged; consumers come from the new index.

### Phase 3 — Configuration and RBAC

- Operator flag `--enable-csi-integration` (default `false`) plus Helm value
  `csi.enabled`.
- ClusterRole rule: `get`, `list`, `watch` on
  `secretproviderclasspodstatuses` and `secretproviderclasses` in
  `secrets-store.csi.x-k8s.io`, rendered only when the value is enabled.
- Startup log line stating clearly why CSI observation is on or off.

### Phase 4 — Policy surface

Decide whether `KickPolicy.spec.discovery.dependencySelector` applies. A
`SecretProviderClass` has labels, so the existing selector can work unchanged —
but the *changed thing* is really the provider-side object, not the CR. Simplest
consistent rule: match the selector against the `SecretProviderClass` labels.

## Traceability

| ID | Name | Unit | Envtest | E2E |
|---|---|---|---|---|
| KICK-FEAT-023 | `SecretProviderClass` reference discovery from CSI volumes | required | required | required |
| KICK-FEAT-024 | `SecretProviderClassPodStatus` change observation | required | required | required |
| KICK-FEAT-025 | CSI integration is opt-in and CRD-gated | required | required | optional |

| ID | Scenario |
|---|---|
| KICK-E2E-060 | Provider secret rotated → SPCPS version changes → exactly one rollout |
| KICK-E2E-061 | Multiple Pods of one workload report SPCPS changes → still exactly one rollout |
| KICK-E2E-062 | CSI integration disabled → SPCPS change causes no rollout |
| KICK-E2E-063 | CSI CRDs absent → operator starts healthy, no error loop |

E2E needs a real driver plus a provider. Reloader's suite uses Vault with the
CSI driver installed in kind (`scripts/e2e-cluster-setup.sh`); the same shape
works here. Never print secret values in diagnostics.

## Risks

- **CRD-absent handling is the top risk.** A controller registered against a
  missing CRD takes down the manager. The gate must be checked at startup, and
  the operator must not require a restart merely because the CRDs appeared
  later — decide explicitly whether late CRD installation requires a restart
  (Reloader effectively does).
- Rotation intervals are driver-configured; e2e will be slow and flaky unless
  the rotation poll interval is set aggressively low in the test cluster.
- `volumeAttributes.secretProviderClass` is the documented attribute key, but
  verify it against the installed driver version.

## Open questions

1. Is a per-workload direct index actually sufficient, or are there mounts where
   the `SecretProviderClass` name is only resolvable per-Pod?
2. Should the CSI-synced Kubernetes `Secret` (when `secretObjects` is used) be
   treated as the change source instead? That path already works today with zero
   new code — it may cover a large share of real deployments and should be
   documented before building anything.
3. Does this belong in the operator at all, or behind a separate optional
   controller binary?
