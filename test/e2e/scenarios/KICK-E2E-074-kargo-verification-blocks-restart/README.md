# KICK-E2E-074 - Kargo verification blocks restart

## Behavior under test
A running AnalysisRun blocks the restart. After that verification is Successful, a stale workload restarts exactly once.

The Stage verification is a real AnalysisTemplate. Kargo creates the AnalysisRun. The scenario does not write Stage status.

## Setup
- resources.yaml
- manifests/

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-074
