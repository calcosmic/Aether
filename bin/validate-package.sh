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
  "docs/README.md"
  "utils/atomic-write.sh"
  "utils/error-handler.sh"
  # Domain modules extracted in Phase 13
  "utils/pheromone.sh"
  "utils/learning.sh"
  "utils/state-api.sh"
  "utils/flag.sh"
  "utils/session.sh"
  "utils/spawn.sh"
  "utils/queen.sh"
  "utils/suggest.sh"
  "utils/swarm.sh"
  # Skills system (v2.0)
  "utils/skills.sh"
  # Hive and midden modules
  "utils/hive.sh"
  "utils/midden.sh"
  # Skills manifests
  "skills/.index.json"
  "skills/colony/.manifest.json"
  "skills/domain/.manifest.json"
  "utils/file-lock.sh"
  "utils/chamber-compare.sh"
  "utils/chamber-utils.sh"
  "utils/clash-detect.sh"
  "utils/colorize-log.sh"
  "utils/council.sh"
  "utils/emoji-audit.sh"
  "utils/immune.sh"
  "utils/merge-driver-lockfile.sh"
  "utils/scan.sh"
  "utils/semantic-cli.sh"
  "utils/spawn-tree.sh"
  "utils/state-loader.sh"
  "utils/swarm-display.sh"
  "utils/watch-spawn-tree.sh"
  "utils/worktree.sh"
  "utils/xml-compose.sh"
  "utils/xml-convert.sh"
  "utils/xml-core.sh"
  "utils/xml-query.sh"
  "utils/xml-utils.sh"
  "utils/hooks/clash-pre-tool-use.js"
  "utils/oracle/oracle.sh"
  "utils/queen-to-md.xsl"
  "templates/QUEEN.md.template"
  "templates/colony-state.template.json"
  "templates/constraints.template.json"
  "templates/colony-state-reset.jq.template"
  "templates/crowned-anthill.template.md"
  "templates/handoff.template.md"
  "templates/handoff-build-error.template.md"
  "templates/handoff-build-success.template.md"
  "templates/session.template.json"
  "templates/pheromones.template.json"
  "templates/midden.template.json"
  "templates/learning-observations.template.json"
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

# Content-aware checks: verify specific files are excluded from the package
cd "$REPO_ROOT"
PACK_LIST=$(npm pack --dry-run 2>&1)

# Check 1: QUEEN.md must not be in the package (template and docs versions are fine)
if echo "$PACK_LIST" | grep -qE '\.aether/QUEEN\.md$'; then
  echo "ERROR: .aether/QUEEN.md would be included in package" >&2
  echo "  QUEEN.md should be created from template during install, not shipped." >&2
  echo "  Add 'QUEEN.md' to .aether/.npmignore" >&2
  exit 1
fi

# Check 2: No temp files in .aether/
if echo "$PACK_LIST" | grep -qE '\.aether/.*\.tmp'; then
  echo "ERROR: Temp files would be included in package:" >&2
  echo "$PACK_LIST" | grep -E '\.aether/.*\.tmp' >&2
  echo "  Delete temp files or add '*.tmp*' to .aether/.npmignore" >&2
  exit 1
fi

# Check 3: CONTEXT.md must not be in the package
if echo "$PACK_LIST" | grep -qE '\.aether/CONTEXT\.md'; then
  echo "ERROR: .aether/CONTEXT.md would be included in package" >&2
  echo "  CONTEXT.md is per-colony. Add 'CONTEXT.md' to .aether/.npmignore" >&2
  exit 1
fi

# Check 4: No .aether/data/ files in the package (covers pheromones.json, constraints.json, midden/, etc.)
if echo "$PACK_LIST" | grep -qE '\.aether/data/'; then
  echo "ERROR: .aether/data/ files would be included in package:" >&2
  echo "$PACK_LIST" | grep -E '\.aether/data/' >&2
  echo "  data/ is local-only. Ensure 'data/' is in .aether/.npmignore" >&2
  exit 1
fi

# Check 5: CROWNED-ANTHILL.md must not be in the package
if echo "$PACK_LIST" | grep -qE '\.aether/CROWNED-ANTHILL\.md'; then
  echo "ERROR: .aether/CROWNED-ANTHILL.md would be included in package" >&2
  echo "  CROWNED-ANTHILL.md is a sealed colony record. Add it to .aether/.npmignore" >&2
  exit 1
fi

# Check 6: No exchange XML data files in the package
if echo "$PACK_LIST" | grep -qE '\.aether/exchange/.*\.xml'; then
  echo "ERROR: Exchange XML data files would be included in package:" >&2
  echo "$PACK_LIST" | grep -E '\.aether/exchange/.*\.xml' >&2
  echo "  Add 'exchange/*.xml' to .aether/.npmignore" >&2
  exit 1
fi

# Check 7: Exchange modules must be present in distribution
# Per D-03: exchange/ modules must be included in npm package
for module in pheromone-xml.sh wisdom-xml.sh registry-xml.sh; do
  if [ ! -f "$AETHER_DIR/exchange/$module" ]; then
    echo "ERROR: Required exchange module missing from .aether/exchange/: $module" >&2
    echo "  Exchange modules are required for XML export/import functionality." >&2
    echo "  Run this from the Aether repo root after editing .aether/ files." >&2
    exit 1
  fi
done

echo "Package validation passed (files + content checks + exchange module check)."
