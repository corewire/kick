#!/usr/bin/env bash
# Seed a scenario's manifests into the shared `apps` repository.
#
# Usage: seed.sh <repo-path> <local-dir> [message]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=test/e2e/setup/gitea/lib.sh
source "${SCRIPT_DIR}/lib.sh"

gitea_seed_dir "$@"
