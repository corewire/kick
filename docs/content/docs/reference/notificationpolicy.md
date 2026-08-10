# NotificationPolicy API Reference

Group/version: `kick.corewire.io/v1alpha1`

Kind: `NotificationPolicy` (namespaced)

A `NotificationPolicy` delivers an HTTP webhook when a `KickRequest` in the same
namespace reaches a selected phase. Delivery is best-effort: a failed webhook
never fails a restart.

## spec

- `suspend` pauses delivery without deleting the policy (default `false`)
- `phases[]` which `KickRequest` phases to deliver. Defaults to the terminal
  phases `Succeeded`, `Failed`, `NoLongerRequired`, `DryRun`.
- `workloadSelector` optional label selector on the `KickRequest` labels. Omit
  to match every request in the namespace.

## spec.webhook

- `url` required, must match `^https?://`
- `method` enum `POST`, `PUT` (default `POST`)
- `timeoutSeconds` default `10`, min `1`, max `120`
- `headers[]` static headers
  - `name` required
  - `value` literal value
  - `valueFrom.name` / `valueFrom.key` read the value from a `Secret` in the
    same namespace
- `auth.bearerToken.name` / `.key` — `Authorization: Bearer <value>`
- `auth.basic.username` / `auth.basic.password` — each a Secret key reference
- `tls.caBundle.name` / `.key` — PEM bundle used to verify the server
- `tls.clientCertificate.name` / `.key` — a `kubernetes.io/tls` Secret used for
  mutual TLS

All credentials are Secret references. Literal credentials cannot be set inline.

## Payload

The request body is JSON with a fixed field set:

```json
{
  "namespace": "team-a",
  "requestName": "api-0f3c",
  "phase": "Succeeded",
  "reason": "RestartCompleted",
  "message": "rollout completed",
  "targetKind": "Deployment",
  "targetName": "api",
  "gitOpsProvider": "argocd",
  "occurredAt": "2026-03-01T12:00:00Z"
}
```

The payload never contains `Secret` or `ConfigMap` data, key names, or content
digests.

## Delivery semantics

- Events are queued in memory. The queue is bounded; when it is full the oldest
  event is dropped and `kick_notification_dropped_total` is incremented.
- Delivery is retried up to three times with exponential backoff. `4xx`
  responses other than `429` are not retried.
- Delivery runs only on the elected leader, so a highly-available deployment
  does not duplicate webhooks.
- TLS is negotiated with a minimum version of TLS 1.2.

## status

- `observedGeneration`
- `lastDeliveryTime`
- `lastError`
- `delivered`, `failed` counters
- `conditions`

## Metrics

- `kick_notification_deliveries_total{namespace,policy,outcome}`
- `kick_notification_dropped_total`
