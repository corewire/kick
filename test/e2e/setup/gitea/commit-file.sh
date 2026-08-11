#!/usr/bin/env bash
# Commit a single manifest into a scenario's path, simulating an operator
# editing one file in git.
#
# Usage: commit-file.sh <repo> <repo-path> <local-file> [message]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=test/e2e/setup/gitea/lib.sh
source "${SCRIPT_DIR}/lib.sh"

gitea_commit_file "$@"
