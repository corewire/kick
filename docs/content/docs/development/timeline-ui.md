# Timeline UI and Tracing

KICK now exposes a timeline API and browser UI for workload restart investigations.

> **⚠ Experimental — do not expose.** The timeline UI and API are an unauthenticated, read-only debug aid intended for local development (localhost / `kubectl port-forward`) only. They expose namespace, workload, policy, and event metadata with no authn/authz. Never expose `--timeline-bind-address` through an Ingress, LoadBalancer, or any untrusted network.

## Endpoints

- Root: `/` (redirects to `/timeline/ui`)
- UI: `/timeline/ui`
- Overview API: `/timeline/overview` (all managed workloads and their events across every namespace)
- API: `/timeline?namespace=<ns>&kind=<Deployment|StatefulSet|DaemonSet>&name=<workload>`
- Discovery API: `/timeline/discovery?namespace=<ns>[&policy=<policy-name>][&kind=<Deployment|StatefulSet|DaemonSet|All>][&name=<substring>]`
- DAG API: `/timeline/dag?namespace=<ns>`

## Cross-namespace overview

The UI opens on a compact, all-namespace overview that answers "what happened, when" at a glance:

- one swimlane per managed workload, grouped by namespace, with a color-coded state band over time;
- color-coded event markers (dependency change, restart, request, waiting/blocked, failure, k8s event) with hover details;
- a chronological event log alongside the lanes;
- a dedicated ruler row with a time picker (from/to), quick presets, and drag-to-zoom (draw a box on the ruler);
- a text filter to narrow lanes and the log by namespace or workload.

## Policy-driven discovery and filtering

The UI now auto-loads workloads discovered from `KickPolicy` selectors in the selected namespace.

- it lists discovered `Deployment`, `StatefulSet`, and `DaemonSet` workloads;
- it supports filtering by policy, workload kind, and workload name substring;
- selecting a discovered workload auto-fills the timeline target and loads that workload timeline.
- it renders a namespace DAG: `KickPolicy -> workload -> Secret/ConfigMap`.

## What the timeline shows

- relevant Secret/ConfigMap change timestamps from observation records;
- KickRequest creation and current phase snapshots;
- controller/emitted Kubernetes events for the workload and related KickRequests;
- workload `kubectl.kubernetes.io/restartedAt` updates.

## Enable timeline server

The manager serves timeline endpoints by default:

```text
--timeline-bind-address=:8090
```

Set empty value to disable:

```text
--timeline-bind-address=
```

## OTEL export (Tempo/Jaeger)

KICK emits a small, high-signal set of spans built to answer one question:
*when did the source change, and when did the workload restart?*

Each restart cycle is a **single trace** with two spans sharing one trace ID:

- `dependency.changed` — a relevant Secret/ConfigMap change with consumers, carrying a `source.changed` event at the observed-change time.
- `restart.executed` — the actual workload patch, carrying a `workload.restarted` event at the restart time.

Correlation is durable: the observer stamps the change's W3C `traceparent` onto the KickRequest (annotation `kick.corewire.io/traceparent`), and the executor resumes that trace when it restarts the workload — even across GitOps gate waits and controller restarts. Per-reconcile bookkeeping is deliberately **not** traced.

Configure OTLP export:

```text
--otel-otlp-endpoint=<collector-host:4317>
--otel-otlp-insecure=true
```

Examples:

- Tempo via OTLP collector service in-cluster.
- Jaeger collector OTLP gRPC endpoint.

When endpoint is unset, tracing remains disabled and has no exporter overhead.

## Tracing demo (Tilt)

`tilt up` deploys a self-contained tracing backend for local development:

- Jaeger all-in-one (`hack/tracing/jaeger.yaml`) in the `kick-tracing` namespace, with in-memory storage and the OTLP receiver enabled.
- The `config/dev` overlay points the manager at it via `--otel-otlp-endpoint=jaeger.kick-tracing.svc.cluster.local:4317`.

Open the Jaeger UI at [http://localhost:16686](http://localhost:16686) (port-forwarded by Tilt), select the `kick-controller` service, and trigger a restart to see source-observation, KickRequest reconciliation, and restart-execution spans.

> The demo backend has no persistence or auth. It is for local development only.
