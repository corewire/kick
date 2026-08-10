#!/usr/bin/env bash
# hack/gen-asciinema.sh — Generate asciinema .cast files for the docs landing page.
# Requires: asciinema, kubectl, the kind-kick-dev cluster with KICK installed.
# Output: docs/static/casts/{restart,kickrequest,events}.cast — shown as tabs on the site.
#
# Each recording is independent: clean state -> apply -> rotate a Secret -> watch
# one perspective of KICK turning that change into exactly one gated restart.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
CAST_DIR="$ROOT/docs/static/casts"
mkdir -p "$CAST_DIR"

# Always target the local kind cluster — never a shared/remote context.
export KUBECONFIG="${KICK_KUBECONFIG:-$ROOT/.kubeconfig-kind-kick-dev}"
CONTEXT="${KIND_CONTEXT:-kind-kick-dev}"
KUBECTL="kubectl --context $CONTEXT -n kick-demo"

TMPFILE="/tmp/kick-demo.yaml"
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
  kubectl --context "$CONTEXT" delete namespace kick-demo --ignore-not-found >/dev/null 2>&1 || true
  # Wait for full teardown so each recording starts from a clean baseline.
  kubectl --context "$CONTEXT" wait --for=delete namespace/kick-demo --timeout=60s >/dev/null 2>&1 || true
}

# Apply resources and wait until KICK has settled the baseline KickRequest
# (baseline never restarts) so the recording only shows the change-driven kick.
prime() {
  kubectl --context "$CONTEXT" apply -f "$TMPFILE" >/dev/null
  $KUBECTL rollout status deploy/web --timeout=60s >/dev/null 2>&1 || true
  sleep 6
}

# ─── Recording 1: rotate a Secret → KICK restarts the Deployment ──────────────
cleanup
prime
echo "Recording 1/3: restart"
asciinema rec "$CAST_DIR/restart.cast" --overwrite --cols 92 --rows 24 --env "" -c "bash --norc --noprofile <<'REC'
echo '# A Deployment consumes web-secret via envFrom. It is running and stable.'
echo '\$ $KUBECTL get deploy web'
$KUBECTL get deploy web
sleep 3
echo ''
echo '# Rotate the Secret. Kubernetes alone would NOT restart the Pod.'
echo '\$ $KUBECTL patch secret web-secret --type merge -p API_TOKEN=bravo'
$KUBECTL patch secret web-secret --type merge -p '{\"stringData\":{\"API_TOKEN\":\"bravo\"}}'
sleep 5
echo ''
echo '# KICK noticed the change and stamped a fresh rollout:'
echo '\$ $KUBECTL get deploy web -o jsonpath restartedAt'
$KUBECTL get deploy web -o jsonpath='{.spec.template.metadata.annotations.kubectl\.kubernetes\.io/restartedAt}'
printf '\n'
sleep 1
echo '\$ $KUBECTL rollout status deploy/web'
$KUBECTL rollout status deploy/web --timeout=60s
sleep 3
printf '\n'
REC"

# ─── Recording 2: one coalesced KickRequest per target ────────────────────────
cleanup
prime
echo "Recording 2/3: kickrequest"
asciinema rec "$CAST_DIR/kickrequest.cast" --overwrite --cols 92 --rows 24 --env "" -c "bash --norc --noprofile <<'REC'
echo '\$ $KUBECTL get kickrequests -w'
$KUBECTL get kickrequests -w &
PID=\$!
sleep 2
$KUBECTL patch secret web-secret --type merge -p '{\"stringData\":{\"API_TOKEN\":\"bravo\"}}' >/dev/null 2>&1
sleep 12
kill \$PID 2>/dev/null || true
sleep 2
printf '\n'
REC"

# ─── Recording 3: Kubernetes events emitted by KICK ───────────────────────────
cleanup
prime
echo "Recording 3/3: events"
asciinema rec "$CAST_DIR/events.cast" --overwrite --cols 110 --rows 24 --env "" -c "bash --norc --noprofile <<'REC'
echo '\$ $KUBECTL get events --field-selector reason!=LeaderElection --watch-only'
$KUBECTL get events --field-selector reason!=LeaderElection --watch-only &
PID=\$!
sleep 2
$KUBECTL patch secret web-secret --type merge -p '{\"stringData\":{\"API_TOKEN\":\"bravo\"}}' >/dev/null 2>&1
sleep 12
kill \$PID 2>/dev/null || true
sleep 2
printf '\n'
REC"

cleanup
rm -f "$TMPFILE"
echo "✓ Generated: $CAST_DIR/{restart,kickrequest,events}.cast"
