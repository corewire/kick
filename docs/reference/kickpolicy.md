# 
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

## spec.schedule

`spec.schedule` is the KICK-native time gate: pure scheduling, evaluated without
any GitOps provider. Omit it to allow restarts at any time.

- `windows[]` KICK-native restart windows
  - `type` enum: `Allow`, `Deny` (required)
  - `cron` 5-field cron expression marking each window start (required)
  - `duration` how long the window stays open from each start, e.g. `1h` (required)
  - `timeZone` IANA zone used to evaluate the cron expression (default UTC)

## spec.gitOps

`spec.gitOps` is optional. When omitted, `provider` defaults to `None` and KICK
restarts without consulting a GitOps tool (gated only by any native windows).

- `provider` enum: `None`, `Auto`, `ArgoCD`, `Flux`, `Kargo` (default `None`)
- `requireReconciled` default: `true` (applies only to a real provider)

`Kargo` is never auto-detected and must be selected explicitly: Kargo does not
write to workloads, Argo CD does, so a Kargo-managed workload is
indistinguishable from a plain Argo CD one. With `Kargo`, KICK resolves the
authorised `Stage` from the owning Application's
`kargo.akuity.io/authorized-stage` annotation, blocks while a Promotion for that
Stage is in flight, and then delegates to the Argo CD gate. More than one
authorised stage is treated as ambiguous ownership and blocks.

See [Running without GitOps](../guides/without-gitops/) for the `None` case.

## spec.restart

- `minInterval` default: `30s`

## spec

- `suspend` pauses the policy without deleting it (default `false`)
- `dryRun` evaluates every gate and the freshness check but never patches a
  workload (default `false`). The `KickRequest` ends in the terminal `DryRun`
  phase with the decision recorded in its conditions, so you can see exactly
  what would have happened.

## status

- `observedGeneration`
- `matchedWorkloads`
- `blockedWorkloads`
- `conditions`
