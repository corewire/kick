# Release Notes

## Compatibility

- API version: `kick.corewire.io/v1alpha1`
- Workload kinds: `Deployment`, `StatefulSet`, `DaemonSet`
- GitOps gating: optional (default `None`); Argo CD adapter implemented first

Kubernetes and Argo CD compatibility must be validated per release by CI/e2e evidence.

## Limitations

- when a GitOps provider is configured, ambiguous or missing ownership blocks automatic restart;
- historical dependency changes before KICK observation start are not reconstructed;
- behavior remains alpha and may change before beta.

## Artifacts

- container image (signed)
- Helm chart package
- CRD bundle
- SBOM
- checksums