# KICK-E2E-075 - Kargo failed verification still restarts

## Behavior under test
A Failed AnalysisRun does not permanently block a stale workload. The restart happens once, after verification is terminal.

The Stage verification is a real AnalysisTemplate. Kargo creates the AnalysisRun. The scenario does not write Stage status.

## Setup
- resources.yaml
- manifests/

## Traceability
- [trace.yaml](./trace.yaml)
- **Scenario ID**: KICK-E2E-075
