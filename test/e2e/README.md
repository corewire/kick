# KICK e2e suite

Chainsaw scenario directories under `test/e2e/scenarios`, one directory per
stable scenario ID. Every directory carries a `trace.yaml` that maps the
scenario to the features it covers.

## Suites

| Target | Scenarios | Cluster prerequisites |
|---|---|---|
| `make test-e2e-core` | everything that needs no integration | KICK only |
| `make test-e2e-argocd` | 024-042 | Gitea, Argo CD |
| `make test-e2e-recovery` | 048-051 | KICK only |
| `make test-e2e-rollouts` | 060-063 | Argo Rollouts |
| `make test-e2e-csi` | 064-067 | Secrets Store CSI driver, OpenBao + its CSI provider |
| `make test-e2e-kargo` | 068-071 | cert-manager, Kargo |
| `make test-e2e` | all of the above | all of the above |

Each suite target installs its own prerequisites first, then redeploys the
manager from the `config/e2e` overlay. Optional integrations are probed once at
manager startup, so the manager is always restarted after a CRD is installed.

Two other targets help while writing scenarios:

- `make test-e2e-scenario E2E=060` runs a single scenario by ID fragment. It
  does not install suite prerequisites, so run the matching setup target first
  (for example `make e2e-rollouts-setup`).
- `make test-e2e-render` renders every scenario without a cluster.

## Timeout budgets

Core scenarios use `test/e2e/chainsaw-configuration.yaml`. Integration
scenarios use `test/e2e/chainsaw-configuration-integration.yaml`, which raises
the assert and exec budgets because they wait for real Argo CD syncs, Argo
Rollouts strategies and secret rotation intervals.

## Fixture ordering

A scenario that asserts on a restart must produce exactly one dependency
change. Creating a Secret and its workload together does not guarantee that:
depending on which one KICK sees first, the initial observation may already
enqueue a request, and under load its restart can even land after the
scenario's own rotation.

Scenarios therefore split their fixtures:

- `resources.yaml` creates the namespace, the `KickPolicy` and the dependency.
- The next step waits for the observation `Lease` KICK writes for that
  dependency (annotations `kick.corewire.io/kind` and
  `kick.corewire.io/objectName`).
- `workload.yaml` creates the workload only afterwards.

The baseline is then established while no consumer exists, so the rotation in
`updated/` is the only change that can ever produce a `KickRequest`.

A scenario that has to observe a workload *while* it is rolling out rolls a
second revision first (`updated/workload.yaml`) and waits for the strategy's
pause step. The pause is a fixed, deterministic window; nothing in the test
depends on how long a container takes to start.

## Git fixtures

Integration scenarios that need a GitOps source push manifests into the
in-cluster Gitea (`kick-e2e-git` namespace) with one repository per scenario:

```
test/e2e/setup/gitea/seed.sh        e2e-037 manifests ./manifests
test/e2e/setup/gitea/commit-file.sh e2e-037 manifests/app.yaml ./updated/app.yaml
```

Repository URLs follow
`http://gitea.kick-e2e-git.svc.cluster.local:3000/kick-e2e/e2e-0NN.git`.

Seeding only replaces a path on `main`. A scenario whose GitOps tool promotes to
another branch therefore drops its repository first with
`test/e2e/setup/gitea/reset-repo.sh`, otherwise the promoted branch survives
from the previous run and the first sync is served from stale content.

## Kargo scenarios

The Kargo scenarios (068-071) drive a real promotion: a Warehouse subscribes to
`main`, the Stage renders `manifests/` onto the `stage/prod` branch and the
`argocd-update` step syncs the Argo CD Application. Three things make that
reproducible:

- The Application has no `syncPolicy.automated`. Kargo owns the sync, so the
  promotion is what drives it and what waits for the result.
- Freight discovery is forced with
  `kubectl annotate warehouse repo kargo.akuity.io/refresh=<epoch> --overwrite`
  instead of waiting out the Warehouse polling interval.
- A scenario that must observe the gate *while* a promotion runs promotes an
  extra workload in `argocd.argoproj.io/sync-wave: "1"` that takes minutes to
  become ready. The rotated Secret is in the default wave, so it is applied
  while the sync - and with it the promotion - is still running. Nothing else
  in that directory carries an annotation: the API server bumps a Deployment's
  `metadata.generation` on annotation changes as well, which would make the
  "was it restarted?" assertion meaningless.

Cleanup deletes Promotions before the Stage they belong to. Kargo indexes
running Promotions by Argo CD Application and resolves the Stage while doing so;
a Promotion that outlives its Stage leaves a nil entry in that index and no
Promotion in the cluster is reconciled again until the Kargo controller is
restarted.
