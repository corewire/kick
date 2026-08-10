# Events Reference

KICK emits typed Kubernetes events with stable reason strings.

Reasons:

- `WaitingForSchedule`
- `WaitingForGitOpsSync`
- `WaitingForRollout`
- `KickStarted`
- `KickSucceeded`
- `KickNoLongerRequired`
- `KickFailed`
- `OwnerUnknown`
- `OwnerAmbiguous`

Use `kubectl describe kickrequest <name>` to inspect event history for a request.