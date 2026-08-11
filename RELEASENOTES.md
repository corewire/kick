# Release Notes

## Compatibility

- API versions: `kick.corewire.io/v1alpha1` — `KickPolicy`, `KickRequest`, `NotificationPolicy`
- Workload kinds: `Deployment`, `StatefulSet`, `DaemonSet`, and `argoproj.io/v1alpha1` `Rollout` (opt-in)
- Dependency sources: `Secret`, `ConfigMap`, and `SecretProviderClass` (opt-in)
- GitOps gating: optional (default `None`); adapters for Argo CD, Flux and Kargo

Kubernetes and Argo CD compatibility must be validated per release by CI/e2e evidence.

## Optional integrations

Both are disabled by default and are additionally skipped when the corresponding
CRD is absent, so enabling them on a cluster that lacks the dependency is safe.

- `--enable-argo-rollouts` (`integrations.argoRollouts.enabled`) — restarts Argo
  Rollouts through `spec.restartAt` rather than the pod template, so a
  configuration change does not re-run the canary or blue-green strategy.
- `--enable-csi-integration` (`integrations.secretsStoreCSI.enabled`) — observes
  `SecretProviderClassPodStatus` and restarts once a rotation has reached every
  pod of the class.

`integrations.kargo.enabled` grants the RBAC needed for `provider: Kargo`; Kargo
is never auto-detected and must be selected explicitly.

## Limitations

- when a GitOps provider is configured, ambiguous or missing ownership blocks automatic restart;
- historical dependency changes before KICK observation start are not reconstructed;
- dependencies cannot be declared by name — only references proven from the pod spec trigger restarts;
- KICK ships no admission webhook and therefore never holds pods `Pending`;
- OpenShift `DeploymentConfig` is not supported;
- `NotificationPolicy` delivery is best-effort: the queue is bounded and drops the
  oldest event under sustained backpressure, and a failed delivery never fails a restart;
- behavior remains alpha and may change before beta.

## Artifacts

- container image (signed)
- Helm chart package
- CRD bundle
- SBOM
- checksums