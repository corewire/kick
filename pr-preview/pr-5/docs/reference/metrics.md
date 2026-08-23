# 
# Metrics Reference

Prometheus metrics are registered in controller-runtime metrics registry.

## `kick_requests_total`

Counter by labels:

- `provider` — GitOps provider name, or `unknown` when none applies
- `result` — `succeeded`, `no_longer_required`, `failed`, `dry_run`

Meaning: completed KickRequest outcomes.

`dry_run` is recorded when a policy sets `spec.dryRun: true`; the request reaches
a terminal decision without a restart being performed.

## `kick_restarts_total`

Counter by labels:

- `provider` — GitOps provider name, or `unknown` when none applies
- `result` — `started`, `succeeded`, `failed`

Meaning: restart execution attempts and outcomes.

## `kick_controller_errors_total`

Counter by labels:

- `controller`
- `reason` — `Unknown` when no reason is supplied

Meaning: controller reconciliation errors.

## `kick_notification_deliveries_total`

Counter by labels:

- `namespace` — namespace of the NotificationPolicy
- `policy` — NotificationPolicy name
- `outcome` — `success` or `failure`

Meaning: NotificationPolicy webhook delivery attempts. A failed delivery never
fails a restart.

## `kick_notification_dropped_total`

Counter, no labels.

Meaning: notification events discarded because the delivery queue was full. A
non-zero value means the webhook endpoint is not keeping up.

