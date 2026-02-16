#!/bin/bash
# sync-to-runtime.sh — Copies allowlisted system files from .aether/ to runtime/
#
# Purpose: In the Aether repo, .aether/ is the source of truth for system files.
#          This script syncs those files INTO runtime/ (the npm staging directory)
#          so that `npm install -g .` packages the latest versions.
#
# Usage: bash bin/sync-to-runtime.sh [--reverse]
#   --reverse  Copy FROM runtime/ TO .aether/ (one-time seed operation)
#
# This script is safe to run multiple times (idempotent).
# It only copies allowlisted files — it never deletes extras in runtime/.

set -e

# Resolve repo root (one level up from bin/)
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AETHER_DIR="$REPO_ROOT/.aether"
RUNTIME_DIR="$REPO_ROOT/runtime"

# Allowlist: these are the system files that sync between .aether/ and runtime/
# Must match the SYSTEM_FILES array in bin/lib/update-transaction.js
SYSTEM_FILES=(
  "aether-utils.sh"
  "coding-standards.md"
  "debugging.md"
  "DISCIPLINES.md"
  "learning.md"
  "planning.md"
  "QUEEN_ANT_ARCHITECTURE.md"
  "tdd.md"
  "verification-loop.md"
  "verification.md"
  "workers.md"
  "workers-new-castes.md"
  "docs/biological-reference.md"
  "docs/command-sync.md"
  "docs/constraints.md"
  "docs/namespace.md"
  "docs/pathogen-schema-example.json"
  "docs/pathogen-schema.md"
  "docs/PHEROMONE-INJECTION.md"
  "docs/PHEROMONE-INTEGRATION.md"
  "docs/PHEROMONE-SYSTEM-DESIGN.md"
  "docs/pheromones.md"
  "docs/progressive-disclosure.md"
  "docs/README.md"
  "docs/VISUAL-OUTPUT-SPEC.md"
  "docs/known-issues.md"
  "docs/implementation-learnings.md"
  "docs/codebase-review.md"
  "docs/planning-discipline.md"
  "utils/atomic-write.sh"
  "utils/chamber-compare.sh"
  "utils/chamber-utils.sh"
  "utils/colorize-log.sh"
  "utils/error-handler.sh"
  "utils/file-lock.sh"
  "utils/spawn-tree.sh"
  "utils/spawn-with-model.sh"
  "utils/state-loader.sh"
  "utils/swarm-display.sh"
  "utils/watch-spawn-tree.sh"
  "templates/QUEEN.md.template"
)

# Determine direction
REVERSE=false
if [ "${1:-}" = "--reverse" ]; then
  REVERSE=true
fi

if [ "$REVERSE" = true ]; then
  SRC_DIR="$RUNTIME_DIR"
  DST_DIR="$AETHER_DIR"
  LABEL="runtime/ -> .aether/ (seeding)"
else
  SRC_DIR="$AETHER_DIR"
  DST_DIR="$RUNTIME_DIR"
  LABEL=".aether/ -> runtime/ (staging)"
fi

# Check source directory exists
if [ ! -d "$SRC_DIR" ]; then
  # Silently exit if source doesn't exist (e.g. installed from npm, no .aether/)
  exit 0
fi

copied=0
skipped=0

for file in "${SYSTEM_FILES[@]}"; do
  src="$SRC_DIR/$file"
  dst="$DST_DIR/$file"

  # Skip if source file doesn't exist
  if [ ! -f "$src" ]; then
    continue
  fi

  # Create destination directory if needed
  dst_dir="$(dirname "$dst")"
  mkdir -p "$dst_dir"

  # Skip if files are identical (compare hashes)
  if [ -f "$dst" ]; then
    src_hash=$(shasum -a 256 "$src" 2>/dev/null | cut -d' ' -f1)
    dst_hash=$(shasum -a 256 "$dst" 2>/dev/null | cut -d' ' -f1)
    if [ "$src_hash" = "$dst_hash" ]; then
      skipped=$((skipped + 1))
      continue
    fi
  fi

  cp "$src" "$dst"
  copied=$((copied + 1))
done

# Only print output if not running in quiet mode (npm postinstall suppresses)
if [ -t 1 ] || [ "${VERBOSE:-}" = "1" ]; then
  echo "Sync ($LABEL): $copied copied, $skipped unchanged"
fi
