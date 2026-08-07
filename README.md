# KICK operator

KICK detects workloads whose consumed Secrets or ConfigMaps changed after their latest rollout, then triggers a controlled restart when the configured GitOps provider allows it.

Supported workload kinds: `Deployment`, `StatefulSet`, `DaemonSet`.

This repository currently contains a **bootstrap baseline**, not a finished operator. It implements only stable, provider-neutral foundations:

- Kubebuilder-compatible project layout;
- `KickRequest` API skeleton;
- GitOps provider contract;
- Deployment dependency extraction;
- unit tests for dependency extraction;
- controller and Argo CD adapter boundaries;
- feature/e2e traceability scaffolding.

Unresolved Kubernetes timestamps and Argo CD ownership/window behavior remain explicit research tasks. Do not replace those tasks with assumptions.

## Specs and references

- authoritative specifications: `ai-docs/kick-operator-specs/kick-specs/`
- copied starter reference: `ai-docs/kick-operator-starter/`

## First checks

```bash
make fmt
make test
make feature-coverage
```

## Local dev with Tilt (kind-kick-dev only)

```bash
make kind-create
make tilt-up
```

Rules enforced by this repository:

- cluster context is `kind-kick-dev`;
- kubeconfig path is `.kubeconfig-kind-kick-dev` in repo root;
- commands always pass explicit `--kubeconfig` and `--context`.

Additional helpers:

```bash
make kind-load
make install
make test-e2e
make uninstall
make tilt-down
```

Timeline and tracing:

```text
--timeline-bind-address=:8090
--otel-otlp-endpoint=<collector-host:4317>
--otel-otlp-insecure=true
```

Timeline UI path: `/timeline/ui`

## Security note

The controller ServiceAccount requires read access to Secrets and ConfigMaps in managed namespaces to evaluate dependency freshness. Treat this ServiceAccount as sensitive and scope RBAC and namespace access accordingly.

Dependencies are pinned as starter values and must be reviewed before the first production implementation pull request.
