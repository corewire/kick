# 
# Troubleshooting

## KickRequest stays in waiting phase

- `WaitingForOwner`: owner missing or ambiguous.
- `WaitingForGate`: provider schedule/window blocks restart.
- `WaitingForApplicationSync`: Argo CD app is not synced.
- `WaitingForRollout`: existing rollout still active.

Check:

```bash
kubectl --kubeconfig .kubeconfig-kind-kick-dev --context kind-kick-dev -n <ns> describe kickrequest <name>
```

## Request fails

Typical causes:

- rollout timeout;
- patch rejection;
- provider query failure.

Inspect controller logs and request conditions.

## No request created after source update

Confirm the source is a supported dependency type and consumed by a managed Deployment.
