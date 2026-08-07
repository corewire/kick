# KICK-E2E-008 imagePullSecrets ignored

## Goal
Reproduce the scenario with three explicit phases: start resources, patch Secret/ConfigMap, then inspect KICK behavior.

## 1. Apply starting resources
```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -f 00-starting-resources.yaml
```

## 2. Patch source data
```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -f 10-patch-secret.yaml
```

## 3. Check what KICK did
```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n kick-example-008 get kickrequests
```

Expected:
- No kick request should be created (imagePullSecrets-only change is ignored).

## Cleanup
```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev delete namespace kick-example-008 --ignore-not-found=true --wait=false
```
