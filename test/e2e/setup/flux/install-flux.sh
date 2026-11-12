#!/usr/bin/env bash
# Install a pinned Flux into the kind cluster for KICK provider e2e tests.
# Only source-, kustomize- and helm-controller are started; the scenarios do not
# use notifications or image automation.
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
FLUX_VERSION="${FLUX_VERSION:-v2.10.0}"
FLUX_NS="flux-system"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }

# Server-side: the Flux CRDs exceed the client-side-apply annotation limit.
kc apply --server-side --force-conflicts \
  -f "https://github.com/fluxcd/flux2/releases/download/${FLUX_VERSION}/install.yaml"

for deploy in notification-controller image-reflector-controller image-automation-controller source-watcher; do
  kc -n "$FLUX_NS" scale deploy "$deploy" --replicas=0 2>/dev/null || true
done
for deploy in source-controller kustomize-controller helm-controller; do
  kc -n "$FLUX_NS" rollout status "deploy/$deploy" --timeout=300s
done

echo "Flux ${FLUX_VERSION} installed in namespace ${FLUX_NS}"
