# KICK-E2E-076 - Kargo reverification after restart

## Behavior under test
With reverifyAfterRestart, KICK patches kargo.akuity.io/reverify once after the rollout and does not restart again.

The Stage verification is a real AnalysisTemplate. Kargo creates the AnalysisRun. The scenario does not write Stage status.

## Setup
- resources.yaml
- manifests/

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-076
