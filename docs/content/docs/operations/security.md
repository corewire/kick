# Security

KICK requires read access to Secrets and ConfigMaps in managed namespaces to evaluate dependency freshness.

Implications:

- treat controller ServiceAccount as sensitive;
- restrict namespace scope where possible;
- avoid exposing logs broadly.

KICK safety constraints:

- no privileged containers;
- no CRI socket access;
- no Secret value logging;
- container runs with a read-only root filesystem and the runtime default seccomp profile.

## Timeline UI

The timeline server is disabled by default. Enable it with `--timeline-bind-address` only on trusted networks, for example via `kubectl port-forward`. It is unauthenticated and exposes namespace names, workload names, and restart state.

## Tracing

OTLP trace export defaults to TLS. Use `--otel-otlp-insecure` only when the collector is reached through a trusted, encrypted path such as a service mesh.