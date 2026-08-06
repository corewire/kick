# Observability

## Conditions

KickRequest SHOULD expose conditions including:

```text
DependenciesObserved
OwnerResolved
GateOpen
OwnerReconciled
RolloutIdle
RestartRequired
RestartApplied
RolloutComplete
```

Every condition uses stable type, status, reason, message, observed generation, and transition time.

## Events

Emit Kubernetes events for meaningful transitions only:

```text
KickRequired
WaitingForSchedule
WaitingForGitOpsSync
WaitingForRollout
KickStarted
KickSucceeded
KickNoLongerRequired
KickFailed
OwnerUnknown
OwnerAmbiguous
```

Do not emit an event on every retry.

## Metrics

Suggested low-cardinality metrics:

```text
kick_requests_total{provider,result}
kick_requests_pending{provider,reason}
kick_restarts_total{provider,result}
kick_gate_wait_seconds{provider,reason}
kick_rollout_duration_seconds{provider,result}
kick_source_events_total{kind,result}
kick_controller_errors_total{controller,reason}
```

Avoid labels containing Secret name, ConfigMap name, Deployment name, namespace, Application name, or request UID unless explicitly justified.

## Logging

Structured logs SHOULD include safe object references and reconciliation IDs.

Never log Secret values, serialized Secret objects, or content hashes.

Expected log levels:

- info: state transitions and successful kicks;
- debug: reconciliation decisions and cache/index details;
- warning: user-correctable blocked states;
- error: unexpected controller failures.

## CLI and kubectl UX

The CRD SHOULD support useful printer columns:

```text
TARGET
PHASE
PROVIDER
OWNER
REASON
AGE
```

## Acceptance criteria

- Tests verify stable reason strings.
- Metrics pass linting and cardinality review.
- Secret content cannot appear in captured logs or events.
- Pending-state transitions are visible without debug logs.
