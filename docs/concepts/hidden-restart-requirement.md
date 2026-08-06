# Hidden Restart Requirement

Kubernetes does not restart Deployments when Secret or ConfigMap content changes.

- Environment variables are resolved at Pod start.
- Mounted files update on node-side sync, but process reload is workload-specific.

Result: configuration drift can exist between intended and running behavior.

KICK closes this gap by tracking dependency changes and triggering controlled rollouts when policy and GitOps gates permit.