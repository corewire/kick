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
each needs a third-party control plane.

**Decision: run the real control planes.** No hand-authored fixtures standing in
for a third-party controller. Every scenario drives the actual product and
asserts on state that product produced. Fixture-only tests would prove KICK is
self-consistent, not that it works against Kargo, Argo Rollouts or the CSI
driver — and T1 is proof that self-consistency is exactly what our current tests
already over-attest.

This is more infrastructure than the previous draft assumed. All of it is
verified below as published and installable.

### Method

For each integration, in order, and not skipping ahead:

1. **Model it.** Write down the exact state machine KICK expects from the third
   party — which fields it reads, which transitions it must tolerate, and what
   the correct behaviour is at each one. On paper, before any YAML.
2. **Verify the model against the product.** Confirm every field name, type and
   transition against the upstream source or CRD schema at a pinned version, the
   way the version and image claims in this document were confirmed. A model
   validated only against our own assumptions is what produced T1.
3. **Implement in Go.** Encode the model as unit and envtest coverage, where
   behaviour is cheap to assert exhaustively and failures point at a line.
4. **Then the e2e**, which proves the real product actually drives that state
   machine. It is the integration proof, not the behaviour matrix — e2e is too
   slow to be where edge cases live.

### Prerequisite for all three — an in-cluster Git server

This is the finding that reshapes the plan.
[render-application.sh](test/e2e/setup/argocd/render-application.sh) points
every existing Argo CD Application at `https://github.com/corewire/kick.git` on
the current branch. The existing suite therefore reads a **remote** repository
and silently depends on the branch being pushed — which it currently is not, so
those 19 scenarios are passing against a stale `main`.

That approach cannot extend to Kargo, because a real promotion **writes** to
Git, and e2e tests must never push to the project's own GitHub repository.

**Infra:** a minimal git server Deployment plus Service, with bare repositories
preseeded at startup.

**Evaluated and rejected: `mcarbonne/minimal-git-server`.** It is a genuinely
good project and fits the brief on every axis but one \u2014 actively maintained
(v2.1.17, two weeks old), MIT, published at `ghcr.io/mcarbonne/minimal-git-server:2`,
a repo-management CLI designed for scripting, truly minimal (Alpine plus shell).

It is **SSH-only**: `config.yml` takes accounts with SSH public keys, it serves
`git-shell` on port 22, and it offers no HTTP transport.

That disqualifies it, because of Kargo rather than any defect of its own.
Kargo's own docs at v1.11.1
(`docs/docs/40-operator-guide/40-security/40-managing-secrets.md`) state that
support for **SSH URLs and SSH private keys is deprecated as of v1.10.0 and
scheduled for removal in v1.13.0**, directing users to HTTPS with token-based
credentials. Building our Kargo scenarios on SSH would mean writing new tests
against a transport that is removed two minor releases from the version we are
targeting. "Production-near" has to mean near the production people will
actually be running.

It would still be the right choice if Argo CD were the only consumer \u2014 Argo CD
supports SSH repositories with no such deprecation.

**Selected: Gitea `1.27.1`** (stable tag verified on Docker Hub). It serves
authenticated smart HTTP, which is exactly the transport Kargo is standardising
on, and it is the de-facto in-cluster git server for Kubernetes test suites.

- Headless bring-up: `GITEA__security__INSTALL_LOCK=true` to skip the web
  installer, then `gitea admin user create --admin` in an init step.
- **Preseeded repositories**, not repositories created by the tests: a seed Job
  creates each bare repo and pushes a known initial commit containing the
  workload manifests. Scenarios then start from a deterministic revision instead
  of racing repository creation.
- Backed by an `emptyDir`. State must not survive the suite \u2014 a git server that
  accumulates commits across runs is a false-green generator.

Credentials are part of the model, not an afterthought. Verified in Kargo
`api/v1alpha1/labels.go:14`: Kargo reads git credentials from a Secret labelled
`kargo.akuity.io/cred-type: git`, with `repoURL`, `username` and `password` as
the supported keys (`pkg/credentials/credentials.go:8-11`). Argo CD needs its
own repository Secret. Both point at the in-cluster service, and both are real
production wiring worth covering.

This also fixes the pushed-branch dependency, so the existing Argo CD scenarios
should migrate to it. That migration is real work and should be its own task \u2014
but it removes a latent source of false green from the suite we already have.

### FEAT-024 Argo Rollouts — real controller

Verified: `manifests/install.yaml` at tag `v1.7.2` returns HTTP 200 and all
images are published.

The assertion that justifies the feature is behavioural — that restarting via
`spec.restartAt` does **not** re-run the canary or blue-green steps, which is
the entire reason we avoided the pod-template annotation. Only the real
controller can demonstrate that it declined to execute its strategy.

**Infra:** one `install.yaml`, namespace `argo-rollouts`, readiness wait on
`deploy/argo-rollouts`.

**Scenarios:** a Secret change restarts a canary Rollout with real pause steps —
assert the pods were replaced, `status.currentStepIndex` did not reset and no
new revision was created; the same for blue-green, asserting no preview service
cutover; `ArgoRolloutComplete` holds the KickRequest until the real controller
reports the replacement finished; a Rollout using `spec.workloadRef` behaves
correctly or is explicitly rejected.

### FEAT-025 Kargo — real control plane

Verified against the current release, not the stale tag in the previous draft:
latest Kargo is **v1.11.1**. `helm show chart
oci://ghcr.io/akuity/kargo-charts/kargo --version 1.11.1` pulls successfully
(digest `sha256:b8cd668e…`). Every component has an `enabled` toggle, so the
footprint can be trimmed: `controller`, `managementController` and
`webhooksServer` on; `api`, `garbageCollector` and `externalWebhooksServer` off.

