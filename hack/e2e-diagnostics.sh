#!/usr/bin/env bash
# Collect cluster state after a failed e2e run: control-plane pod logs, events,
# and the Argo CD / Kargo objects scenarios assert on. Never dumps Secrets or
# ConfigMaps.
#
# Usage: hack/e2e-diagnostics.sh <output-dir>
# Requires explicit kubeconfig/context; never uses an ambient context.
set -uo pipefail

OUT="${1:?output directory required}"
CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
NAMESPACES=(kick-system argocd kargo kick-e2e-git argo-rollouts cert-manager csi kube-system)

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }

mkdir -p "$OUT"

kc get pods -A -o wide > "$OUT/pods.txt" 2>&1
kc get events -A --sort-by=.lastTimestamp > "$OUT/events.txt" 2>&1

for kind in applications.argoproj.io appprojects.argoproj.io rollouts.argoproj.io \
  stages.kargo.akuity.io promotions.kargo.akuity.io freights.kargo.akuity.io \
  warehouses.kargo.akuity.io analysisruns.argoproj.io \
  kickrequests.kick.corewire.io kickpolicies.kick.corewire.io; do
  if kc get crd "$kind" >/dev/null 2>&1; then
    kc get "$kind" -A -o yaml > "$OUT/${kind%%.*}.yaml" 2>&1
  fi
done

for ns in "${NAMESPACES[@]}"; do
  kc get namespace "$ns" >/dev/null 2>&1 || continue
  for pod in $(kc -n "$ns" get pods -o name 2>/dev/null); do
    name="${pod#pod/}"
    kc -n "$ns" logs "$name" --all-containers --prefix --timestamps --tail=5000 \
      > "$OUT/logs-${ns}-${name}.txt" 2>&1
  done
done

echo "e2e diagnostics written to $OUT"
