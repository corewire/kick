---
title: Quickstart
weight: 20
aliases:
  - /docs/getting-started/quickstart/
description: Watch KICK restart a Deployment when its Secret changes — no GitOps tool required.
---

This validates the end-to-end KICK flow on a Kind cluster:

1. install KICK;
2. apply a Deployment that reads a Secret, plus a `KickPolicy`;
3. change the Secret;
4. watch KICK restart the Deployment.

> The local dev commands use context `kind-kick-dev` and kubeconfig
> `.kubeconfig-kind-kick-dev`. Add `--context` / `--kubeconfig` to match your setup.

## 1) Install KICK

```bash
make kind-create
make kind-load
make install
```

Already have a cluster? Install the chart instead — see
[Installation](../installation/).

## 2) Apply a workload and a policy

```bash
kubectl -n shop apply -f - <<'EOF'
apiVersion: v1
kind: Namespace
metadata: { name: shop }
---
apiVersion: v1
kind: Secret
metadata: { name: web-secret, namespace: shop }
type: Opaque
stringData: { API_TOKEN: alpha }
---
apiVersion: apps/v1
kind: Deployment
metadata: { name: web, namespace: shop, labels: { app: web } }
spec:
  replicas: 1
  selector: { matchLabels: { app: web } }
  template:
    metadata: { labels: { app: web } }
    spec:
      containers:
      - name: app
        image: nginx
        envFrom:
        - secretRef: { name: web-secret }
---
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata: { name: web, namespace: shop }
spec:
  discovery:
    workloadSelector:
      matchLabels: { app: web }
EOF
```

## 3) Change the Secret

```bash
kubectl -n shop patch secret web-secret --type merge \
  -p '{"stringData":{"API_TOKEN":"bravo"}}'
```

## 4) Observe the request and rollout

```bash
kubectl -n shop get kickrequests -w
kubectl -n shop rollout status deploy/web --timeout=5m
```

Success signal:

- a `KickRequest` appears and reaches `Succeeded` or `NoLongerRequired`;
- the Deployment starts a fresh rollout.

## Optional: gate on Argo CD

To make restarts respect Argo CD ownership and sync windows, install Argo CD and
set `spec.gitOps.provider: Auto` on the policy:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl -n argocd rollout status deploy/argocd-server --timeout=5m
```

See the [Argo CD guide](../guides/argocd/) for a full Argo CD-tracked workload
and policy.
