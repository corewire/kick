#!/usr/bin/env bash
# hack/gen-asciinema.sh — Generate asciinema .cast files for the docs landing page.
# Requires: asciinema, kubectl, the kind-kick-dev cluster with KICK installed.
# Output: docs/static/casts/{restart,kickrequest,events}.cast — shown as tabs on the site.
#
# Kubeconfig and context are pinned to the local kind cluster once, up front, so
# recorded commands stay clean (plain `kubectl -n kick-demo ...`). Each recording
# runs from a clean namespace and waits for the KickRequest to settle Succeeded.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
CAST_DIR="$ROOT/docs/static/casts"
STEP_DIR="$(mktemp -d)"
mkdir -p "$CAST_DIR"
trap 'rm -rf "$STEP_DIR"' EXIT

# Target the local kind cluster — never a shared/remote context. Set once here so
# the recorded shells inherit it and the on-screen commands need no --context.
export KUBECONFIG="${KICK_KUBECONFIG:-$ROOT/.kubeconfig-kind-kick-dev}"
kubectl config use-context "${KIND_CONTEXT:-kind-kick-dev}" >/dev/null

TMPFILE="$STEP_DIR/kick-demo.yaml"
cat > "$TMPFILE" <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: kick-demo
---
apiVersion: kick.corewire.io/v1alpha1
kind: KickPolicy
metadata:
  name: default
  namespace: kick-demo
spec:
  # provider defaults to None: KICK self-gates and restarts immediately on a
  # stale dependency, no GitOps owner required.
  discovery:
    workloadSelector: {}
---
apiVersion: v1
kind: Secret
metadata:
  name: web-secret
  namespace: kick-demo
type: Opaque
stringData:
  API_TOKEN: alpha
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: kick-demo
  labels: { app: web }
spec:
  replicas: 1
  selector:
    matchLabels: { app: web }
  template:
    metadata:
      labels: { app: web }
    spec:
      containers:
        - name: app
          image: registry.k8s.io/pause:3.10
          envFrom:
            - secretRef:
                name: web-secret
EOF

cleanup() {
  kubectl delete namespace kick-demo --ignore-not-found >/dev/null 2>&1 || true
  kubectl wait --for=delete namespace/kick-demo --timeout=60s >/dev/null 2>&1 || true
}

# Apply resources and wait until KICK settles the baseline KickRequest (baseline
# never restarts) so a recording only shows the change-driven kick.
prime() {
  kubectl apply -f "$TMPFILE" >/dev/null
  kubectl -n kick-demo rollout status deploy/web --timeout=90s >/dev/null 2>&1 || true
  sleep 6
}

# ── Step script 1: the full story — rotate Secret, see the restart, see Succeeded
cat > "$STEP_DIR/restart.sh" <<'STEPEOF'
set -uo pipefail
NS=kick-demo
echo '# A Deployment consumes web-secret via envFrom. It is running and stable.'
sleep 2
echo '$ kubectl -n kick-demo get deploy web'
sleep 1
kubectl -n $NS get deploy web
sleep 3
echo ''
echo '# Rotate the Secret. Kubernetes alone would NOT restart the Pod.'
sleep 2
echo '$ kubectl -n kick-demo patch secret web-secret -p API_TOKEN=bravo'
sleep 1
kubectl -n $NS patch secret web-secret --type merge -p '{"stringData":{"API_TOKEN":"bravo"}}'
sleep 4
echo ''
echo '# KICK stamped kubectl.kubernetes.io/restartedAt and the Deployment rolled out:'
sleep 2
echo '$ kubectl -n kick-demo rollout status deploy/web'
sleep 1
kubectl -n $NS rollout status deploy/web --timeout=90s
sleep 3
echo ''
echo '# Exactly one KickRequest handled it — watch it settle Succeeded:'
sleep 2
echo '$ kubectl -n kick-demo get kickrequests -w'
sleep 1
kubectl -n $NS get kickrequests -w &
PID=$!
for _ in $(seq 1 45); do
  ph=$(kubectl -n $NS get kickrequest deployment-web -o jsonpath='{.status.phase}' 2>/dev/null || true)
  [ "$ph" = "Succeeded" ] && break
  sleep 2
done
sleep 3
kill $PID 2>/dev/null || true
sleep 5
printf '\n'
STEPEOF

# ── Step script 2: the KickRequest lifecycle table, ending Succeeded
cat > "$STEP_DIR/kickrequest.sh" <<'STEPEOF'
set -uo pipefail
NS=kick-demo
echo '# One coalesced KickRequest per target, from Pending to Succeeded.'
sleep 2
echo '$ kubectl -n kick-demo get kickrequests -w'
sleep 1
kubectl -n $NS get kickrequests -w &
PID=$!
sleep 3
kubectl -n $NS patch secret web-secret --type merge -p '{"stringData":{"API_TOKEN":"bravo"}}' >/dev/null 2>&1
for _ in $(seq 1 45); do
  ph=$(kubectl -n $NS get kickrequest deployment-web -o jsonpath='{.status.phase}' 2>/dev/null || true)
  [ "$ph" = "Succeeded" ] && break
  sleep 2
done
sleep 3
kill $PID 2>/dev/null || true
sleep 2
echo ''
echo '$ kubectl -n kick-demo get kickrequests'
sleep 1
kubectl -n $NS get kickrequests
sleep 6
printf '\n'
STEPEOF

# ── Step script 3: the underlying Kubernetes events
cat > "$STEP_DIR/events.sh" <<'STEPEOF'
set -uo pipefail
NS=kick-demo
echo '# The restart is a normal, auditable Kubernetes rollout — watch the events.'
sleep 2
echo '$ kubectl -n kick-demo get events --watch-only'
sleep 1
kubectl -n $NS get events --field-selector reason!=LeaderElection --watch-only &
PID=$!
sleep 3
kubectl -n $NS patch secret web-secret --type merge -p '{"stringData":{"API_TOKEN":"bravo"}}' >/dev/null 2>&1
sleep 16
kill $PID 2>/dev/null || true
sleep 3
printf '\n'
STEPEOF

# --idle-time-limit caps long silent gaps (e.g. the ~30s requeue wait) so playback
# stays snappy while the recording still captures the real Succeeded transition.
cleanup; prime
echo "Recording 1/3: restart"
asciinema rec "$CAST_DIR/restart.cast" --overwrite --cols 92 --rows 24 --idle-time-limit 2 --env "" -c "bash --norc --noprofile $STEP_DIR/restart.sh"

cleanup; prime
echo "Recording 2/3: kickrequest"
asciinema rec "$CAST_DIR/kickrequest.cast" --overwrite --cols 92 --rows 24 --idle-time-limit 2 --env "" -c "bash --norc --noprofile $STEP_DIR/kickrequest.sh"

cleanup; prime
echo "Recording 3/3: events"
asciinema rec "$CAST_DIR/events.cast" --overwrite --cols 110 --rows 24 --idle-time-limit 2 --env "" -c "bash --norc --noprofile $STEP_DIR/events.sh"

cleanup
echo "Generated: $CAST_DIR/{restart,kickrequest,events}.cast"
