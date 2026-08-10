---
title: Installation
weight: 10
aliases:
  - /docs/getting-started/
  - /docs/getting-started/installation/
description: Install the KICK operator with Helm, or from source for local development.
llmsDescription: |
  Installation guide for the KICK operator. Prerequisites: Kubernetes 1.28+,
  Helm 3.12+. Install via the Helm chart (oci://ghcr.io/corewire/charts/kick)
  into namespace kick-system. Key values: argocd.enabled, image.repository/tag.
  Helm installs CRDs on first install but does not upgrade them — re-apply the
  CRDs with kubectl on upgrade. A from-source path (kind + make) is provided for
  local development.
---

KICK ships as a single controller plus two cluster-scoped CRDs
(`KickPolicy`, `KickRequest`). The recommended install is the Helm chart.

## Prerequisites

- Kubernetes 1.28+
- Helm 3.12+
- (optional) Argo CD, if you want to gate restarts behind GitOps

## Helm install

```bash
helm install kick oci://ghcr.io/corewire/charts/kick \
  --namespace kick-system \
  --create-namespace
```

That installs the CRDs, RBAC, and the controller `Deployment`.

### Common values

Override with `--set key=value` or a `-f values.yaml` file:

| Value | Default | Description |
|-------|---------|-------------|
| `image.repository` | `ghcr.io/corewire/kick` | Controller image. |
| `image.tag` | chart `appVersion` | Controller image tag. |
| `argocd.enabled` | `true` | Grant RBAC to read Argo CD `Application`/`AppProject` for GitOps gating. |
| `argocd.applicationNamespaces` | `["*"]` | Namespaces KICK may read Argo CD objects from. |
| `leaderElection.enabled` | `true` | Enable leader election for HA. |
| `replicaCount` | `1` | Controller replicas. |

If you do not use Argo CD, disable its RBAC:

```bash
helm install kick oci://ghcr.io/corewire/charts/kick \
  --namespace kick-system --create-namespace \
  --set argocd.enabled=false
```

### Upgrading CRDs

Helm installs the CRDs on first install but **does not** upgrade them on
`helm upgrade`. When a release changes the CRDs, re-apply them explicitly:

```bash
kubectl apply -f https://raw.githubusercontent.com/corewire/kick/main/config/crd/bases/kick.corewire.io_kickpolicies.yaml
kubectl apply -f https://raw.githubusercontent.com/corewire/kick/main/config/crd/bases/kick.corewire.io_kickrequests.yaml
```

## From source (local development)

For a throwaway Kind cluster with the controller built from your working tree:

```bash
make kind-create   # create the kind-kick-dev cluster
make kind-load     # build and load the controller image
make install       # install CRDs + controller
```

Local defaults: context `kind-kick-dev`, kubeconfig `.kubeconfig-kind-kick-dev`.
You can also install the chart directly from the repo checkout:

```bash
helm install kick charts/kick --namespace kick-system --create-namespace
```

## Verify

```bash
kubectl -n kick-system get pods
kubectl get crd | grep kick.corewire.io
```

The controller Pod should be `Running` and both CRDs
(`kickpolicies`, `kickrequests`) present.

## Next steps

- [Quickstart](../quickstart/) — see KICK restart a Deployment when a Secret changes.
- [Concepts](../concepts/) — how discovery, freshness, and GitOps gating work.
