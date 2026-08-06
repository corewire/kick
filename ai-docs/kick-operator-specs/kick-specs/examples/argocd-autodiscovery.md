# Argo CD autodiscovery example

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payments-api
  namespace: payments
  annotations:
    argocd.argoproj.io/tracking-id: payments-app:apps/Deployment:payments/payments-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payments-api
  template:
    metadata:
      labels:
        app: payments-api
    spec:
      containers:
        - name: api
          image: registry.example.com/payments-api:1.4.0
          envFrom:
            - secretRef:
                name: payments-database
            - configMapRef:
                name: payments-config
          volumeMounts:
            - name: tls
              mountPath: /etc/payments/tls
              readOnly: true
      volumes:
        - name: tls
          secret:
            secretName: payments-tls
      imagePullSecrets:
        - name: registry-credentials
```

## Discovered graph

```text
Secret/payments/payments-database ─┐
                                   ├─> Deployment/payments/payments-api
ConfigMap/payments/payments-config ┤
                                   │
Secret/payments/payments-tls ──────┘
```

`Secret/payments/registry-credentials` is excluded because it is used only by `imagePullSecrets`.

## Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: payments-app
  namespace: application-team-a
spec:
  project: production
  destination:
    namespace: payments
    name: production-cluster
status:
  sync:
    status: Synced
  health:
    status: Healthy
```

## AppProject

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
  namespace: argocd
spec:
  syncWindows:
    - kind: allow
      schedule: "0 2 * * *"
      duration: 1h
      timeZone: Europe/Berlin
      applications:
        - payments-app
```

## Example decision

```text
latest dependency change: 2026-08-06T09:20:00Z
current ReplicaSet created: 2026-08-06T08:00:00Z
Application: Synced and idle
window: open
result: kick required and permitted
```
