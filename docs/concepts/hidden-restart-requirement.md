# Hidden restart requirement

Kubernetes does **not** restart a workload when the content of a `Secret` or
`ConfigMap` it consumes changes. The change lands in the API server, but the
running Pod keeps serving its old configuration.

![Updating a Secret or ConfigMap does nothing in Kubernetes, so the Pod keeps its old config and drifts from the intended state.](/images/the-problem.drawio.svg)

There are two reasons:

- **Environment variables** are resolved once, at Pod start. A later change to
  the source is never re-read.
- **Mounted files** are updated on the node eventually, but whether the process
  reloads them is entirely workload-specific — most never do.

The result is silent **configuration drift**: the intended state (the updated
Secret/ConfigMap) and the running state (the Pod's in-memory config) diverge,
with no error and no signal.

KICK closes this gap. It tracks each workload's dependencies and triggers a
controlled rollout when one changes — immediately by default, or gated on
[GitOps state and schedule windows](../gitops-gates/) when configured.
