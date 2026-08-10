# KickPolicy API Reference

Group/version: `kick.corewire.io/v1alpha1`

Kind: `KickPolicy`

## spec.discovery

Two optional label selectors scope the policy. An empty or omitted selector
matches everything on its axis.

- `workloadSelector` — which workloads the policy manages (the actors that may be
  restarted). Omit to match all supported workloads in the namespace.
- `dependencySelector` — which consumed `Secret`/`ConfigMap` changes count as a
  trigger. Omit to treat every discovered dependency as a trigger.

A workload restarts when it **consumes a changed dependency**, the workload is in
`workloadSelector` scope, and the changed dependency is in `dependencySelector`
scope. `dependencySelector` also scopes freshness: out-of-scope dependencies are
ignored entirely.

## spec.gitOps

`spec.gitOps` is optional. When omitted, `provider` defaults to `None` and KICK
restarts without consulting a GitOps tool (gated only by any native windows).

- `provider` enum: `None`, `Auto`, `ArgoCD`, `Flux` (default `None`)
- `requireReconciled` default: `true` (applies only to a real provider)
- `schedule.source` enum: `Provider`, `None` (default `Provider`)
- `schedule.windows[]` KICK-native allow/deny windows (evaluated without a provider)

## spec

- `minInterval` default: `30s`

## status

- `observedGeneration`
- `matchedWorkloads`
- `blockedWorkloads`
- `conditions`