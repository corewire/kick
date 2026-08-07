#!/usr/bin/env bash
# Render an Argo CD Application that reads this repo and ALWAYS targets the
# current git branch. Never hardcode the revision.
#
# Usage: render-application.sh <app-name> <app-namespace> <repo-path> <dest-namespace> [project]
set -euo pipefail

APP_NAME="${1:?app name required}"
APP_NS="${2:?application namespace required}"
REPO_PATH="${3:?repo path required}"
DEST_NS="${4:?destination namespace required}"
PROJECT="${5:-default}"

REPO_URL="${KICK_E2E_REPO_URL:-https://github.com/corewire/kick.git}"
BRANCH="$(git rev-parse --abbrev-ref HEAD)"

if [[ "$BRANCH" == "HEAD" ]]; then
  echo "detached HEAD: set KICK_E2E_BRANCH explicitly" >&2
  BRANCH="${KICK_E2E_BRANCH:?branch required in detached HEAD}"
fi

cat <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ${APP_NAME}
  namespace: ${APP_NS}
spec:
  project: ${PROJECT}
  source:
    repoURL: ${REPO_URL}
    path: ${REPO_PATH}
    targetRevision: ${BRANCH}
  destination:
    server: https://kubernetes.default.svc
    namespace: ${DEST_NS}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
