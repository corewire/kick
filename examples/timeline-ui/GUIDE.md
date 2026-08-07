# Timeline UI demo example

This example is a small namespace that exists only to demo timeline discovery, DAG rendering, and per-workload timeline events in the UI.

Files:
- `00-starting-resources.yaml`: Namespace, KickPolicy, Secret, ConfigMap, and 3 workload kinds.
- `10-patch-secret.yaml`: A relevant Secret content change.

## Apply

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -f examples/timeline-ui/00-starting-resources.yaml
```

Then open:
- http://localhost:8090/timeline/ui

In the UI:
1. Set namespace to `timeline-demo`.
2. Click `Refresh Discovery`.
3. Confirm DAG nodes/edges appear for policy -> workload -> dependencies.
4. Click a discovered workload card to load its timeline.

## Trigger relevant change

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -f examples/timeline-ui/10-patch-secret.yaml
```

Refresh the selected workload timeline. You should see dependency-relevant-change entries and, when applicable, kickrequest and rollout entries.

## Cleanup

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev delete namespace timeline-demo
```
