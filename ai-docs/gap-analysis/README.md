# Gap analysis: "When to use Reloader or Wave instead"

The comparison page ([docs/content/docs/comparison.md](docs/content/docs/comparison.md))
listed five situations where KICK is the wrong tool. This folder turned each
bullet into a proven claim plus an implementation plan.

**All five are now decided and closed.** The plan documents are kept as the
evidence trail behind the decisions; the code and the user-facing docs are the
source of truth for current behaviour.

## Outcomes

| # | Gap | Plan | Decision |
|---|---|---|---|
| 1 | No Argo CD / Flux → KICK's advantage does not apply | [01](01-gitops-free-operation.md) | Was already false. Docs corrected, proven by `KICK-E2E-058` |
| 2 | Secrets mounted via Secrets Store CSI driver | [02](02-secrets-store-csi.md) | Implemented behind `--enable-csi-integration` |
| 3 | Application reads a Secret through the API | [03](03-unreferenced-dependencies.md) | **Declined.** Documented as an explicit non-goal |
| 4 | Alerting webhooks, `DeploymentConfig`, Argo Rollouts | [04](04-reloader-parity-gaps.md) | Webhooks and Argo Rollouts implemented, plus Kargo. `DeploymentConfig` declined |
| 5 | Initial restart on adoption | [05](05-adoption-without-initial-restart.md) | Was already a non-issue. Documented, plus `spec.dryRun` |

## Cross-cutting decisions, as settled

1. **KICK does not read workload annotations.** The opt-in story stays
   `KickPolicy` selectors. This is what made gap 3 a non-goal.
2. **KICK ships no admission webhook.** Blocking pod creation cluster-wide is a
   worse failure mode than the one it would prevent, so Wave's "hold pods
   `Pending`" behaviour is a non-goal.
3. **Extra kinds are opt-in and CRD-gated.** Argo `Rollout` is supported via
   `spec.restartAt`; `DeploymentConfig` is not supported.
4. **Parity is not the goal.** Gaps 3 and `DeploymentConfig` are answered with
   "use Reloader", not with code.

## Traceability, as allocated

| Feature | Name | E2E |
|---|---|---|
| KICK-FEAT-021 | Dry-run policy evaluation without restart | KICK-E2E-057 |
| KICK-FEAT-022 | Operation without a GitOps provider | KICK-E2E-058 |
| KICK-FEAT-023 | Secrets Store CSI rotation detection | none — needs the CSI driver |
| KICK-FEAT-024 | Argo Rollout workload restarts | none — needs the Rollouts controller |
| KICK-FEAT-025 | Kargo stage promotion gate | none — needs a Kargo control plane |
| KICK-FEAT-026 | NotificationPolicy webhook delivery | KICK-E2E-059 |

Features without e2e coverage carry a rationale in
[traceability/features.yaml](traceability/features.yaml) and are covered by unit
tests instead.

## Evidence index

Upstream references are file paths in the respective repositories; they were
verified in August 2026 against `stakater/Reloader` and `wave-k8s/wave` default
branches. KICK line references describe the state **before** these changes.

### KICK (this repository, pre-change)

| Fact | Source |
|---|---|
| Dependency discovery covered only `env`, `envFrom`, and mounted/projected volumes | [internal/dependency/extractor.go](internal/dependency/extractor.go) |
| `imagePullSecrets` deliberately excluded | [internal/dependency/extractor.go](internal/dependency/extractor.go) |
| Reverse index only existed for `Deployment`, `StatefulSet`, `DaemonSet` | [internal/dependency/index.go](internal/dependency/index.go) |
| Only `Secret` and `ConfigMap` were observed as change sources | [internal/controller/source_observer_controller.go](internal/controller/source_observer_controller.go) |
| Restart wrote only `kubectl.kubernetes.io/restartedAt` | [internal/executor/restart.go](internal/executor/restart.go) |
| GitOps gating is opt-in; the default provider is `None` | [api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go) |
| KICK-native cron windows work without any GitOps provider | [api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go) |
| KICK ships no admission webhook (no webhook manifests in `config/`) | absence of matches under [config/](config) |

### Stakater Reloader

| Fact | Source |
|---|---|
| Watches `SecretProviderClassPodStatus` as a change source | `pkg/kube/resourcemapper.go` (`ResourceMap`) |
| CSI support is opt-in and CRD-gated | `internal/pkg/options/flags.go` (`EnableCSIIntegration`), `internal/pkg/cmd/reloader.go` |
| CSI change detection hashes `status.objects[].id` + `version` | `internal/pkg/util/util.go` (`GetSHAfromSecretProviderClassPodStatus`) |
| Named (unreferenced) dependencies by annotation | `internal/pkg/options/flags.go` (`secret.reloader.stakater.com/reload`, `configmap.reloader.stakater.com/reload`) |
| Label-based search mode | `reloader.stakater.com/search` + `reloader.stakater.com/match` |
| Webhook instead of reload | `internal/pkg/util/util.go` — flag `--webhook-url`, "webhook to trigger instead of performing a reload" |
| OpenShift `DeploymentConfig` support | `deployments/kubernetes/chart/reloader/templates/clusterrole.yaml` (`apps.openshift.io`) |
| Argo Rollouts support | same chart, gated by `.Values.reloader.isArgoRollouts` and `argoproj.io/v1alpha1` |
| Known upstream weakness: no per-resource reload rate limiting | `CLAUDE.md` |
| Known upstream weakness: CSI → workload link is indirect | `CLAUDE.md` |

### Wave

| Fact | Source |
|---|---|
| Named (unreferenced) dependencies by annotation | `pkg/core/types.go` — `wave.pusher.com/extra-configmaps`, `wave.pusher.com/extra-secrets` |
| Mutating webhook avoids the initial restart | `README.md`, "Webhooks": *"This will prevent triggering restarts when adding the hash annotation initially."* |
| Webhook holds pods `Pending` while a required source is missing | `README.md`, "Webhooks": *"Pods will stay in state `Pending` instead of `ContainerCreating`."* |
| Kinds: Deployment, StatefulSet, DaemonSet only | `README.md`, "Compatibility" |
