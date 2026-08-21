# End-to-end tests

KICK's end-to-end tests run Chainsaw scenarios against a kind cluster. Every
scenario directory under `test/e2e/scenarios` maps to one stable scenario ID and
carries a `trace.yaml` linking it to the features it proves.

## Suites

| Target | Covers | Installs |
|---|---|---|
| `make test-e2e-core` | restart, policy and observation behaviour | KICK |
| `make test-e2e-argocd` | Argo CD ownership, sync windows and sync state | Gitea, Argo CD |
| `make test-e2e-recovery` | crash and restart recovery | KICK |
| `make test-e2e-rollouts` | Argo Rollouts restarts | Argo Rollouts |
| `make test-e2e-csi` | Secrets Store CSI rotation | CSI driver and provider |
| `make test-e2e-kargo` | Kargo promotion gating | cert-manager, Kargo |

Each target installs its own prerequisites and then redeploys the manager from
the `config/e2e` overlay. The manager probes the optional integration CRDs once
at startup, so it is always restarted after a CRD is installed — otherwise the
integration would stay silently inactive.

## Working on a single scenario

```bash
make e2e-rollouts-setup            # prerequisites for the suite
make test-e2e-scenario E2E=060     # one scenario, integration timeout budget
make test-e2e-render               # render every scenario without a cluster
```

## GitOps fixtures

Scenarios that need a real GitOps source push manifests into the in-cluster
Gitea, one repository per scenario, so parallel scenarios never share state:

```bash
test/e2e/setup/gitea/seed.sh        e2e-037 manifests ./manifests
test/e2e/setup/gitea/commit-file.sh e2e-037 manifests/app.yaml ./updated/app.yaml
```

Argo CD sync windows also block Argo CD's own sync, so a scenario that closes a
window must deliver the dependency change directly instead of through git.

