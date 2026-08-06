# Boilerplate status

## Implemented

- Kubebuilder-compatible layout and controller manager.
- `KickRequest` v1alpha1 type skeleton.
- Provider-neutral GitOps contract.
- Deployment dependency extraction for env, envFrom, Secret/ConfigMap volumes, projected volumes, and init containers.
- Explicit exclusion of imagePullSecrets.
- Unit tests for extraction and deduplication.
- CRD/RBAC/Kustomize starter manifests.
- Traceability checker and e2e scenario template.

## Intentionally not implemented

- durable Secret/ConfigMap content-change observation;
- update timestamp semantics;
- ReplicaSet freshness evaluation;
- Argo CD tracking-id parsing and fallback ownership index;
- AppProject sync-window evaluation;
- restart execution;
- rollout completion observation;
- production Helm chart and release workflows.

These items remain governed by the modular task and specification files.
