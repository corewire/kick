# Full KICK flow and cases

```mermaid
flowchart TD
    A[Secret or ConfigMap event] --> B{Relevant content changed?}
    B -- No --> Z[Ignore]
    B -- Yes --> C[Lookup consuming Deployments]
    C --> D[Create or update one KickRequest per Deployment]
    D --> E[Resolve GitOps provider and owner]
    E --> F{Exactly one owner?}
    F -- No --> F1[Block with typed reason]
    F -- Yes --> G[Evaluate provider schedule and reconciliation gate]
    G --> H{Gate open?}
    H -- No --> H1[Persist waiting state and requeue]
    H -- Yes --> I[Re-read Deployment, rollout, dependencies and observations]
    I --> J{Rollout active?}
    J -- Yes --> J1[Wait]
    J -- No --> K{Dependency newer than current rollout?}
    K -- No --> L[NoLongerRequired]
    K -- Yes --> M[Patch standard rollout restart annotation]
    M --> N[Observe rollout]
    N --> O{Complete?}
    O -- Yes --> P[Succeeded]
    O -- Failed or timeout --> Q[Failed]
```

## Important cases

### Argo CD OutOfSync

Wait until the Application is Synced. Then recompute freshness. An Argo CD-created ReplicaSet may remove the need for KICK.

### Window closed

Persist the request and re-evaluate when the window may open or when the AppProject changes.

### Several dependency changes

Coalesce into one active request. Re-read the complete dependency set before action and perform at most one rollout.

### Dependency removed

The removed source no longer participates. If no current dependency is newer, mark `NoLongerRequired`.

### Existing rollout

Wait for completion. Compare dependency changes against the resulting current ReplicaSet.

### Controller restart

Recover every non-terminal KickRequest. Timers and previous in-memory queue state are not authoritative.
