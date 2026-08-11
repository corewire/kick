#!/usr/bin/env bash
# Shared helpers for driving the in-cluster Gitea from e2e scenarios.
#
# Git runs inside the Gitea pod: the kind node IP is not routable from the host,
# and a background port-forward is the usual source of flakes in this fixture.
# Scripts are piped to `sh -s` with positional arguments so nothing is
# interpolated into a nested shell.

CONTEXT="${KICK_E2E_CONTEXT:-kind-kick-dev}"
KUBECONFIG_PATH="${KICK_E2E_KUBECONFIG:-${KUBECONFIG:-.kubeconfig-kind-kick-dev}}"
GITEA_NS="${GITEA_NS:-kick-e2e-git}"
GITEA_USER="${GITEA_USER:-kick-e2e}"
GITEA_PASSWORD="${GITEA_PASSWORD:-kick-e2e-throwaway}"

# Cluster-internal base URL. Anonymous clone works, so Argo CD needs no
# credentials; only pushes authenticate.
GITEA_INTERNAL_URL="http://gitea.${GITEA_NS}.svc.cluster.local:3000"

kc() { kubectl --kubeconfig "$KUBECONFIG_PATH" --context "$(gitea_context)" "$@"; }

# Chainsaw hands scripts a generated kubeconfig that does not contain the kind
# context name, so the context is resolved rather than assumed.
gitea_context() {
  if [[ -n "${_GITEA_CONTEXT:-}" ]]; then
    echo "$_GITEA_CONTEXT"
    return
  fi
  if kubectl --kubeconfig "$KUBECONFIG_PATH" config get-contexts -o name 2>/dev/null | grep -qx "$CONTEXT"; then
    _GITEA_CONTEXT="$CONTEXT"
  else
    _GITEA_CONTEXT="$(kubectl --kubeconfig "$KUBECONFIG_PATH" config current-context)"
  fi
  echo "$_GITEA_CONTEXT"
}

# Run a script from stdin inside the Gitea pod with "$@" set to the extra args.
gitea_sh() { kc -n "$GITEA_NS" exec -i deploy/gitea -- sh -s -- "$@"; }

gitea_repo_url() { echo "${GITEA_INTERNAL_URL}/${GITEA_USER}/${1}.git"; }

# `gitea admin user create` refuses to run as root and needs the generated
# app.ini, so it runs as the `git` user against the on-disk config.
gitea_ensure_admin() {
  gitea_sh "$GITEA_USER" "$GITEA_PASSWORD" <<'EOF'
set -eu
conf=/data/gitea/conf/app.ini
if su git -c "gitea admin user list --config $conf" | awk 'NR > 1 { print $2 }' | grep -qx "$1"; then
  echo "admin user $1 already exists"
  exit 0
fi
su git -c "gitea admin user create --config $conf --username $1 --password $2 --email $1@example.invalid --admin --must-change-password=false"
EOF
}

# Gitea serves a repository over HTTP only when a database record exists, so
# repositories are created through the API rather than with `git init --bare`.
gitea_create_repo() {
  gitea_sh "$GITEA_USER" "$GITEA_PASSWORD" "$1" <<'EOF'
set -eu
code=$(curl -s -o /dev/null -w '%{http_code}' -u "$1:$2" \
  -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:3000/api/v1/user/repos \
  -d "{\"name\":\"$3\",\"private\":false,\"auto_init\":true,\"default_branch\":\"main\"}")
case "$code" in
201) echo "created repository $3" ;;
409) echo "repository $3 already exists" ;;
*) echo "unexpected status $code creating repository $3" >&2; exit 1 ;;
esac
EOF
}

_gitea_clone() {
  gitea_sh "$1" "$(gitea_repo_url "$1")" <<'EOF'
set -eu
work="/tmp/kick-e2e/$1"
rm -rf "$work"
mkdir -p /tmp/kick-e2e
git clone -q "$2" "$work"
EOF
}

_gitea_push() {
  gitea_sh "$1" "$2" "$3" "$GITEA_USER" "$GITEA_PASSWORD" <<'EOF'
set -eu
cd "/tmp/kick-e2e/$1"
git config user.email "$4@example.invalid"
git config user.name "$4"
git add -A "$2"
if git diff --cached --quiet; then
  echo "no change to commit for $1/$2"
  exit 0
fi
git commit -q -m "$3"
git push -q "http://$4:$5@127.0.0.1:3000/$4/$1.git" HEAD:main
echo "pushed $1/$2"
EOF
}

# Replace <path> in <repo> with the contents of <local-dir> and push.
#
# Every scenario owns its repository, so parallel scenarios never share a clone
# directory or race each other's pushes on main. The path is replaced rather
# than merged, so a scenario's initial repository state is a pure function of
# its fixture directory regardless of what ran before it.
gitea_seed_dir() {
  local repo="$1" path="$2" local_dir="$3" message="${4:-seed $2}"
  gitea_create_repo "$repo" >/dev/null
  _gitea_clone "$repo"
  gitea_sh "$repo" "$path" <<'EOF'
set -eu
work="/tmp/kick-e2e/$1"
rm -rf "${work:?}/$2"
mkdir -p "$work/$2"
EOF
  tar cf - -C "$local_dir" . |
    kc -n "$GITEA_NS" exec -i deploy/gitea -- tar xf - -C "/tmp/kick-e2e/${repo}/${path}"
  _gitea_push "$repo" "$path" "$message"
}

# Write a single file below <path> and push, leaving the rest of the path alone.
# This is how a scenario simulates an operator editing one manifest in git.
gitea_commit_file() {
  local repo="$1" path="$2" local_file="$3" message="${4:-update $2}"
  _gitea_clone "$repo"
  tar cf - -C "$(dirname "$local_file")" "$(basename "$local_file")" |
    kc -n "$GITEA_NS" exec -i deploy/gitea -- tar xf - -C "/tmp/kick-e2e/${repo}/${path}"
  _gitea_push "$repo" "$path" "$message"
}

# Latest commit on main; used to assert a promotion wrote a commit.
gitea_head_sha() {
  gitea_sh "$(gitea_repo_url "$1")" <<'EOF'
set -eu
git ls-remote "$1" refs/heads/main | cut -f1
EOF
}
