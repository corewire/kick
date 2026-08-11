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
each needs a third-party control plane. Installing Argo CD v2.13.3 into the kind
cluster took roughly one minute via
[test/e2e/setup/argocd/install-argocd.sh](test/e2e/setup/argocd/install-argocd.sh),
so that rationale is weaker than it looked.

**Change.** For each integration, in decreasing order of value:

1. **Argo Rollouts** — cheapest. Install the controller, deploy a Rollout with a
   canary strategy consuming a Secret, change the Secret, assert `spec.restartAt`
   moves, the pod template is untouched, and the canary steps are *not* executed.
2. **Secrets Store CSI** — install the driver plus the `e2e-provider` test
   provider (no cloud account needed). Assert a rotation produces exactly one
   restart, and that a partially-rotated cluster produces none.
3. **Kargo** — most expensive; needs Kargo plus Argo CD plus a Git repo.

Each needs its own setup script under `test/e2e/setup/`, a Makefile target
mirroring `test-e2e-argocd`, and an exclusion range added to `test-e2e-core`.
Then flip the corresponding `e2e:` entries to `required` and delete the
rationales.

**Acceptance.** `make test-e2e-rollouts` (etc.) green, and
`make feature-coverage` reporting no `not_applicable` e2e for these features.

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
T5 is fifteen minutes and closes the documentation loop. Then T4, which is the
largest and would have caught T1. T6 and T7 are genuine but neither blocks a
release.
