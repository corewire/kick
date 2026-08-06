# KICK operator starter

KICK detects workloads whose consumed Secrets or ConfigMaps changed after their latest rollout, then triggers a controlled restart when the configured GitOps provider allows it.

This repository is **boilerplate**, not a finished operator. It deliberately implements only stable, provider-neutral foundations:

- Kubebuilder-compatible project layout;
- `KickRequest` API skeleton;
- GitOps provider contract;
- Deployment dependency extraction;
- unit tests for dependency extraction;
- controller and Argo CD adapter boundaries;
- feature/e2e traceability scaffolding.

Unresolved Kubernetes timestamps and Argo CD ownership/window behavior remain explicit research tasks. Do not replace those tasks with assumptions.

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
make uninstall
make tilt-down
```

Dependencies are pinned as starter values and must be reviewed before the first production implementation pull request.
# kick
