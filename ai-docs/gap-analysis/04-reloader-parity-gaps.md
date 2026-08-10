# Gap 4: Reloader features with no KICK counterpart

> **Comparison bullet.** *You need alerting webhooks, OpenShift
> `DeploymentConfig`, or Argo Rollouts. These are Reloader features with no KICK
> counterpart.*

## Status

**Accurate for all three.** They are, however, three unrelated features with
very different cost and value, and should be decided separately rather than as
one bullet.

| Feature | Recommendation |
|---|---|
| Notification webhook | Build — small, high value, fits the existing executor |
| Argo Rollouts | Consider — moderate cost, real GitOps-shop overlap |
| OpenShift `DeploymentConfig` | Decline — deprecated upstream, high cost, low overlap |

## Proof

### KICK supports exactly three kinds

Restart execution dispatches over `Deployment`, `StatefulSet`, `DaemonSet`:

```go
func rolloutComplete(workload client.Object) bool {
	switch w := workload.(type) {
	case *appsv1.Deployment:
	case *appsv1.StatefulSet:
	case *appsv1.DaemonSet:
```

[internal/executor/restart.go](internal/executor/restart.go#L156-L167)

The same three kinds are the only indexed workload types:

```go
for _, obj := range []client.Object{&appsv1.Deployment{}, &appsv1.StatefulSet{}, &appsv1.DaemonSet{}} {
```

[internal/dependency/index.go](internal/dependency/index.go#L57-L66)

The only side effect KICK produces is the `restartedAt` annotation — there is no
notification path anywhere in the executor:

```go
const restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"
```

[internal/executor/restart.go](internal/executor/restart.go#L23)

### Reloader: notification webhook

```go
cmd.PersistentFlags().StringVar(&options.WebhookUrl, "webhook-url", "",
	"webhook to trigger instead of performing a reload")
```

`stakater/Reloader`, `internal/pkg/util/util.go`

```go
// WebhookUrl is the URL to send webhook notifications to instead of performing reloads
WebhookUrl string `json:"webhookUrl"`
```

`stakater/Reloader`, `pkg/common/common.go`

Note the semantics: Reloader's webhook is *instead of* the reload, not in
addition to it. It is a delegation hook, not an alerting hook.

### Reloader: OpenShift `DeploymentConfig` and Argo Rollouts

Both are compiled in:

```go
argorolloutv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
argorollout "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
openshiftv1 "github.com/openshift/api/apps/v1"
appsclient "github.com/openshift/client-go/apps/clientset/versioned"
```

`stakater/Reloader`, `internal/pkg/testutil/kube.go`

RBAC is conditional on cluster capabilities and a chart value:

```yaml
{{- if and (.Capabilities.APIVersions.Has "argoproj.io/v1alpha1") (.Values.reloader.isArgoRollouts) }}
```

`stakater/Reloader`, `deployments/kubernetes/chart/reloader/templates/clusterrole.yaml`
(the same file carries an `apps.openshift.io` rule for `DeploymentConfig`)

E2E covers both:

`stakater/Reloader`, `test/e2e/core/workloads_test.go` —
`Entry("DeploymentConfig", Label("openshift"), utils.WorkloadDeploymentConfig)`

## Plan A — Notification webhook (recommended)

KICK's webhook should be an **outbound notification**, not Reloader's
delegation model. KICK already has durable `KickRequest` state machine
transitions; a notification is a natural projection of those transitions.

### Semantics

Fire on `KickRequest` phase transitions, with a configurable subset. Candidate
events:

| Event | Why it matters |
|---|---|
| `Blocked` | A kick has been waiting on a GitOps gate or window — the most operationally interesting signal, and something neither Reloader nor Wave can report |
| `Executing` | A restart just started |
| `Succeeded` | Rollout completed |
| `Failed` | Rollout timed out or failed |

`Blocked` is the differentiator. "Config changed 6 hours ago, restart still
gated by a freeze window" is a genuinely useful alert that no competitor can
produce.

### API surface

Cluster-scoped configuration, not per-policy, to keep the endpoint and its
credentials out of tenant-writable objects:

- Operator flag `--notification-url` plus Helm value, or
- A `KickNotification` CRD if per-policy routing is wanted later.

Start with the flag. Adding a CRD later is additive; removing one is not.

### Payload

JSON, stable schema, versioned field `apiVersion`. Include: request name and
namespace, target ref, phase, reason, gate reason, timestamps, and the
`traceparent` already stamped on each request
([internal/controller/source_observer_controller.go](internal/controller/source_observer_controller.go#L52-L70)).
**Never include Secret names' contents or hashes** — per AGENTS.md, KICK must
never log Secret data or content digests. Source object *names* are already
visible in `KickRequest`, so they are acceptable; digests are not.

### Delivery

- Bounded queue, drop-oldest with a dropped-count metric. A notification path
  must never block reconciliation.
- Retry with backoff, capped; failures are metrics + events, never a reason to
  fail a `KickRequest`.
- Optional bearer token from a `Secret`, never from a flag.
- Outbound URL is operator-configured only, so no SSRF pivot from tenant input.
  If a `KickNotification` CRD is added later, this becomes a real SSRF surface
  and needs an allowlist.

### Alternative worth considering first

Kubernetes `Event`s plus the existing metrics may already cover most of the
alerting need, via Alertmanager on the metrics. Check
[internal/controller/observability.go](internal/controller/observability.go) and
[specs/13-observability.md](ai-docs/kick-operator-specs/kick-specs/specs/13-observability.md)
before building a webhook — a `kick_request_blocked_seconds` gauge may be the
cheaper answer.

## Plan B — Argo Rollouts (consider)

Argo Rollouts users overlap heavily with KICK's target audience (Argo CD shops),
so this is the most defensible kind extension.

Work required:

1. Dependency on `argoproj.io/v1alpha1` types (client-go free — use
   controller-runtime's unstructured or the typed scheme).
2. Extraction: `Rollout.spec.template` is a `PodTemplateSpec`, so
   `extractDependenciesFromPodSpec` works unchanged — only the dispatch in
   `ExtractDependenciesForObject` needs a case
   ([internal/dependency/extractor.go](internal/dependency/extractor.go#L34-L46)).
   `Rollout.spec.workloadRef` pointing at a Deployment is the harder case and
   must be handled explicitly or explicitly rejected.
3. Index registration for the new kind.
4. Freshness: [internal/freshness/evaluator.go](internal/freshness/evaluator.go)
   compares change time against the current rollout's start. Argo Rollouts has
   its own progression model (`status.currentPodHash`, pause states, analysis
   runs). This is the real cost — decide what "restart already happened" means
   for a paused canary.
5. Execution: writing `restartedAt` into `spec.template.metadata.annotations`
   triggers a Rollout, but Argo Rollouts also has a native
   `spec.restartAt` field. Using the native field is more correct and avoids
   fighting the controller. Verify with `kubectl explain rollout.spec.restartAt`
   before committing.
6. Conditional RBAC + CRD-presence gate, same pattern as
   [02-secrets-store-csi.md](ai-docs/gap-analysis/02-secrets-store-csi.md).

Gate this behind CRD presence, and do not compile Argo Rollouts types into the
core packages — keep it behind the same kind-dispatch boundary the existing
three kinds use.

## Plan C — OpenShift `DeploymentConfig` (recommended: decline)

`DeploymentConfig` is deprecated by Red Hat in favour of `Deployment`. Adding it
means an OpenShift API dependency, an OpenShift-only e2e lane, and a fourth
rollout-completion model, for a shrinking user base.

Recommended position: document it as an explicit non-goal on the comparison
page, and point `DeploymentConfig` users at Reloader. Revisit only on concrete
user demand.

## Traceability

| ID | Name | Unit | Envtest | E2E |
|---|---|---|---|---|
| KICK-FEAT-028 | Outbound notification on `KickRequest` transitions | required | required | required |
| KICK-FEAT-029 | Notification delivery never blocks or fails reconciliation | required | optional | required |
| KICK-FEAT-030 | Argo Rollouts as a managed workload kind | required | required | required |

| ID | Scenario |
|---|---|
| KICK-E2E-068 | Config change → `Executing` and `Succeeded` notifications delivered in order |
| KICK-E2E-069 | Gate blocks a kick → `Blocked` notification delivered once, not per requeue |
| KICK-E2E-070 | Notification endpoint unreachable → rollout still succeeds, drop metric increments |
| KICK-E2E-071 | Argo `Rollout` with a changed ConfigMap → exactly one Rollout restart |

`DeploymentConfig` gets no IDs — it is proposed as a non-goal.

## Risks

- A notification webhook is an easy place to leak. Enforce the "no Secret data,
  no digests" rule in code and assert it in a unit test on the payload builder.
- "Delivered once, not per requeue" is the hard correctness property.
  Notifications must be driven by observed phase transitions, not by reconcile
  invocations — otherwise a blocked request spams the endpoint every requeue
  interval.
- Argo Rollouts freshness semantics are genuinely unresolved; treat step 4 above
  as research before committing to the feature.

## Open questions

1. Does `Rollout.spec.restartAt` interact correctly with an in-progress canary,
   or does it abort it? Must be answered before promising Argo Rollouts support.
2. Should notifications be Reloader-style *instead of* restarting (a dry-run /
   approval mode)? That is arguably a more interesting feature than alerting:
   "KICK detected staleness, tell my system, let it decide."
3. Are Kubernetes Events plus metrics sufficient, making the whole webhook
   unnecessary?
