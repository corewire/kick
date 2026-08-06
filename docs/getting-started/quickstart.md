# Quickstart (Kind + Argo CD)

This quickstart validates the end-to-end KICK flow:

1. create cluster and install KICK;
2. apply a Deployment with Secret/ConfigMap dependencies;
3. create a KickPolicy;
4. update a dependency;
5. observe KickRequest lifecycle.

## 1) Cluster and KICK

```bash
make kind-create
make kind-load
make install
```

## 2) Install Argo CD

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev create namespace argocd
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n argocd rollout status deploy/argocd-server --timeout=5m
```

## 3) Apply example workload and policy

Use examples from [examples/argocd-autodiscovery.md](../../examples/argocd-autodiscovery.md).

Apply:

- Secret and ConfigMap used by the Deployment;
- Deployment with Argo CD tracking annotation;
- KickPolicy selecting the workload.

## 4) Trigger dependency change

Patch a referenced Secret or ConfigMap data field.

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n payments patch secret payments-database --type merge -p '{"stringData":{"password":"rotated"}}'
```

## 5) Observe request and rollout

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n payments get kickrequests -w
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n payments describe kickrequest <name>
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n payments rollout status deploy/payments-api --timeout=5m
```

Success signal:

- KickRequest reaches `Succeeded` or `NoLongerRequired`.

Failure signal:

- KickRequest reaches `Failed` or remains blocked in a waiting phase.