#!/usr/bin/env bash
# Install a pinned Argo CD into the kind cluster for KICK provider e2e tests.
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
ARGOCD_VERSION="${ARGOCD_VERSION:-v2.13.3}"
ARGOCD_NS="${ARGOCD_NS:-argocd}"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }

kc create namespace "$ARGOCD_NS" --dry-run=client -o yaml | kc apply -f -
kc apply -n "$ARGOCD_NS" \
  -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

kc -n "$ARGOCD_NS" rollout status deploy/argocd-repo-server --timeout=300s
kc -n "$ARGOCD_NS" rollout status statefulset/argocd-application-controller --timeout=300s

echo "Argo CD ${ARGOCD_VERSION} installed in namespace ${ARGOCD_NS}"
