#!/usr/bin/env bash
# Install an in-cluster Gitea for KICK integration e2e tests.
#
# The suites must not depend on a public forge: Argo CD and Kargo have to clone
# and push against a repository whose contents a scenario controls, and pushing
# to github.com from a test is not an option. Gitea is backed by emptyDir, so
# every install starts from an empty instance.
#
# Requires explicit kubeconfig/context; never uses an ambient context.
set -euo pipefail

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-.kubeconfig-kind-kick-dev}"
GITEA_NS="${GITEA_NS:-kick-e2e-git}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=test/e2e/setup/gitea/lib.sh
source "${SCRIPT_DIR}/lib.sh"

kc apply -f "${SCRIPT_DIR}/gitea.yaml"
kc -n "$GITEA_NS" rollout status deploy/gitea --timeout=300s

gitea_ensure_admin

# `apps` holds manifests Argo CD renders. `promotions` is the repository Kargo
# writes to during a promotion; keeping them apart stops a Kargo push from
# racing an Argo CD scenario.
gitea_create_repo apps
gitea_create_repo promotions

echo "Gitea installed in namespace ${GITEA_NS} at ${GITEA_INTERNAL_URL}"
