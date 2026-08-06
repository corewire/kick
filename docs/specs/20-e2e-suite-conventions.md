# End-to-end suite conventions

## Purpose

This specification defines how KICK end-to-end tests are structured, executed, diagnosed, and connected to feature traceability.

The design deliberately combines two proven approaches:

- DROP-style small, behavior-oriented Chainsaw scenario directories with explicit success and failure scenarios;
- provider-keycloak-style lifecycle verification that proves a resource is stable, survives controller restart, and completes cleanup rather than merely reaching a transient Ready state.

## Mandatory test stack

- Kind provides the Kubernetes cluster.
- A locally built KICK image is loaded into Kind.
- KICK is installed through the production Helm chart.
- Chainsaw executes declarative scenarios.
- Real Argo CD is installed for Argo CD adapter tests.
- Go test or a small shell harness MAY orchestrate suite-level setup, but scenario assertions MUST remain visible in scenario files.

The suite MUST run through one developer command:

```text
make test-e2e
```

Useful narrower commands MUST exist:

```text
make test-e2e-core
make test-e2e-argocd
make test-e2e-recovery
make test-e2e-scenario E2E=KICK-E2E-032
make test-e2e-render
```

`test-e2e-render` validates and renders every Chainsaw scenario without creating a cluster. It catches malformed tests and missing referenced files quickly.

## Directory layout

Every scenario has one isolated directory:

```text
test/e2e/
  README.md
  chainsaw-configuration.yaml
  setup/
    kind-config.yaml
    argocd/
    shared-fixtures/
  KICK-E2E-001-secret-envfrom/
    trace.yaml
    chainsaw-test.yaml
    00-namespace.yaml
    01-application.yaml
    02-secret.yaml
    03-deployment.yaml
    10-update-secret.yaml
    20-assert-request.yaml
    30-assert-rollout.yaml
  KICK-E2E-028-owner-missing/
    trace.yaml
    chainsaw-test.yaml
    ...
```

Rules:

- Directory names MUST start with the stable scenario ID.
- One directory tests one externally meaningful behavior.
- Success and failure behavior MUST be separate scenarios unless they are inseparable steps of one lifecycle.
- Files use numeric prefixes in execution or reading order.
- Shared setup is limited to expensive infrastructure. Behavior-specific resources stay inside the scenario.
- Scenario directories MUST be independently runnable after suite setup.

## Required scenario metadata

Every scenario contains `trace.yaml`:

```yaml
scenarioID: KICK-E2E-032
name: open-window-synced-application-allows-kick
features:
  - KICK-FEAT-010
  - KICK-FEAT-011
provider: argocd
class: behavior
required: true
```

Allowed classes:

- `smoke`
- `behavior`
- `failure`
- `race`
- `recovery`
- `security`
- `compatibility`

The coverage checker MUST compare this metadata with `traceability/e2e-scenarios.yaml` and fail on disagreement.

## Scenario anatomy

A scenario SHOULD follow this lifecycle:

1. **Arrange**: create namespace, GitOps owner, dependency, and workload.
2. **Baseline**: prove the workload is initially stable and no KickRequest exists.
3. **Act**: change one relevant input or GitOps condition.
4. **Intermediate assertion**: verify the expected pending or blocked state.
5. **Release gate**: open a window, finish sync, or recover the controller when applicable.
6. **Final assertion**: verify the expected rollout or no-op.
7. **Stability assertion**: wait through at least one additional reconciliation interval and prove no duplicate rollout or status churn occurs.
8. **Cleanup assertion**: delete scenario resources and prove owned or temporary resources disappear.

A test MUST NOT pass merely because a workload briefly reports Ready.

## Stable-state assertions

For every scenario that causes a kick, assert all applicable outcomes:

- exactly one new ReplicaSet is created;
- the Deployment rollout completes;
- the current ReplicaSet is newer than the relevant observed dependency change;
- the KickRequest reaches its documented terminal state;
- a second reconciliation does not create another ReplicaSet;
- the Argo CD Application returns to or remains `Synced` as expected;
- no controller error or warning event indicates an unhandled failure.

For no-op and blocked scenarios, assert:

- the ReplicaSet count and current ReplicaSet identity remain unchanged;
- the request reason and conditions match the expected gate decision;
- the controller remains stable over repeated reconciliations.

## Lifecycle and recovery tests

The suite MUST test more than create-and-ready behavior. Required lifecycle patterns include:

### Controller restart while pending

1. Create a blocked request.
2. Record the request and current ReplicaSet identity.
3. restart the KICK controller Deployment.
4. wait for leader election and cache synchronization.
5. prove the pending request is recovered.
6. release the gate.
7. prove exactly one rollout occurs.

### Controller restart after action

