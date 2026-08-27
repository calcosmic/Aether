#!/usr/bin/env bash
# version-sync.sh — rewrite every "current version" declaration in the docs
# from the single source of truth, .aether/version.json.
#
# Six documents each carried their own hand-maintained version string, and they
# drifted to three different values while the runtime was on a fourth. This
# makes the docs derived rather than remembered.
#
# Historical release notes are deliberately NOT rewritten: a changelog entry
# about v1.0.41 should keep saying v1.0.41.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' .aether/version.json | head -1)"
if [ -z "$VERSION" ]; then
  echo "version-sync: could not read version from .aether/version.json" >&2
  exit 1
fi
TODAY="$(date -u +%Y-%m-%d)"
echo "Syncing docs to v${VERSION}"

# sync_line <file> <sed-expression>
sync_line() {
  local file="$1" expr="$2"
  [ -f "$file" ] || return 0
  perl -pi -e "$expr" "$file"
}

# README badge and npm bootstrap sentence.
sync_line README.md 's{(badge/colony-v)\d+\.\d+\.\d+}{${1}'"$VERSION"'}g'
sync_line README.md 's{`aether-colony\@\d+\.\d+\.\d+` bootstraps Aether `\d+\.\d+\.\d+`}{`aether-colony\@'"$VERSION"'` bootstraps Aether `'"$VERSION"'`}g'

# CLAUDE.md / AGENTS.md carry a header line, a Quick Reference table row, and a footer.
for doc in CLAUDE.md AGENTS.md; do
  sync_line "$doc" 's{^(> \*\*Current Version:\*\* v)\d+\.\d+\.\d+}{${1}'"$VERSION"'}'
  sync_line "$doc" 's{^(\| Version \| v)\d+\.\d+\.\d+}{${1}'"$VERSION"'}'
  sync_line "$doc" 's{^\*Updated for Aether v\d+\.\d+\.\d+ [-—\x{2014}]+ \d{4}-\d{2}-\d{2}\*}{*Updated for Aether v'"$VERSION"' — '"$TODAY"'*}'
done

# Codex and OpenCode platform docs carry a footer line.
sync_line .codex/CODEX.md 's{^\*Updated for Aether v\d+\.\d+\.\d+ [-—\x{2014}]+ \d{4}-\d{2}-\d{2}\*}{*Updated for Aether v'"$VERSION"' — '"$TODAY"'*}'
sync_line .opencode/OPENCODE.md 's{^\*Updated for Aether v\d+\.\d+\.\d+ [-—\x{2014}]+ \d{4}-\d{2}-\d{2}\*}{*Updated for Aether v'"$VERSION"' — '"$TODAY"'*}'

# OpenCode had no version line at all; append one if still missing.
if [ -f .opencode/OPENCODE.md ] && ! grep -q "Updated for Aether v" .opencode/OPENCODE.md; then
  printf '\n*Updated for Aether v%s — %s*\n' "$VERSION" "$TODAY" >> .opencode/OPENCODE.md
fi

# npm/package.json is the version the packed release candidate ships, and
# TestPackedNPMReleaseCandidateContract fails when it disagrees with
# .aether/version.json. It was NOT synced here, so every release depended on
# somebody remembering to edit it by hand -- and on 2026-08-27 the v1.0.64 bump
# forgot it and the gate caught the mismatch. Derived, not remembered.
sync_line npm/package.json 's{^(  "version": ")\d+\.\d+\.\d+(")}{${1}'"$VERSION"'${2}}'

echo "Docs synced to v${VERSION}"
