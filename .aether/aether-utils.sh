#!/bin/bash
# Aether Colony Utility Layer
# Single entry point for deterministic colony operations
#
# Usage: bash .aether/aether-utils.sh <subcommand> [args...]
#
# All subcommands output JSON to stdout.
# Non-zero exit on error with JSON error message to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd 2>/dev/null || echo "$SCRIPT_DIR")"
DATA_DIR="$SCRIPT_DIR/data"

# Initialize lock state before sourcing (file-lock.sh trap needs these)
LOCK_ACQUIRED=${LOCK_ACQUIRED:-false}
CURRENT_LOCK=${CURRENT_LOCK:-""}

# Source shared infrastructure
source "$SCRIPT_DIR/utils/file-lock.sh"
source "$SCRIPT_DIR/utils/atomic-write.sh"

# --- JSON output helpers ---
# Success: JSON to stdout, exit 0
json_ok() { printf '{"ok":true,"result":%s}\n' "$1"; }

# Error: JSON to stderr, exit 1
json_err() { printf '{"ok":false,"error":"%s"}\n' "$1" >&2; exit 1; }

# --- Subcommand dispatch ---
cmd="${1:-help}"
shift 2>/dev/null || true

case "$cmd" in
  help)
    cat <<'EOF'
{"ok":true,"commands":["help","version"],"description":"Aether Colony Utility Layer — deterministic ops for the ant colony"}
EOF
    ;;
  version)
    json_ok '"0.1.0"'
    ;;
  *)
    json_err "Unknown command: $cmd"
    ;;
esac