cert-manager is a hard requirement, not optional: the chart documents that
`webhooksServer.tls.selfSignedCert: true` (the default) **must** have
cert-manager CRDs present, and there is no provision for running the webhooks
server without TLS. `cert-manager.yaml` at `v1.16.2` returns HTTP 200.

**Infra:** cert-manager → Kargo chart 1.11.1 → a Kargo `Project`, a `Warehouse`
subscribed to the Gitea repository, and a `Stage` whose promotion template runs
real `git-clone` / `git-commit` / `git-push` / `argocd-update` steps against an
Argo CD Application annotated `kargo.akuity.io/authorized-stage`. Argo CD is
already installed.

**Scenarios:** a real promotion in flight blocks the gate — assert KICK observes
`GateOwnerReconciling` while `status.currentPromotion` is set by the Kargo
controller, not by the test; the restart proceeds only after the Promotion
reaches `Succeeded` and Argo CD reports Synced; an unannotated Application
yields `GateOwnerUnknown`; two authorized stages yield `GateOwnerAmbiguous`.

This is the most expensive of the three and the one most likely to be flaky.
Budget for it accordingly rather than discovering it late.

### FEAT-023 Secrets Store CSI — real driver and real provider

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

**Remaining cost, honestly stated:** bootstrapping OpenBao — enable KV v2,
enable the Kubernetes auth method, create a policy and a role. Roughly fifteen
`bao` CLI calls against a dev-mode server, with `test/bats/_helpers.bash` and
`test/bats/configs` in the provider repo as a working template.

Also: the driver manifest ships `--enable-secret-rotation=false` with
`--rotation-poll-interval=2m`. Both must be patched — rotation on, interval
well under the suite's `exec` timeout — or the test cannot observe a rotation at
all.

**Infra:** CSI driver (rotation enabled, short poll interval) → OpenBao in dev
mode → the OpenBao provider DaemonSet → bootstrap script.

**Scenarios:** a real `bao kv put` that changes content produces exactly one
restart, driven end to end by the driver's own rotation loop; a `bao kv put`
writing byte-identical content produces none, because the provider's version
HMAC is unchanged; a pod mounting the SPC through a CSI volume is discovered by
the reverse index; a rotation observed on some pods but not yet others produces
no restart until they converge — which requires more than one node, see below.

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

**Timeouts need their own config.** The current `exec: 180s` in
[chainsaw-configuration.yaml](test/e2e/chainsaw-configuration.yaml) is already
tight — KICK-E2E-057 was killed at exactly 180s during this work. A real Kargo
promotion plus an Argo CD sync, or a driver rotation poll, will not fit. These
suites need a separate chainsaw configuration with longer `exec` and `assert`
budgets rather than shortened polling that produces flakes. Do not lower the
core suite's timeouts to match.

**The kind cluster needs more than one node.**
[hack/kind-config.yaml](hack/kind-config.yaml) declares a single
`control-plane`. The CSI driver is a DaemonSet and staged rotation across nodes
is a real production failure mode — modelling it needs workers. The full stack
(Argo CD, Kargo, cert-manager, Argo Rollouts, CSI driver, OpenBao, git server)
also wants the headroom. Adding workers slows cluster creation, so it may be
worth a separate integration kind config rather than changing the default.

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

## T8 — The 19 Argo CD e2e scenarios are placeholders

Found while implementing T4. `KICK-E2E-024` … `KICK-E2E-042` are named after
distinct Argo CD behaviours — `outofsync-waits`, `deny-window-waits`,
`multiple-owners-block` — but 18 of the 19 assert the same two values:

```
phase  = WaitingForOwner
reason = OwnerUnknown
```

No scenario in `test/e2e/scenarios/` references `argoproj.io` at all, so none of
them creates an Argo CD `Application` or `AppProject`. They exercise the
baseline "no owner found" path and nothing else, while `check_feature_coverage`
counts them as required e2e coverage for `KICK-FEAT-008` … `KICK-FEAT-013`.

The traceability matrix is therefore reporting coverage that does not exist.

**Two further gaps this exposed.**

1. Argo CD's default `application.resourceTrackingMethod` is `label`, so the
   `argocd.argoproj.io/tracking-id` annotation the provider reads is never
   written. Against a stock Argo CD the annotation path is dead and every
   resolution goes through the `status.resources` fallback. Scenarios must put
   the annotation in their own manifests to exercise it.
2. `ApplicationNamespaces` was not configurable — `cmd/main.go` hard-coded
   `ControlPlaneNamespace: "argocd"` and left the list empty, so an Application
   outside the Argo CD namespace could never be found. Fixed by adding
   `--argocd-namespace` and `--argocd-application-namespaces`.

**Acceptance.** Every scenario drives a real `Application` backed by the
in-cluster Gitea, and asserts the phase and gate reason its name claims.

---

## Suggested order

T1 first — it is a shipped defect that makes an advertised feature a no-op.
T2 and T3 are small and share the timeline test fixture, so do them together.
T5 is fifteen minutes and closes the documentation loop.

Then T4. The dependency order is forced rather than chosen: the **git server**
first, because Kargo cannot promote without it; then **Argo Rollouts**, which
needs nothing but its own install manifest and so validates the suite scaffolding
cheaply; then **CSI**, which adds OpenBao and the multi-node cluster; then
**Kargo**, which needs cert-manager, the git server and Argo CD together and is
the most likely to be flaky.

Each follows the method above: model, verify against the pinned upstream,
implement in Go, then e2e.

T6 and T7 are genuine but neither blocks a release. T7 is worth doing before T4
regardless, because the startup-probe ordering it describes will otherwise bite
whoever writes the setup scripts.
