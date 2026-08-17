#!/usr/bin/env bash
# gsd-install.sh — file-copy install of GSD (get-shit-done) into an isolated HOME.
#
# Sourceable library (not directly executable). GSD ships no installer: it is
# a plain file copy of an existing installation on the operator's real home
# directory. This lib copies that installation into a benchmark lane's
# isolated HOME so the GSD lane can run hermetically, the same way the Aether
# lanes get a fresh `aether install --package-dir`.
#
# Exposes:
#   gsd_install_into_home SOURCE_HOME   — copy GSD from SOURCE_HOME into $HOME
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

# gsd_install_into_home SOURCE_HOME
#
# Copies get-shit-done, its gsd-* agents, and its gsd-* skills from
# SOURCE_HOME (the operator's real home, captured by the caller before any
# HOME override) into the current (already-isolated) $HOME. Resolves the
# gsd-sdk executable on PATH and prepends its directory for this lane, since
# gsd-sdk is a node script on PATH rather than something inside ~/.claude.
gsd_install_into_home() {
  local source_home="${1:-${REAL_HOME:-}}"

  [ -n "$source_home" ] || fail "gsd_install_into_home: SOURCE_HOME not provided and REAL_HOME not set"
  [ -f "$source_home/.claude/get-shit-done/VERSION" ] \
    || fail "gsd_install_into_home: $source_home/.claude/get-shit-done/VERSION not found — GSD must already be installed in the operator's real home directory"

  step "copying get-shit-done tree into isolated HOME ($HOME)"
  mkdir -p "$HOME/.claude"
  cp -R "$source_home/.claude/get-shit-done" "$HOME/.claude/get-shit-done"

  step "copying gsd-* agents into isolated HOME"
  mkdir -p "$HOME/.claude/agents"
  local agent_count=0
  if compgen -G "$source_home/.claude/agents/gsd-*.md" > /dev/null; then
    cp "$source_home"/.claude/agents/gsd-*.md "$HOME/.claude/agents/"
    agent_count="$(ls "$HOME"/.claude/agents/gsd-*.md 2>/dev/null | wc -l | tr -d ' ')"
  fi

  step "copying gsd-* skills into isolated HOME"
  mkdir -p "$HOME/.claude/skills"
  local skill_count=0
  if compgen -G "$source_home/.claude/skills/gsd-*" > /dev/null; then
    cp -R "$source_home"/.claude/skills/gsd-* "$HOME/.claude/skills/"
    skill_count="$(ls -d "$HOME"/.claude/skills/gsd-* 2>/dev/null | wc -l | tr -d ' ')"
  fi

  [ "$agent_count" -gt 0 ] || fail "gsd_install_into_home: 0 gsd-* agents copied from $source_home/.claude/agents/"
  [ "$skill_count" -gt 0 ] || fail "gsd_install_into_home: 0 gsd-* skills copied from $source_home/.claude/skills/"

  # gsd-sdk is a node script on PATH, not inside ~/.claude — make it
  # reachable for this lane rather than copying it.
  local gsd_sdk_path
  gsd_sdk_path="$(command -v gsd-sdk || true)"
  if [ -n "$gsd_sdk_path" ]; then
    local gsd_sdk_dir
    gsd_sdk_dir="$(cd "$(dirname "$gsd_sdk_path")" && pwd)"
    export PATH="$gsd_sdk_dir:$PATH"
  else
    fail "gsd_install_into_home: gsd-sdk not found on PATH before HOME override"
  fi

  local version
  version="$(cat "$HOME/.claude/get-shit-done/VERSION")"
  echo "gsd-install version=$version agents=$agent_count skills=$skill_count"
}
