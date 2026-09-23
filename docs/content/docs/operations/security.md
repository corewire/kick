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

## Observation records

KICK must read Secret and ConfigMap content to tell a content change from a
metadata-only write; Kubernetes exposes no server-side content digest
(`resourceVersion` and `managedFields` change on label writes too). The durable
record KICK keeps per source is an HMAC-SHA256 of the content under a
per-installation key, never a plain digest, so a reader of the record cannot
brute-force low-entropy values from it. The key lives in the
`kick-fingerprint-key` Secret in the manager namespace and is generated on
first start; deleting it rotates the key and re-anchors every record on its
next observation without restarting anything
([internal/observation/hash.go](https://github.com/corewire/kick/blob/main/internal/observation/hash.go),
[internal/observation/key.go](https://github.com/corewire/kick/blob/main/internal/observation/key.go)).
Helm release Secrets (`type: helm.sh/release.v1`) are excluded from the watch
at the API server.

## Timeline UI

The timeline server is disabled by default. Enable it with `--timeline-bind-address` only on trusted networks, for example via `kubectl port-forward`. It is unauthenticated and exposes namespace names, workload names, and restart state.

## Tracing

OTLP trace export defaults to TLS. Use `--otel-otlp-insecure` only when the collector is reached through a trusted, encrypted path such as a service mesh.