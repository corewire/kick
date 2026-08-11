#!/usr/bin/env bash
# Install a pinned Argo Rollouts into the kind cluster for KICK provider e2e tests.
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
ROLLOUTS_VERSION="${ROLLOUTS_VERSION:-v1.7.2}"
ROLLOUTS_NS="${ROLLOUTS_NS:-argo-rollouts}"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }

kc create namespace "$ROLLOUTS_NS" --dry-run=client -o yaml | kc apply -f -
kc apply -n "$ROLLOUTS_NS" \
  -f "https://github.com/argoproj/argo-rollouts/releases/download/${ROLLOUTS_VERSION}/install.yaml"

kc -n "$ROLLOUTS_NS" rollout status deploy/argo-rollouts --timeout=300s

echo "Argo Rollouts ${ROLLOUTS_VERSION} installed in namespace ${ROLLOUTS_NS}"
