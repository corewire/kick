# Metrics Reference

Prometheus metrics are registered in controller-runtime metrics registry.

## `kick_requests_total`

Counter by labels:

- `provider`
- `result`

Meaning: completed KickRequest outcomes.

## `kick_restarts_total`

Counter by labels:

- `provider`
- `result`

Meaning: restart execution attempts and outcomes.

## `kick_controller_errors_total`

Counter by labels:

- `controller`
- `reason`

Meaning: controller reconciliation errors.