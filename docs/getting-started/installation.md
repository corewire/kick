# Installation

## Prerequisites

- Docker
- `kind`
- `kubectl`
- `helm`
- `make`

## Create local cluster

```bash
make kind-create
```

KICK local defaults:

- context: `kind-kick-dev`
- kubeconfig: `.kubeconfig-kind-kick-dev`

## Build and load controller image

```bash
make kind-load
```

## Install CRDs and controller

```bash
make install
```

## Verify install

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n kick-system get pods
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev get crd | grep kick.corewire.io
```