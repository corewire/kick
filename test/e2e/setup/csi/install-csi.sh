#!/usr/bin/env bash
# Install the Secrets Store CSI stack the KICK CSI e2e suite runs against:
# the upstream driver, an OpenBao dev server and the OpenBao CSI provider.
#
# Every component is the real one. The driver's own rotation reconciler polls
# the provider, and the provider reports an object version that is an HMAC of
# the secret's content, so a byte-identical write produces no version change.
# KICK observes only what the driver writes to SecretProviderClassPodStatus.
#
# The driver CRDs must exist before the KICK manager starts, because optional
# integrations are probed once at start-up. The Makefile orders this target
# ahead of e2e-install for that reason.
#
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
CSI_NS="${CSI_NS:-csi}"
CSI_DRIVER_VERSION="${CSI_DRIVER_VERSION:-1.4.6}"
CSI_PROVIDER_VERSION="${CSI_PROVIDER_VERSION:-v2.0.3}"
# Rotation is the signal under test, so the driver has to poll well inside the
# integration suite's assert budget. The chart default is 2m.
CSI_ROTATION_POLL_INTERVAL="${CSI_ROTATION_POLL_INTERVAL:-15s}"
# Namespaces whose default service account may read the suite's secrets. Listed
# explicitly rather than bound with a wildcard.
CSI_SCENARIO_NAMESPACES="${CSI_SCENARIO_NAMESPACES:-kick-e2e-064,kick-e2e-065,kick-e2e-066,kick-e2e-067}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$CONTEXT" "$@"; }
helm_csi() { helm --kubeconfig "$KUBECONFIG_PATH" --kube-context "$CONTEXT" "$@"; }
bao() { kc -n "$CSI_NS" exec -i deploy/openbao -- bao "$@"; }

# The driver mints the service-account token the provider logs in with, so its
# audience has to match the audience bound to the OpenBao auth role below.
helm_csi upgrade --install secrets-store-csi-driver secrets-store-csi-driver \
  --repo https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts \
  --version "$CSI_DRIVER_VERSION" \
  --namespace "$CSI_NS" --create-namespace \
  --wait --timeout 5m \
  --set enableSecretRotation=true \
  --set rotationPollInterval="$CSI_ROTATION_POLL_INTERVAL" \
  --set 'tokenRequests[0].audience=openbao'

kc apply -f "${SCRIPT_DIR}/openbao.yaml"
kc -n "$CSI_NS" rollout status deploy/openbao --timeout=300s

# The upstream provider manifest omits metadata.namespace on its Role and
# RoleBinding, so without an explicit -n they land in whatever namespace the
# kubeconfig points at. The provider then cannot read its HMAC key secret and
# silently reports every mounted object with an empty version, which would make
# every rotation invisible to KICK and every scenario below vacuous.
kc apply -n "$CSI_NS" -f "https://raw.githubusercontent.com/openbao/openbao-csi-provider/${CSI_PROVIDER_VERSION}/deployment/openbao-csi-provider.yaml"
kc -n "$CSI_NS" rollout status daemonset/openbao-csi-provider --timeout=300s
kc -n "$CSI_NS" get role openbao-csi-provider-role >/dev/null

# Bootstrap OpenBao. Every step is idempotent so the target can be re-run
# against a cluster that already has the stack installed.
bao auth enable kubernetes >/dev/null 2>&1 || true
# The API server address is resolved inside the OpenBao pod, so the variable
# must survive this shell unexpanded.
# shellcheck disable=SC2016
kc -n "$CSI_NS" exec -i deploy/openbao -- sh -ec \
  'bao write auth/kubernetes/config kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443"' >/dev/null

echo 'path "secret/data/*" { capabilities = ["read"] }' |
  bao policy write kick-e2e - >/dev/null

bao write auth/kubernetes/role/kick-e2e \
  bound_service_account_names=default \
  bound_service_account_namespaces="$CSI_SCENARIO_NAMESPACES" \
  audience=openbao \
  policies=kick-e2e \
  ttl=20m >/dev/null

# Fail loudly rather than leaving the manager to start without the integration.
for crd in secretproviderclasses.secrets-store.csi.x-k8s.io \
  secretproviderclasspodstatuses.secrets-store.csi.x-k8s.io; do
  kc get crd "$crd" >/dev/null
done

echo "Secrets Store CSI driver ${CSI_DRIVER_VERSION} + OpenBao provider ${CSI_PROVIDER_VERSION} installed in namespace ${CSI_NS} (rotation poll ${CSI_ROTATION_POLL_INTERVAL})"
