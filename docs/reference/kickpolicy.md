# KickPolicy API Reference

Group/version: `kick.corewire.io/v1alpha1`

Kind: `KickPolicy`

## spec.discovery

- `mode` enum: `Auto`
- `workloadSelector` optional label selector

## spec.gitOps

- `provider` enum: `Auto`, `ArgoCD`, `Flux`
- `requireReconciled` default: `true`
- `schedule.source` enum: `Provider`, `None` (default `Provider`)

## spec

- `minInterval` default: `30s`

## status

- `observedGeneration`
- `matchedWorkloads`
- `blockedWorkloads`
- `conditions`