1. Trigger a kick.
2. restart KICK after the Deployment patch but before terminal observation.
3. prove recovery does not issue a second patch.
4. prove the request reaches the correct terminal state.

### Idempotency soak

After a terminal state, wait for multiple normal reconcile opportunities and assert that:

- no additional ReplicaSet appears;
- no terminal request returns to a non-terminal phase;
- no repeated warning events are emitted.

This reflects the principle that a controller can look healthy while still being trapped in a continuous update loop.

## Real and fake dependencies

Use the smallest realistic environment capable of proving the behavior:

- Real Argo CD MUST be used for ownership, Application status, self-heal, and sync-window compatibility tests.
- Pure KICK core tests that do not concern GitOps semantics MAY use a deterministic fake provider adapter deployed for tests.
- Argo CD behavior MUST NOT be simulated when the scenario claims Argo CD compatibility.
- External services unrelated to the behavior SHOULD be replaced by small in-cluster fixtures.

## Suite setup and teardown

The suite harness MUST:

1. create a uniquely named Kind cluster or use an explicitly supplied existing cluster;
2. build KICK from the current source tree;
3. load the exact image into Kind;
4. install CRDs and the Helm chart;
5. wait for controller readiness and leader election;
6. install the pinned Argo CD version for provider scenarios;
7. execute Chainsaw;
8. collect diagnostics before teardown on failure;
9. delete the cluster unless preservation was requested.

Supported environment controls SHOULD include:

```text
KIND_CLUSTER_NAME
KUBECONFIG
KICK_E2E_KEEP_CLUSTER=true
KICK_E2E_SKIP_ARGOCD=true
KICK_E2E_SCENARIO=KICK-E2E-032
KICK_E2E_ARTIFACT_DIR=...
```

The suite MUST never silently use an unrelated current kubectl context.

## Failure diagnostics

On any scenario or suite failure, collect at minimum:

```text
artifacts/e2e/<run-id>/
  cluster-info.txt
  nodes.yaml
  events.yaml
  kick/
    pods.yaml
    deployment.yaml
    logs-current.txt
    logs-previous.txt
  argocd/
    applications.yaml
    appprojects.yaml
    pods.yaml
    logs/
  workloads/
    deployments.yaml
    replicasets.yaml
    pods.yaml
  requests.yaml
  chainsaw/
    report.xml
    logs.txt
```

Also collect descriptions of failed or pending KickRequests and affected Deployments.

CI MUST upload the artifact directory even when the test step fails. Secrets MUST be redacted; Secret `.data` and `.stringData` MUST never be included.

## CI partitioning

Pull-request CI MUST run:

- render/validation of all scenarios;
- core smoke and behavior scenarios;
- mandatory Argo CD scenarios against the primary supported Argo CD version;
- feature-to-e2e traceability validation.

Nightly or scheduled CI SHOULD run:

- oldest and newest supported Kubernetes versions;
- all supported Argo CD minor versions;
- recovery, race, and soak scenarios;
- repeated execution to expose flakes.

Release CI MUST run the complete mandatory matrix.

Tests MAY be sharded by scenario class or provider. Sharding MUST preserve per-scenario artifacts and stable IDs.

## Flake policy

- Required scenarios MUST NOT be disabled or skipped without an approved, time-bounded rationale in traceability metadata.
- CI MUST NOT hide flakes through unconditional retries.
- A diagnostic retry MAY run once after failure, but the job remains failed and both attempts are retained.
- Assertions use Kubernetes conditions and bounded waits, never arbitrary sleeps as the primary synchronization mechanism.
- Time-sensitive sync-window tests SHOULD use short dynamically generated windows with sufficient clock margin.

## Test data rules

- Namespaces and names must be scenario-specific.
- Test Secrets contain obviously fake values.
- Assertions must never print Secret values.
- Tests must not depend on execution order between scenario directories.
- Cleanup must be idempotent.
- Images used by workload fixtures must be pinned and minimal.

## Source practices adopted

From DROP:

- scenario-based Chainsaw directories;
- separate success, failure, pacing/race, and discovery scenarios;
- one Make target for local execution;
- Kind-based execution with the locally built operator image.

From provider-keycloak:

- local environment setup that installs all real control-plane dependencies;
- render-only inspection before execution;
- lifecycle phases that verify apply, controller restart/recovery, stable observation, and deletion;
- explicit recognition that Ready alone does not prove convergence or absence of an update loop.

## Acceptance criteria

- Every required `KICK-E2E-NNN` has an independently runnable scenario directory.
- Scenario metadata and the central traceability matrix agree.
- The suite proves stable convergence, not only transient readiness.
- Controller restart and duplicate-action protection are exercised.
- Failures always produce useful, redacted artifacts.
- Local and CI entry points execute the same scenario files.
