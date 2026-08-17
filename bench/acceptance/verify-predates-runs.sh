#!/usr/bin/env bash
# verify-predates-runs.sh — proves from git history alone that every
# acceptance script in bench/acceptance/*.sh was both first committed and
# last modified before any benchmark result directory existed.
#
# This is the executable form of Phase 186's third success criterion —
# without this script, "acceptance predates results" is a claim in a
# document, not something a stranger can check.
#
# Usage: bench/acceptance/verify-predates-runs.sh
#   (no arguments — this script inspects the repository it runs inside)
#
# Exit 0 means: every acceptance script's introducing commit is strictly
#   earlier than every result directory's introducing commit, AND no
#   acceptance script was modified after the first result directory landed.
#   If no result directory exists yet, this exits 0 vacuously and says so —
#   a vacuous pass is never presented as a full check.
# Exit 1 means: an acceptance script is untracked/uncommitted (no provable
#   date at all — worse than a late one), or the ordering was violated, or
#   an acceptance script was modified after results started landing.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

fail() {
  echo "VERIFY-PREDATES-RUNS FAIL: $*" >&2
  exit 1
}

# human_date EPOCH_SECONDS — renders an epoch timestamp in a readable form
# for failure messages, never raw epoch seconds.
human_date() {
  local epoch="$1"
  if date -r "$epoch" '+%Y-%m-%d %H:%M:%S %Z' >/dev/null 2>&1; then
    date -r "$epoch" '+%Y-%m-%d %H:%M:%S %Z'
  else
    date -d "@$epoch" '+%Y-%m-%d %H:%M:%S %Z'
  fi
}

shopt -s nullglob
ACCEPTANCE_FILES=(bench/acceptance/*.sh)
shopt -u nullglob

[ "${#ACCEPTANCE_FILES[@]}" -gt 0 ] || fail "no acceptance scripts found under bench/acceptance/*.sh — nothing to verify"

declare -a SCRIPT_NAMES=()
declare -a SCRIPT_INTRODUCED_TS=()
declare -a SCRIPT_LAST_MODIFIED_TS=()

for f in "${ACCEPTANCE_FILES[@]}"; do
  # An untracked or never-committed script has no provable date at all,
  # which is worse than a late one — refuse outright, naming the file.
  if ! git ls-files --error-unmatch "$f" >/dev/null 2>&1; then
    fail "acceptance script is untracked (never committed): $f — an uncommitted script has no provable date"
  fi

  INTRO_LINE="$(git log --diff-filter=A --follow --format='%H|%ct' -- "$f" | tail -1)"
  [ -n "$INTRO_LINE" ] || fail "acceptance script has no introducing (add) commit in git history: $f — an uncommitted script has no provable date"
  INTRO_TS="${INTRO_LINE##*|}"

  LAST_MOD_TS="$(git log --format='%ct' -- "$f" | head -1)"
  [ -n "$LAST_MOD_TS" ] || fail "acceptance script has no commit history at all: $f"

  # Working-tree changes not yet committed also have no provable date.
  if ! git diff --quiet -- "$f" 2>/dev/null || ! git diff --cached --quiet -- "$f" 2>/dev/null; then
    fail "acceptance script has uncommitted changes: $f — an uncommitted change has no provable date"
  fi

  SCRIPT_NAMES+=("$f")
  SCRIPT_INTRODUCED_TS+=("$INTRO_TS")
  SCRIPT_LAST_MODIFIED_TS+=("$LAST_MOD_TS")
done

SCRIPT_COUNT="${#SCRIPT_NAMES[@]}"

shopt -s nullglob
RESULT_DIRS=(bench/results/*/)
shopt -u nullglob

if [ "${#RESULT_DIRS[@]}" -eq 0 ]; then
  echo "VERIFY-PREDATES-RUNS: no run results exist yet (bench/results/*/ is empty or absent) — the ordering holds VACUOUSLY. Checked $SCRIPT_COUNT acceptance script(s); this is not a full check until at least one result directory exists."
  exit 0
fi

declare -a RESULT_DIR_NAMES=()
declare -a RESULT_INTRODUCED_TS=()

for d in "${RESULT_DIRS[@]}"; do
  d="${d%/}"
  INTRO_LINE="$(git log --diff-filter=A --follow --format='%H|%ct' -- "$d" | tail -1)"
  [ -n "$INTRO_LINE" ] || fail "result directory has no introducing (add) commit in git history: $d"
  INTRO_TS="${INTRO_LINE##*|}"
  RESULT_DIR_NAMES+=("$d")
  RESULT_INTRODUCED_TS+=("$INTRO_TS")
done

# Earliest result directory introduction timestamp.
EARLIEST_RESULT_TS="${RESULT_INTRODUCED_TS[0]}"
EARLIEST_RESULT_DIR="${RESULT_DIR_NAMES[0]}"
for i in "${!RESULT_INTRODUCED_TS[@]}"; do
  if [ "${RESULT_INTRODUCED_TS[$i]}" -lt "$EARLIEST_RESULT_TS" ]; then
    EARLIEST_RESULT_TS="${RESULT_INTRODUCED_TS[$i]}"
    EARLIEST_RESULT_DIR="${RESULT_DIR_NAMES[$i]}"
  fi
done

VIOLATIONS=0
for i in "${!SCRIPT_NAMES[@]}"; do
  NAME="${SCRIPT_NAMES[$i]}"
  INTRO_TS="${SCRIPT_INTRODUCED_TS[$i]}"
  LAST_MOD_TS="${SCRIPT_LAST_MODIFIED_TS[$i]}"

  if [ "$INTRO_TS" -ge "$EARLIEST_RESULT_TS" ]; then
    echo "VERIFY-PREDATES-RUNS FAIL: $NAME was first committed $(human_date "$INTRO_TS"), which is not strictly before $EARLIEST_RESULT_DIR's introducing commit at $(human_date "$EARLIEST_RESULT_TS")" >&2
    VIOLATIONS=$((VIOLATIONS + 1))
  fi

  if [ "$LAST_MOD_TS" -ge "$EARLIEST_RESULT_TS" ]; then
    echo "VERIFY-PREDATES-RUNS FAIL: $NAME was last modified $(human_date "$LAST_MOD_TS"), which is not strictly before $EARLIEST_RESULT_DIR's introducing commit at $(human_date "$EARLIEST_RESULT_TS") — a script rewritten after seeing results is the same failure as one written after them" >&2
    VIOLATIONS=$((VIOLATIONS + 1))
  fi
done

if [ "$VIOLATIONS" -gt 0 ]; then
  exit 1
fi

echo "VERIFY-PREDATES-RUNS PASS: all $SCRIPT_COUNT acceptance script(s) were both created and last modified strictly before the earliest result directory ($EARLIEST_RESULT_DIR, introduced $(human_date "$EARLIEST_RESULT_TS"))"
exit 0
