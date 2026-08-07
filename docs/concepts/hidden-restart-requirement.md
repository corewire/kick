# Hidden Restart Requirement

![Updating a Secret or ConfigMap does nothing in Kubernetes, so the Pod keeps its old config and drifts from the intended state.](../images/the-problem.drawio.svg)

Kubernetes does not restart Deployments when Secret or ConfigMap content changes.

- Environment variables are resolved at Pod start.
- Mounted files update on node-side sync, but process reload is workload-specific.

Result: configuration drift can exist between intended and running behavior.

KICK closes this gap by tracking dependency changes and triggering controlled rollouts when policy permits — immediately by default, or gated on GitOps state and windows when configured.