#!/usr/bin/env bash
# Install cert-manager and a real Kargo control plane for the KICK Kargo e2e
# suite.
#
# Kargo's own controller drives the promotions the scenarios assert on; nothing
# in the suite writes Kargo status by hand. cert-manager is a hard requirement:
# the chart's webhooks server defaults to a self-signed certificate issued
# through cert-manager, and there is no provision for running it without TLS.
#
# Only the components the suite needs are enabled. The API server, garbage
# collector and external webhooks server stay off, which also keeps the
# footprint of the kind cluster down.
#
# The Kargo CRDs must exist before the KICK manager starts, because optional
# integrations are probed once at start-up. The Makefile orders this target
# ahead of e2e-install for that reason.
#
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
KARGO_NS="${KARGO_NS:-kargo}"
KARGO_VERSION="${KARGO_VERSION:-1.11.1}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.16.2}"
ARGOCD_NS="${ARGOCD_NS:-argocd}"
# Floor Kargo enforces on Warehouse reconciliation. The chart default is 5m.
KARGO_WAREHOUSE_INTERVAL="${KARGO_WAREHOUSE_INTERVAL:-20s}"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }
helm_kargo() { helm --kubeconfig "$KUBECONFIG_PATH" --kube-context "$CONTEXT" "$@"; }

kc apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
for deploy in cert-manager cert-manager-webhook cert-manager-cainjector; do
  kc -n cert-manager rollout status "deployment/${deploy}" --timeout=300s
done

# Argo CD integration on: the controller factors Application health and sync
# state into Stage health and runs the argocd-update promotion step. Argo
# Rollouts integration off: the suite never verifies Freight through analysis,
# and disabling it grants the controller fewer permissions.
#
# The chart enforces a five-minute floor on Warehouse reconciliation, which is
# longer than a scenario is willing to wait for a commit to be discovered. The
# floor is lowered here rather than in the scenarios, so nothing in the suite
# depends on Kargo noticing a commit faster than it is configured to.
#
# The in-cluster Gitea used by the suite serves plain HTTP, and Kargo refuses to
# send Git credentials over HTTP unless told otherwise. This is a test-only
# concession; a real installation should keep the default.
helm_kargo upgrade --install kargo oci://ghcr.io/akuity/kargo-charts/kargo \
  --version "$KARGO_VERSION" \
  --namespace "$KARGO_NS" --create-namespace \
  --wait --timeout 10m \
  --set api.enabled=false \
  --set garbageCollector.enabled=false \
  --set externalWebhooksServer.enabled=false \
  --set controller.argocd.integrationEnabled=true \
  --set controller.argocd.namespace="$ARGOCD_NS" \
  --set controller.rollouts.integrationEnabled=false \
  --set controller.allowCredentialsOverHTTP=true \
  --set controller.reconcilers.warehouses.minReconciliationInterval="$KARGO_WAREHOUSE_INTERVAL"

kc -n "$KARGO_NS" rollout status deployment/kargo-controller --timeout=300s
kc -n "$KARGO_NS" rollout status deployment/kargo-management-controller --timeout=300s
kc -n "$KARGO_NS" rollout status deployment/kargo-webhooks-server --timeout=300s

# Fail loudly rather than leaving the manager to start without the integration.
for crd in projects.kargo.akuity.io stages.kargo.akuity.io \
  warehouses.kargo.akuity.io promotions.kargo.akuity.io; do
  kc get crd "$crd" >/dev/null
done

echo "Kargo ${KARGO_VERSION} installed in namespace ${KARGO_NS} (cert-manager ${CERT_MANAGER_VERSION})"
