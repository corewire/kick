# Timeline UI and Tracing

KICK now exposes a timeline API and browser UI for workload restart investigations.

## Endpoints

- Root: `/` (redirects to `/timeline/ui`)
- UI: `/timeline/ui`
- API: `/timeline?namespace=<ns>&kind=<Deployment|StatefulSet|DaemonSet>&name=<workload>`
- Discovery API: `/timeline/discovery?namespace=<ns>[&policy=<policy-name>][&kind=<Deployment|StatefulSet|DaemonSet|All>][&name=<substring>]`
- DAG API: `/timeline/dag?namespace=<ns>`

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

KICK now emits OTEL spans for source observation, KickRequest reconciliation, and restart execution.

Configure OTLP export:

```text
--otel-otlp-endpoint=<collector-host:4317>
--otel-otlp-insecure=true
```

Examples:

- Tempo via OTLP collector service in-cluster.
- Jaeger collector OTLP gRPC endpoint.

When endpoint is unset, tracing remains disabled and has no exporter overhead.
