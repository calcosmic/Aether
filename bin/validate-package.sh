#!/bin/bash
# validate-package.sh — Pre-packaging validation for aether-colony npm package
#
# Purpose: Replaces sync-to-runtime.sh (removed in v4.0). Instead of copying files
#          to a staging directory, this script validates that .aether/ is ready for
#          direct packaging: required files exist, private directories are excluded.
#
# Usage: bash bin/validate-package.sh [--dry-run]
#   --dry-run  Run npm pack --dry-run to show what would be published
#
# This script is safe to run multiple times (idempotent).

set -euo pipefail

# Resolve paths relative to script location
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AETHER_DIR="$(cd "$SCRIPT_DIR/../.aether" && pwd)"

# --dry-run mode: delegate to npm pack and exit
if [ "${1:-}" = "--dry-run" ]; then
  cd "$REPO_ROOT"
  npm pack --dry-run 2>&1
  exit 0
fi

# Required files that must exist in .aether/ before packaging
REQUIRED_FILES=(
  "aether-utils.sh"
  "workers.md"
  "CONTEXT.md"
  "model-profiles.yaml"
  "docs/README.md"
  "utils/atomic-write.sh"
  "utils/error-handler.sh"
  "utils/file-lock.sh"
  "templates/QUEEN.md.template"
  "templates/colony-state.template.json"
  "templates/constraints.template.json"
  "templates/colony-state-reset.jq.template"
  "templates/crowned-anthill.template.md"
  "templates/handoff.template.md"
  "templates/handoff-build-error.template.md"
  "templates/handoff-build-success.template.md"
  "rules/aether-colony.md"
)

# Verify required files exist
for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$AETHER_DIR/$file" ]; then
    echo "ERROR: Required file missing from .aether/: $file" >&2
    echo "  Run this from the Aether repo root after editing .aether/ files." >&2
    exit 1
  fi
done

# Private directories that must never be published
PRIVATE_DIRS=(
  "data"
  "dreams"
  "oracle"
  "checkpoints"
  "locks"
  "temp"
  "archive"
  "chambers"
)

# .aether/.npmignore is the effective ignore file for the .aether/ subdirectory.
# npm-packlist reads subdirectory .npmignore files when walking included directories.
AETHER_NPMIGNORE="$AETHER_DIR/.npmignore"

# Verify each private directory is excluded in .aether/.npmignore (hard block)
for dir in "${PRIVATE_DIRS[@]}"; do
  if [ -d "$AETHER_DIR/$dir" ]; then
    if ! grep -qF "$dir/" "$AETHER_NPMIGNORE" 2>/dev/null; then
      echo "ERROR: Private directory .aether/$dir/ exists but is NOT excluded in .aether/.npmignore" >&2
      echo "  Add '$dir/' to .aether/.npmignore to prevent publishing private data." >&2
      exit 1
    fi
  fi
done

echo "Package validation passed."
