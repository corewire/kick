# Follow-up tasks

Work remaining after the five gap-analysis decisions were implemented
(commits `9067d5e`, `acdc70a`). Ordered by severity: T1–T3 are defects in
shipped code, T4–T7 are gaps.

Nothing here is started.

---

## T1 — `SecretProviderClass` is dropped from the freshness path

**Severity: blocking. The Secrets Store CSI feature does not work end to end.**

`dependencyToSourceIdentity` in
[internal/controller/kickrequest_controller.go](internal/controller/kickrequest_controller.go)
maps only `dependency.Secret` and `dependency.ConfigMap`, and returns
`false` for anything else:

```go
default:
    return observation.SourceIdentity{}, false
```

Its only caller, `latestRelevantChanges`, treats `false` as `continue`. So a
`SecretProviderClass` dependency never contributes a change time, the freshness
evaluator sees no relevant change, and the request resolves as
`NoLongerRequired` instead of restarting.

The rest of the chain is correct: `SecretProviderClassObservationReconciler`
writes the record under `SourceKindSecretProviderClass`, the reverse index
resolves the consuming workloads, and `dependencyLabels` scopes it. Only this
one translation is missing.

**Change.** Add `case dependency.SecretProviderClass:` returning
`observation.SourceKindSecretProviderClass`.

**Acceptance.**
- A unit test in `internal/controller/` asserting `dependencyToSourceIdentity`
  round-trips all three kinds, and that `latestRelevantChanges` returns the
  stored time for a `SecretProviderClass` dependency.
- The test must fail against the current code.

**Why it was missed.** Unit tests covered the fingerprint, the extractor and the
observer in isolation. This seam has no unit test, and T4 (the e2e that would
have caught it) was ruled `not_applicable`. Treat that as the lesson, not the
individual bug.

---

## T2 — Timeline reports CSI dependencies under the wrong kind

**Severity: wrong data shown to users.**

`sourceKind` in [internal/timeline/service.go](internal/timeline/service.go)
is a two-way branch:

```go
if kind == dependency.ConfigMap {
    return observation.SourceKindConfigMap
}
return observation.SourceKindSecret
```

A `SecretProviderClass` dependency therefore queries the observation store as
`Secret/<name>`, which either misses or — if a real Secret shares the name —
returns another object's change history.

**Change.** Make it an exhaustive `switch` over `dependency.Kind`.

**Acceptance.** Extend `internal/timeline/service_test.go` with a
`SecretProviderClass` dependency whose stored record is only findable under the
correct kind.

---

## T3 — Timeline shows no restart time for Argo Rollouts

**Severity: incomplete data.**

`workloadRestartedAt` in [internal/timeline/service.go](internal/timeline/service.go)
switches over `*appsv1.Deployment`, `*appsv1.StatefulSet`, `*appsv1.DaemonSet`
and returns `nil` by default. `loadWorkload` returns an
`*unstructured.Unstructured` for a Rollout, so it always falls through.

A Rollout's restart marker is `spec.restartAt`, not a pod-template annotation,
so this needs its own accessor rather than a fourth call to `parseRestartedAt`.

**Change.** Add an `*unstructured.Unstructured` case reading `spec.restartAt`,
guarded by `dependency.IsArgoRollout`.

**Acceptance.** Unit test in `internal/timeline/` with an unstructured Rollout
carrying `spec.restartAt`.

---

## T4 — No e2e coverage for CSI, Argo Rollouts or Kargo

**Severity: high. T1 is direct evidence this matters.**

`KICK-FEAT-023`, `024` and `025` are marked `e2e: not_applicable` in
[traceability/features.yaml](traceability/features.yaml) on the grounds that
each needs a third-party control plane. That rationale conflates two very
different levels of infrastructure, and for two of the three features the
cheaper level is sufficient.

### Two tiers

**Tier 1 — CRDs only.** Install the third-party CRDs, hand-author the
third-party objects as test fixtures. No controller runs. This is not a
watered-down test: the real CRD's OpenAPI schema validates every field name,
type and enum, so a fixture that KICK's code agrees with is a fixture the real
controller would also accept. It catches exactly the class of defect T1 belongs
to — a translation KICK never performs — and it is hermetic and fast.

**Tier 2 — real control plane.** Install the actual controller. Only needed
where the assertion is about the third party's *behaviour* rather than the
object's shape.

The tier each feature needs differs, and so does the cost.

### FEAT-025 Kargo — Tier 1, and Tier 1 is correct

Verified: Kargo ships standalone CRD manifests at
`charts/kargo/resources/crds/kargo.akuity.io_{stages,promotions}.yaml`
(tag `v1.1.1`). Those are the only two kinds KICK reads.

KICK's Kargo provider never talks to a Kargo controller. It reads
`Stage.status.currentPromotion.name`, `Promotion.spec.stage`,
`Promotion.status.phase`, and the `kargo.akuity.io/authorized-stage` annotation
on an Argo CD Application. Every one of those is writable by a test.

Running real Kargo would need cert-manager, a Git server, a container registry
and Argo CD, to end up asserting against status fields the test could have
written directly. Tier 2 here buys almost nothing.

