#!/usr/bin/env python3
"""Regenerate real-restart chainsaw tests for KICK discovery/trigger scenarios.

Baseline test strategy (proves operator functionality AND correctness):
  1. apply base resources (provider None policy + workload + dependency)
  2. wait until the baseline KickRequest settles terminal WITHOUT restarting the
     workload  -> correctness: baseline does not spuriously restart
  3. patch the dependency (a relevant data change)
  4. assert the operator ACTUALLY restarts the workload: the pod template gets
     kubectl.kubernetes.io/restartedAt stamped -> real rollout
"""
import glob
import os
import re

# scenario -> (kickrequest name, workload kubectl kind, workload name, patch cmd tail)
SECRET_PATCH = "patch secret {name} --type merge -p '{{\"stringData\":{{\"token\":\"bravo\"}}}}'"
CONFIG_PATCH = "patch configmap {name} --type merge -p '{{\"data\":{{\"mode\":\"bravo\"}}}}'"

SCENARIOS = {
    "001": ("deployment-app-001", "deployment", "app-001", SECRET_PATCH.format(name="app-secret")),
    "002": ("deployment-app-002", "deployment", "app-002", SECRET_PATCH.format(name="app-secret")),
    "003": ("deployment-app-003", "deployment", "app-003", CONFIG_PATCH.format(name="app-config")),
    "004": ("deployment-app-004", "deployment", "app-004", SECRET_PATCH.format(name="app-secret")),
    "005": ("deployment-app-005", "deployment", "app-005", CONFIG_PATCH.format(name="app-config")),
    "006": ("deployment-app-006", "deployment", "app-006", SECRET_PATCH.format(name="app-secret")),
    "007": ("deployment-app-007", "deployment", "app-007", SECRET_PATCH.format(name="app-secret")),
    "011": ("deployment-secret-api", "deployment", "secret-api", SECRET_PATCH.format(name="app-secret")),
    "012": ("deployment-config-api", "deployment", "config-api", CONFIG_PATCH.format(name="app-config")),
    "054": ("statefulset-app-054", "statefulset", "app-054", SECRET_PATCH.format(name="app-secret")),
    "055": ("daemonset-app-055", "daemonset", "app-055", SECRET_PATCH.format(name="app-secret")),
}

RA_JSONPATH = r"{.spec.template.metadata.annotations.kubectl\.kubernetes\.io/restartedAt}"


def assert_step(ns, kr, wlkind, wlname, patch):
    return f"""    - name: assert-dependency-change-restarts-workload
      try:
        - script:
            content: |
              set -eu
              NS={ns}
              deadline=$(( $(date +%s) + 150 ))

              # 1. Baseline: KICK discovers the dependency and settles the request
              #    terminal by adopting the workload's initial rollout.
              while true; do
                phase=$(kubectl -n $NS get kickrequest {kr} -o jsonpath='{{.status.phase}}' 2>/dev/null || true)
                if [ "$phase" = "Succeeded" ] || [ "$phase" = "NoLongerRequired" ]; then
                  break
                fi
                if [ "$(date +%s)" -ge "$deadline" ]; then
                  echo "baseline KickRequest {kr} never settled" >&2
                  kubectl -n $NS get kickrequests -o yaml >&2 || true
                  exit 1
                fi
                sleep 3
              done

              # Correctness: the baseline must NOT have restarted the workload.
              before=$(kubectl -n $NS get {wlkind} {wlname} -o jsonpath='{RA_JSONPATH}' 2>/dev/null || true)
              if [ -n "$before" ]; then
                echo "baseline unexpectedly restarted {wlname}" >&2
                kubectl -n $NS get {wlkind} {wlname} -o yaml >&2 || true
                exit 1
              fi

              # 2. Mutate the referenced dependency (a relevant change).
              kubectl -n $NS {patch}

              # 3. The change MUST actually restart the workload: KICK stamps the
              #    pod template with kubectl.kubernetes.io/restartedAt.
              while true; do
                after=$(kubectl -n $NS get {wlkind} {wlname} -o jsonpath='{RA_JSONPATH}' 2>/dev/null || true)
                if [ -n "$after" ] && [ "$after" != "$before" ]; then
                  break
                fi
                if [ "$(date +%s)" -ge "$deadline" ]; then
                  echo "dependency change did not restart {wlname}" >&2
                  kubectl -n $NS get {wlkind} {wlname} -o yaml >&2 || true
                  kubectl -n $NS get kickrequest {kr} -o yaml >&2 || true
                  exit 1
                fi
                sleep 3
              done
"""


def main():
    for scn, (kr, wlkind, wlname, patch) in SCENARIOS.items():
        d = glob.glob(f"test/e2e/scenarios/KICK-E2E-{scn}-*/")[0]
        ns = f"kick-e2e-{scn}"
        tf = os.path.join(d, "chainsaw-test.yaml")
        with open(tf) as fh:
            old = fh.read()
        m = re.search(r"^metadata:\n  name: (.+)$", old, re.M)
        name = m.group(1)
        new = (
            "apiVersion: chainsaw.kyverno.io/v1alpha1\n"
            "kind: Test\n"
            "metadata:\n"
            f"  name: {name}\n"
            "spec:\n"
            "  steps:\n"
            "    - name: create-fixtures\n"
            "      try:\n"
            "        - apply:\n"
            "            file: resources.yaml\n"
            + assert_step(ns, kr, wlkind, wlname, patch) +
            "    - name: cleanup\n"
            "      try:\n"
            "        - delete:\n"
            "            ref:\n"
            "              apiVersion: v1\n"
            "              kind: Namespace\n"
            f"              name: {ns}\n"
        )
        with open(tf, "w") as fh:
            fh.write(new)
        print(f"wrote {tf}  (KR={kr}, patch={patch.split()[1]})")


if __name__ == "__main__":
    main()
