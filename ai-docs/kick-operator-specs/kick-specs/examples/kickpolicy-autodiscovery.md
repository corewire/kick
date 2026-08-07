# KickPolicy autodiscovery example

## Policy

```yaml
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: production
  namespace: payments
spec:
  discovery:
    workloadSelector:
      matchLabels:
        kick.corewire.io/enabled: "true"
  gitOps:
    provider: ArgoCD
    requireReconciled: true
    schedule:
      source: Provider
  minInterval: 30s
```

This policy enables KICK only for workloads in namespace payments that match label kick.corewire.io/enabled=true.

## Managed workload

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payments-api
  namespace: payments
  labels:
    kick.corewire.io/enabled: "true"
  annotations:
    argocd.argoproj.io/tracking-id: payments-app:apps/Deployment:payments/payments-api
spec:
  template:
    spec:
      containers:
        - name: api
          image: registry.example.com/payments-api:1.4.0
          envFrom:
            - secretRef:
                name: payments-database
            - configMapRef:
                name: payments-config
      volumes:
        - name: tls
          secret:
            secretName: payments-tls
      imagePullSecrets:
        - name: registry-credentials
```

## Discovery result

```text
Managed:
  Deployment/payments/payments-api

Dependencies:
  Secret/payments/payments-database
  ConfigMap/payments/payments-config
  Secret/payments/payments-tls

Excluded:
  Secret/payments/registry-credentials (imagePullSecrets)
```

## Scope example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: billing-api
  namespace: billing
  labels:
    kick.corewire.io/enabled: "true"
```

Deployment billing-api is unmanaged because KickPolicy production is namespaced to payments and cannot select workloads in billing.

## Conflict example

If two KickPolicy resources in namespace payments both match payments-api, the workload is blocked with reason ConflictingPolicies and KICK does not restart it.
