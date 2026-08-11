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
- `KickDryRun`
- `OwnerUnknown`
- `OwnerAmbiguous`

Use `kubectl describe kickrequest <name>` to inspect event history for a request

`KickDryRun` is emitted instead of `KickStarted`/`KickSucceeded` when the
matching KickPolicy sets `spec.dryRun: true`. The restart is never performed..