# Gap analysis: "When to use Reloader or Wave instead"

The comparison page ([docs/content/docs/comparison.md](docs/content/docs/comparison.md))
lists five situations where KICK is the wrong tool. This folder turns each of
those five bullets into a proven claim plus an implementation plan, so we can
decide which gaps to close, which to close partially, and which to keep as
documented non-goals.

Nothing here is implemented. These are proposals for discussion.

## The five gaps

| # | Gap | Plan | Recommendation |
|---|---|---|---|
| 1 | No Argo CD / Flux → KICK's advantage does not apply | [01-gitops-free-operation.md](ai-docs/gap-analysis/01-gitops-free-operation.md) | Already mostly closed — fix the docs, prove it with e2e |
| 2 | Secrets mounted via Secrets Store CSI driver | [02-secrets-store-csi.md](ai-docs/gap-analysis/02-secrets-store-csi.md) | Implement, optional controller |
| 3 | Application reads a Secret through the API | [03-unreferenced-dependencies.md](ai-docs/gap-analysis/03-unreferenced-dependencies.md) | Implement, policy-level (no workload annotations) |
| 4 | Alerting webhooks, `DeploymentConfig`, Argo Rollouts | [04-reloader-parity-gaps.md](ai-docs/gap-analysis/04-reloader-parity-gaps.md) | Split: webhooks yes, Argo Rollouts maybe, `DeploymentConfig` no |
| 5 | Initial restart on adoption | [05-adoption-without-initial-restart.md](ai-docs/gap-analysis/05-adoption-without-initial-restart.md) | Investigate — may already be a non-issue |

## Evidence index

Every claim in the comparison page is backed by a source below. Upstream
references are file paths in the respective repositories; they were verified in
August 2026 against `stakater/Reloader` and `wave-k8s/wave` default branches.

### KICK (this repository)

| Fact | Source |
|---|---|
| Dependency discovery covers only `env`, `envFrom`, and mounted/projected volumes | [internal/dependency/extractor.go](internal/dependency/extractor.go#L48-L103) |
| `imagePullSecrets` deliberately excluded | [internal/dependency/extractor.go](internal/dependency/extractor.go#L9-L11) |
| Reverse index only exists for `Deployment`, `StatefulSet`, `DaemonSet` | [internal/dependency/index.go](internal/dependency/index.go#L57-L66) |
| Only `Secret` and `ConfigMap` are observed as change sources | [internal/controller/source_observer_controller.go](internal/controller/source_observer_controller.go#L41-L48) |
| Restart writes only `kubectl.kubernetes.io/restartedAt` | [internal/executor/restart.go](internal/executor/restart.go#L23) |
| GitOps gating is opt-in; the default provider is `None` | [api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go#L61-L73) |
| KICK-native cron windows work without any GitOps provider | [api/v1alpha1/kickpolicy_types.go](api/v1alpha1/kickpolicy_types.go#L40-L59) |
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

## Cross-cutting decisions to make first

These affect more than one plan, so decide them before writing specs.

1. **Does KICK accept workload annotations at all?** Today the answer is no —
   the whole opt-in story is `KickPolicy` selectors. Gaps 3 and 5 are much
   cheaper if we allow a narrow, KICK-owned annotation, and noticeably harder
   if we do not. See [03](ai-docs/gap-analysis/03-unreferenced-dependencies.md#option-b-workload-annotation)
2. **Does KICK ever run an admission webhook?** Gap 5 (and part of gap 2)
   depends on it. A webhook adds cert management, a failure mode that can block
   pod creation cluster-wide, and a large operational surface for a young
   project.
3. **How far does "extra kinds" go?** `DeploymentConfig` and Argo Rollouts both
   require new client dependencies and change what "current rollout" means in
   [internal/freshness](internal/freshness). See
   [04](ai-docs/gap-analysis/04-reloader-parity-gaps.md).
4. **Do we want feature parity at all?** KICK's stated positioning is narrow and
   GitOps-aware. Some of these gaps are better answered with "use Reloader" than
   with code.

## Traceability

New identifiers proposed by these plans, allocated above the current maximums
(`KICK-FEAT-020`, `KICK-E2E-056`):

| Plan | Feature IDs | E2E IDs |
|---|---|---|
| 01 | KICK-FEAT-021 … 022 | KICK-E2E-057 … 059 |
| 02 | KICK-FEAT-023 … 025 | KICK-E2E-060 … 063 |
| 03 | KICK-FEAT-026 … 027 | KICK-E2E-064 … 067 |
| 04 | KICK-FEAT-028 … 030 | KICK-E2E-068 … 071 |
| 05 | KICK-FEAT-031 … 032 | KICK-E2E-072 … 074 |

IDs are reserved on paper only. Nothing is added to
[traceability/features.yaml](traceability/features.yaml) until a plan is
accepted and a task file exists.
