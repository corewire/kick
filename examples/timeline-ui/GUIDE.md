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

In the UI (opens on the cross-namespace overview):
1. Find the `timeline-demo` namespace group and its workload lanes.
2. Hover event markers for details; read the event log for "what happened when".
3. Drag a box on the time ruler to zoom, or set the from/to picker to focus a window.
4. Type `timeline-demo` in the filter to isolate the demo workloads.

## Trigger relevant change

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev apply -f examples/timeline-ui/10-patch-secret.yaml
```

Click `Refresh` (or enable `auto`). You should see a dependency-relevant-change marker and, when applicable, kickrequest and rollout events on the `timeline-demo` lanes.

## Cleanup

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev delete namespace timeline-demo
```
