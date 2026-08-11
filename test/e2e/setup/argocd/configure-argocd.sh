#!/usr/bin/env bash
# Configure an already installed Argo CD for the KICK integration scenarios:
# allow Applications to live next to their workloads ("apps in any namespace")
# and provide the shared AppProject those Applications use.
# Idempotent: the controller is only restarted when the setting actually changes.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
ARGOCD_NS="${ARGOCD_NS:-argocd}"
APP_NAMESPACES="${KICK_E2E_APP_NAMESPACES:-kick-e2e-*}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }

kc apply -f "${script_dir}/appproject.yaml"

current="$(kc -n "$ARGOCD_NS" get configmap argocd-cmd-params-cm \
  -o jsonpath='{.data.application\.namespaces}' 2>/dev/null || true)"

if [[ "$current" == "$APP_NAMESPACES" ]]; then
  echo "Argo CD already allows Applications in ${APP_NAMESPACES}"
  exit 0
fi

kc -n "$ARGOCD_NS" patch configmap argocd-cmd-params-cm --type merge \
  -p "{\"data\":{\"application.namespaces\":\"${APP_NAMESPACES}\"}}"
kc -n "$ARGOCD_NS" rollout restart statefulset/argocd-application-controller
kc -n "$ARGOCD_NS" rollout restart deployment/argocd-server
kc -n "$ARGOCD_NS" rollout status statefulset/argocd-application-controller --timeout=300s
kc -n "$ARGOCD_NS" rollout status deployment/argocd-server --timeout=300s

echo "Argo CD now allows Applications in ${APP_NAMESPACES}"
