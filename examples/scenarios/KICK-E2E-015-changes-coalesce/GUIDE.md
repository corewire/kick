# KICK-E2E-015 changes coalesce

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
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n kick-example-015 get kickrequest app-015 -o yaml
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n kick-example-015 get kickrequests
```

Expected:
- A KickRequest named after the Deployment should exist and move to waiting/owner resolution states in this baseline.

## Cleanup
```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev delete namespace kick-example-015 --ignore-not-found=true --wait=false
```
