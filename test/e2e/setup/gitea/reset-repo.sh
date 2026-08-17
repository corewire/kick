#!/usr/bin/env bash
# Drop a scenario's repository so the next seed starts from an empty history.
#
# Usage: reset-repo.sh <repo>
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=test/e2e/setup/gitea/lib.sh
source "${SCRIPT_DIR}/lib.sh"

gitea_delete_repo "$1"
