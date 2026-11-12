#!/usr/bin/env bash
# Wait until a live, non-terminating manager pod holds the leader lease.
#
# `rollout status` returns once the new pod is Ready, but a replaced leader that
# did not release the lease keeps it until expiry. Scenarios started in that
# window change sources before KICK observes them (issue #34).
#
# Usage: hack/e2e-wait-leader.sh [timeout-seconds]
set -euo pipefail

TIMEOUT="${1:-120}"
CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" -n kick-system "$@"; }

deadline=$(( $(date +%s) + TIMEOUT ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  holder=$(kc get lease kick.corewire.io -o jsonpath='{.spec.holderIdentity}' 2>/dev/null || true)
  pod="${holder%%_*}"
  if [ -n "$pod" ] && [ -z "$(kc get pod "$pod" -o jsonpath='{.metadata.deletionTimestamp}' 2>/dev/null || echo gone)" ]; then
    echo "leader: $pod"
    exit 0
  fi
  sleep 2
done
echo "no live manager pod acquired the leader lease within ${TIMEOUT}s (holder: ${holder:-none})" >&2
exit 1
