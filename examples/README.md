# Examples

Hand-authored examples for people exploring KICK behavior.

## Timeline UI demo

- [examples/timeline-ui/GUIDE.md](examples/timeline-ui/GUIDE.md)
- [examples/timeline-ui/00-starting-resources.yaml](examples/timeline-ui/00-starting-resources.yaml)
- [examples/timeline-ui/10-patch-secret.yaml](examples/timeline-ui/10-patch-secret.yaml)

Use this when you want a small namespace that cleanly demonstrates:
- KickPolicy workload discovery;
- namespace DAG rendering (`KickPolicy -> workload -> Secret/ConfigMap`);
- timeline events after relevant dependency changes.

## Scenario examples

- [examples/scenarios/README.md](examples/scenarios/README.md)

These mirror e2e scenario fixtures and include broader behavior coverage.
