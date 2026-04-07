# =============================================================================
# DEPRECATED — This script has been superseded by the Go binary (aether CLI).
# All functionality is now available via: aether <subcommand>
# Do NOT modify this file — it is retained for reference only.
# See: cmd/ (Go source) | Run: aether --help
# =============================================================================
#
#!/usr/bin/env bash
# Merge Driver: npm lockfile auto-merge
#
# Resolves package-lock.json merge conflicts by keeping "ours" (the branch
# being merged into). Lockfiles are deterministic outputs of package.json,
# so the correct resolution is always to regenerate from the target branch's
# package.json, which effectively means keeping "ours".
#
# Usage: merge-driver-lockfile.sh %O %A %B
#   %O = ancestor (base version)
#   %A = ours (current branch, the version we want to keep)
#   %B = theirs (incoming branch)
#
# Git merge driver contract:
#   Exit 0 = conflict resolved (merge continues)
#   Exit non-zero = conflict unresolved
#
# This driver is configured via:
#   git config merge.lockfile.name "npm lockfile auto-merge"
#   git config merge.lockfile.driver "bash .aether/utils/merge-driver-lockfile.sh %O %A %B"

set -euo pipefail

ANCESTOR="${1:-}"
OURS="${2:-}"
THEIRS="${3:-}"

# Strategy: keep "ours" unchanged.
# The "ours" file already contains the correct content on disk.
# We do nothing -- git considers the file resolved when we exit 0.
#
# Log for debugging (to stderr so it does not pollute merge output)
echo "[aether] merge-driver: resolved package-lock.json conflict (kept ours)" >&2

exit 0