**Infra:** `kubectl apply -f` two CRD files. Argo CD is already installed.

**Scenarios:** gate blocks on `status.currentPromotion` set; gate blocks on a
non-terminal Promotion; gate delegates to Argo CD once the Promotion is
`Succeeded`; gate returns `GateOwnerUnknown` for an unannotated Application;
gate returns `GateOwnerAmbiguous` for two authorized stages.

### FEAT-024 Argo Rollouts — Tier 2, and Tier 2 is cheap

Verified: `manifests/install.yaml` at tag `v1.7.2` exists (HTTP 200), and
`manifests/crds` exists if a Tier 1 subset is ever wanted. All images are
published and pullable.

Tier 1 would confirm KICK writes `spec.restartAt`. But the assertion that
actually justifies the feature — that restarting a Rollout via `spec.restartAt`
does **not** re-run the canary or blue-green steps, which is the whole reason we
avoided the pod-template annotation — requires the real controller to observe
the Rollout and execute, or decline to execute, its strategy.

**Infra:** one `install.yaml`, namespace `argo-rollouts`, plus a
`rollout status`-equivalent wait on `deploy/argo-rollouts`.

**Scenarios:** Secret change restarts a canary Rollout via `spec.restartAt` with
the pod template byte-identical and no canary step executed; the same for
blue-green; `ArgoRolloutComplete` gates the KickRequest until pods are actually
replaced; a Rollout using `spec.workloadRef` is handled or explicitly rejected.

### FEAT-023 Secrets Store CSI — Tier 1 first, Tier 2 with OpenBao

The e2e-provider really is unpublished, and the mechanism is now clear:
`registry.k8s.io/csi-secrets-store/e2e-provider` returns `"tags":[]` where
`csi-secrets-store/driver` and `sig-storage/livenessprobe` return populated
manifests, so the empty result is real and not an artefact of the registry API.
The staging registry `gcr.io/k8s-staging-csi-secrets-store/e2e-provider` is
empty too. Upstream's Makefile explains why: line 388 is
`kind load docker-image`, never a push. It is a build-time-only fixture.

**But that never mattered, because the provider is interchangeable.** Verified
in the driver's `pkg/secrets-store/nodeserver.go:249`: the **driver** calls
`createOrUpdateSecretProviderClassPodStatus(..., objectVersions)`, where
`objectVersions` is whatever the provider returned over gRPC. KICK reads only
that driver-authored status. So any conforming provider works, and we should
pick one with published images rather than build the upstream test fixture.

**OpenBao is that provider.** All artifacts verified published:

| Component | Artifact | Verified |
|---|---|---|
| CSI driver | `deploy/secrets-store-csi-driver.yaml` @ `v1.4.6` | HTTP 200 |
| OpenBao provider | `openbao/openbao-csi-provider:2.0.3` | Docker Hub + ghcr.io, multi-arch |
| Provider manifest | `deployment/openbao-csi-provider.yaml` @ `v2.0.3` | complete DaemonSet, namespace `csi` |
| OpenBao server | `openbao/openbao:2.6.1` | Docker Hub |

One property makes OpenBao a better fixture than the upstream e2e-provider for
our purposes. `internal/provider/provider.go:193` computes the reported version
as an HMAC of the secret content, not a KV revision number. The version string
therefore changes exactly when the content changes — a content fingerprint,
which is precisely the semantics KICK's freshness model assumes. A no-op write
produces no version change and so must produce no restart, and that becomes a
directly assertable scenario.

**Remaining Tier 2 cost, honestly stated:** bootstrapping OpenBao — enable KV
v2, enable the Kubernetes auth method, create a policy and a role. Roughly
fifteen `bao` CLI calls via `kubectl exec` against a dev-mode server, with
`test/bats/_helpers.bash` and `test/bats/configs` in the provider repo as a
working template. Non-trivial, but bounded and one-time, and far cheaper than
vendoring a Go build.

Also: the driver manifest ships `--enable-secret-rotation=false` with
`--rotation-poll-interval=2m`. Both must be patched — rotation on, interval
well under chainsaw's 180s `exec` timeout — or the test cannot observe a
rotation at all.

**Still do Tier 1 first.** `SecretProviderClassPodStatus` has no spec; it is
status-only and KICK's observer reads nothing else, so a test can write
`status.objects[].version` directly and reproduce the driver's exact signal.
That is a two-CRD, no-controller test, and **it is what catches T1**. Tier 2
then proves the real driver produces the status shape Tier 1 assumes — which is
the one thing Tier 1 cannot self-verify.

**Infra (Tier 1):** the `SecretProviderClass` and
`SecretProviderClassPodStatus` CRDs from the driver repo. Nothing else.

**Scenarios (Tier 1):** a version change across all pods' `status.objects`
triggers exactly one restart (fails today because of T1); mixed versions across
pods produce zero restarts until they converge; a pod referencing the SPC
through a CSI volume is discovered by the reverse index.

**Scenarios (Tier 2):** a real `bao kv put` that changes content produces
exactly one restart; a `bao kv put` writing identical content produces none.

### Shared plumbing

- Setup scripts under `test/e2e/setup/<tool>/`, following the existing
  [install-argocd.sh](test/e2e/setup/argocd/install-argocd.sh) contract: pinned
  version, explicit `--kubeconfig`/`--context`, never an ambient context, and a
  readiness wait.
- One Makefile target per suite mirroring `test-e2e-argocd`, plus the matching
  ID range added to the `grep -Ev` exclusion in `test-e2e-core`. That exclusion
  list is already an unreadable 24-ID alternation; adding three more ranges is
  the point at which it should become a label or directory convention instead.
- IDs from `KICK-E2E-060`. Each scenario owns its `kick-e2e-NNN` namespace.
- Keep script deadlines under the 180s `exec` timeout in
  [chainsaw-configuration.yaml](test/e2e/chainsaw-configuration.yaml).

### Ordering constraint that is easy to get wrong

The manager probes for these CRDs **once, at startup** (T7). A setup script that
installs CRDs *after* KICK is deployed produces a green-looking cluster in which
the integration is silently inactive and every scenario fails for a reason that
looks nothing like the cause. Every setup script must install CRDs before the
manager starts, or restart the manager afterwards. Worth an assertion in the
scripts rather than a comment.

**Acceptance.** `make test-e2e-kargo`, `make test-e2e-rollouts` and
`make test-e2e-csi` green; `KICK-FEAT-023` and `025` move to `e2e: required`;
`024` moves to `required`; `make feature-coverage` clean.

---

## T5 — Metrics and events references are missing the new signals

**Severity: low, purely documentation.**

[docs/content/docs/reference/metrics.md](docs/content/docs/reference/metrics.md)
does not list:

- `kick_notification_deliveries_total{namespace,policy,outcome}`
- `kick_notification_dropped_total`

and does not mention that `kick_requests_total` gained a `dry_run` value for its
`result` label.

[docs/content/docs/reference/events.md](docs/content/docs/reference/events.md)
does not list the `KickDryRun` reason.

**Change.** Add all four. Run `make docs-gen`.

**Acceptance.** `make docs-gen-check` clean; every reason in
`internal/controller/observability.go` and every metric in
`internal/metrics` and `internal/notify/metrics.go` appears in the references.

---

## T6 — `NotificationPolicy` has no reconciler

**Severity: medium, usability.**

Status is written only by `WebhookDispatcher.recordStatus`, i.e. only when an
event fires. Consequences:

- `status.observedGeneration` lags an edit indefinitely on a quiet namespace.
- A policy whose `auth.bearerToken` names a nonexistent Secret looks healthy
  until the first delivery.
- There is no `Ready` condition, so `kubectl get notificationpolicy` cannot show
  whether the policy is usable.

**Change.** Add a reconciler that, on spec change, resolves every referenced
Secret key and sets `Ready`/`ValidationFailed`. It must not send anything.

**Open question to settle first:** should an unresolvable credential suspend
delivery, or only report? Reporting only is more consistent with "a failed
notification never fails a restart".

**Acceptance.** Envtest asserting `Ready: False` with a missing Secret and
`Ready: True` once created. Requires a new `KICK-FEAT-0NN` allocation and, per
[AGENTS.md](AGENTS.md), an e2e scenario unless a rationale is recorded.

---

## T7 — Optional integrations are probed only at startup

**Severity: low, operational sharp edge.**

`newProviderRegistry` and the `optionalWorkloadKinds` block in
[cmd/main.go](cmd/main.go) call `apiprobe.KindAvailable` once, before
`mgr.Start`. Installing Argo Rollouts, the CSI driver or Kargo *after* KICK is
running leaves the integration silently inactive until the pod restarts.

This is deliberate — controller-runtime aborts the manager when an index or
watch is registered for an absent kind — but it is undocumented, and the failure
mode is silence.

**Change.** Pick one:

- **(a)** Document it in
  [docs/content/docs/reference/configuration.md](docs/content/docs/reference/configuration.md)
  and log a startup line naming each integration that was requested but skipped.
- **(b)** Watch the CRD list and self-terminate when a requested kind appears,
  letting the Deployment restart the pod.

**(a) is strongly preferred.** (b) trades a clear operational step for a
surprising self-restart.

**Acceptance.** For (a): a log line per skipped integration, plus a row in the
configuration reference.

---

## Suggested order

T1 first — it is a shipped defect that makes an advertised feature a no-op.
T2 and T3 are small and share the timeline test fixture, so do them together.
T5 is fifteen minutes and closes the documentation loop.

Then T4, in ascending order of infrastructure cost, which is the reverse of what
I assumed before checking: **CSI Tier 1** (two CRDs, and it is the test that
catches T1), then **Kargo Tier 1** (two CRDs, Argo CD already present), then
**Argo Rollouts Tier 2** (one install manifest), then **CSI Tier 2** (driver +
OpenBao + provider, all published images, plus an OpenBao bootstrap script).

T6 and T7 are genuine but neither blocks a release. T7 is worth doing before T4
regardless, because the startup-probe ordering it describes will otherwise bite
whoever writes the setup scripts.
