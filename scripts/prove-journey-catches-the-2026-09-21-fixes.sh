#!/usr/bin/env bash
# prove-journey-catches-the-2026-09-21-fixes.sh — 207-05-PLAN.md Task 2 (UED-09).
#
# Proves the journey (207-01..04) is worth running: for each of the five
# landed 2026-09-21 fixes cmd/testdata/journey/fix-reverts.json declares,
# this script reverts that fix's own exact, minimal, single-occurrence text
# swap inside a THROWAWAY working copy, drives only the journey step that
# fix protects (the AETHER_JOURNEY_STEP single-step mode 207-04 built), and
# asserts the step FAILS. A step that still passes with the fix reverted
# means the journey does not catch that blocker, and is reported as exactly
# that failure -- named, not softened.
#
# ISOLATION RULE, followed throughout this script: nothing here writes to
# the real $HOME/.claude/ or the real ~/.aether/ hub. Every hook and menu
# command the driven step exercises comes from the throwaway copy's own
# freshly-built practice project, isolated via AETHER_HUB_DIR. HOME itself
# is deliberately left ALONE (never overridden) -- the same rule
# scripts/proof-screens-reach-the-owner.sh and
# scripts/build-messy-practice-project.sh already follow.
#
# WORKING-COPY RULE, followed throughout this script: the owner's own
# checkout at $ROOT is NEVER switched, shelved, reset, cleaned, or modified.
# Every revert is applied inside an ADDITIONAL working copy this script
# creates with `git worktree add --detach` from $ROOT's own HEAD, under the
# scratch directory, and removed again before this script moves on to the
# next entry. Other sessions share $ROOT; this project has already lost
# work to a script that forgot that (CLAUDE.md's "Concurrent sessions in
# one repo" note). This script never runs `git checkout`, `git stash`,
# `git reset`, `git restore`, or `git clean` anywhere (WR-03,
# 207-REVIEW.md: `git clean -fd` would silently delete untracked files in
# $ROOT and is at least as dangerous as the other four), and never creates a
# bare clone.
#
# Each throwaway working copy's own directory name carries the project's
# name ("Aether") -- a throwaway checkout whose path lacks it has tripped a
# path-name-sensitive root-resolution check elsewhere in this repo before
# (CLAUDE.md's "gate worktree path needs Aether" note); naming it
# defensively here costs nothing.
#
# A run that applies no reverts at all must never report success -- that is
# exactly the failure mode where a proof quietly proves nothing, which this
# project has shipped before. An empty or unreadable table refuses to start,
# naming the reason.
#
# AETHER_FIX_REVERT_ONLY=<id> restricts this run to one declared entry (the
# money-aware verification path this plan's own <verify> block uses).
# AETHER_FIX_REVERT_STEP_TIMEOUT=<seconds> overrides the per-entry journey
# step's wall-clock cap (default 900s -- generously above every real
# measured single-step cost in 207-04-SUMMARY.md).
# AETHER_FIX_REVERT_TABLE=<path> overrides which table this script reads
# (test-only; the real run always reads the committed table at $ROOT).
#
# Exits non-zero on the first entry whose revert the journey did not catch,
# after reporting every entry this run touched. Requires: go, git, jq,
# claude.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

fail() { echo "PROOF FAIL: $*" >&2; exit 1; }
step() { echo; echo "==> $*"; }

for tool in go git jq claude; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool missing: $tool"
done

SCRATCH_BASE="${TMPDIR:-/tmp}"
WORK="$(mktemp -d "${SCRATCH_BASE%/}/aether-fix-revert-proof.XXXXXX")"
CURRENT_WORKTREE=""
cleanup() {
  if [ -n "$CURRENT_WORKTREE" ] && [ -d "$CURRENT_WORKTREE" ]; then
    git -C "$ROOT" worktree remove --force "$CURRENT_WORKTREE" >/dev/null 2>&1 || rm -rf "$CURRENT_WORKTREE"
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

# Isolate the hub only. HOME is intentionally untouched -- see the
# ISOLATION RULE above.
export AETHER_HUB_DIR="$WORK/hub"
mkdir -p "$AETHER_HUB_DIR"

TABLE="${AETHER_FIX_REVERT_TABLE:-$ROOT/cmd/testdata/journey/fix-reverts.json}"
[ -f "$TABLE" ] || fail "no fix-revert table at $TABLE"
jq empty "$TABLE" 2>/dev/null || fail "fix-revert table is not valid JSON: $TABLE"
ENTRY_COUNT=$(jq '.reverts | length' "$TABLE")
[ "$ENTRY_COUNT" -gt 0 ] || fail "the fix-revert table at $TABLE is empty -- a run that applies no reverts must never report success"

ONLY="${AETHER_FIX_REVERT_ONLY:-}"
STEP_TIMEOUT_SECS="${AETHER_FIX_REVERT_STEP_TIMEOUT:-900}"

REPORT_ROWS_FILE="$WORK/report-rows.jsonl"
: >"$REPORT_ROWS_FILE"

APPLIED_COUNT=0
FAILURES=0

IDS=$(jq -r '.reverts[].id' "$TABLE")

for ID in $IDS; do
  if [ -n "$ONLY" ] && [ "$ID" != "$ONLY" ]; then
    continue
  fi

  ENTRY=$(jq -c --arg id "$ID" '.reverts[] | select(.id == $id)' "$TABLE")
  FILE=$(printf '%s' "$ENTRY" | jq -r '.file')
  SYMBOL=$(printf '%s' "$ENTRY" | jq -r '.symbol')
  COMMIT=$(printf '%s' "$ENTRY" | jq -r '.commit')
  BLOCKER=$(printf '%s' "$ENTRY" | jq -r '.blocker')
  JSTEP=$(printf '%s' "$ENTRY" | jq -r '.journey_step')
  EXPECTED=$(printf '%s' "$ENTRY" | jq -r '.expected_failure')
  FIND=$(printf '%s' "$ENTRY" | jq -r '.mutation.find')
  REPLACE=$(printf '%s' "$ENTRY" | jq -r '.mutation.replace')

  step "[$ID] $BLOCKER -- reverting $SYMBOL in $FILE ($COMMIT), journey step \"$JSTEP\""

  WT="$WORK/Aether-fix-revert-$ID"
  CURRENT_WORKTREE="$WT"
  git -C "$ROOT" worktree add --detach -q "$WT" HEAD \
    || fail "[$ID] could not create a throwaway working copy with git worktree add"

  TARGET="$WT/$FILE"
  [ -f "$TARGET" ] || fail "[$ID] $FILE not found in the throwaway copy"

  OCCURRENCES=$(grep -cF -- "$FIND" "$TARGET" || true)
  [ "$OCCURRENCES" = "1" ] \
    || fail "[$ID] mutation.find occurs $OCCURRENCES time(s) in $TARGET, want exactly 1 -- the fix-revert table has rotted against the current source"

  APPLY_STATUS=0
  python3 - "$TARGET" "$FIND" "$REPLACE" <<'PYEOF' || APPLY_STATUS=$?
import sys

path, find, replace = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path) as f:
    content = f.read()
if content.count(find) != 1:
    sys.exit(1)
content = content.replace(find, replace, 1)
with open(path, "w") as f:
    f.write(content)
PYEOF
  [ "$APPLY_STATUS" -eq 0 ] || fail "[$ID] failed to apply the declared mutation to $TARGET"

  (cd "$WT" && go build ./cmd/aether) >"$WORK/$ID-build.log" 2>&1 \
    || fail "[$ID] the throwaway copy does not even build after reverting $SYMBOL -- see $WORK/$ID-build.log"

  step "[$ID] driving the journey's \"$JSTEP\" step against the reverted copy"

  LOG="$WORK/$ID.log"
  run_step() {
    (cd "$WT" && AETHER_JOURNEY_STEP="$JSTEP" go test -tags journey -run TestJourney -count=1 -timeout "${STEP_TIMEOUT_SECS}s" ./cmd -v) >"$LOG" 2>&1
  }

  set +e
  run_step
  STATUS=$?
  set -e

  RETRIED="no"
  if [ "$STATUS" -ne 0 ] && grep -qiE 'rate.?limit|overloaded|timed?.?out' "$LOG"; then
    echo "    [$ID] transient failure (exit $STATUS) -- retrying once, as instructed; not counted as a real failure unless it recurs"
    RETRIED="yes"
    set +e
    run_step
    STATUS=$?
    set -e
  fi

  if [ "$STATUS" -eq 0 ]; then
    CAUGHT="false"
    echo "    [$ID] FAIL: the journey's \"$JSTEP\" step PASSED with $SYMBOL reverted -- the journey does not catch this blocker"
    FAILURES=$((FAILURES + 1))
  else
    CAUGHT="true"
    echo "    [$ID] caught: the journey's \"$JSTEP\" step failed as expected ($EXPECTED)"
  fi

  jq -n \
    --arg id "$ID" --arg blocker "$BLOCKER" --arg commit "$COMMIT" --arg file "$FILE" \
    --arg symbol "$SYMBOL" --arg jstep "$JSTEP" --arg expected "$EXPECTED" \
    --argjson caught "$CAUGHT" --arg retried "$RETRIED" --argjson exit_status "$STATUS" \
    '{id: $id, blocker: $blocker, commit: $commit, file: $file, symbol: $symbol, journey_step: $jstep, expected_failure: $expected, caught: $caught, retried: $retried, exit_status: $exit_status}' \
    >>"$REPORT_ROWS_FILE"

  APPLIED_COUNT=$((APPLIED_COUNT + 1))

  git -C "$ROOT" worktree remove --force "$WT" >/dev/null 2>&1 || rm -rf "$WT"
  CURRENT_WORKTREE=""
done

[ "$APPLIED_COUNT" -gt 0 ] \
  || fail "no entries were applied (AETHER_FIX_REVERT_ONLY=$ONLY matched nothing in $TABLE) -- a run that applies no reverts must never report success"

TABLE_SCHEMA=$(jq -r '.schema_version' "$TABLE")
VERDICT="pass"
[ "$FAILURES" -eq 0 ] || VERDICT="fail"

REPORT="$WORK/journey-fix-reverts-report.json"
jq -s \
  --arg schema "journey-fix-reverts-report/v1" \
  --arg table_schema "$TABLE_SCHEMA" \
  --arg verdict "$VERDICT" \
  '{schema_version: $schema, table_schema_version: $table_schema, verdict: $verdict, results: .}' \
  "$REPORT_ROWS_FILE" >"$REPORT"

# Written beside the journey report -- inside the already-sanctioned
# .aether/data/worker-debug/ write-allowlist prefix (cmd/hook_cmds.go's
# sanctionedDataWritePrefixes), gitignored by this repo's own .aether/.gitignore.
REPORT_DEST="$ROOT/.aether/data/worker-debug/journey-fix-reverts-report.json"
mkdir -p "$(dirname "$REPORT_DEST")" 2>/dev/null && cp "$REPORT" "$REPORT_DEST" 2>/dev/null || true

step "summary"
echo "Five landed 2026-09-21 fixes protect the everyday lifecycle against real blockers a real day of use hit. Here is whether reverting each one, in a throwaway copy, made the journey fail where it should:"
jq -r '.results[] | "  - " + .blocker + ": " + (if .caught then "caught -- the journey failed at the \"" + .journey_step + "\" step, as it should" else "NOT CAUGHT -- the journey still passed with this fix reverted" end)' "$REPORT"
echo "The sixth 2026-09-21 problem (a status card advising a command with no menu wrapper) has no fix yet. Its check is built and still honestly red; Phase 208 is what closes it."
if [ "$FAILURES" -eq 0 ]; then
  echo "Fixes proven caught this run: $APPLIED_COUNT -- never six."
else
  echo "Fixes proven caught this run: $((APPLIED_COUNT - FAILURES)) of $APPLIED_COUNT attempted -- never six."
fi

if [ "$FAILURES" -gt 0 ]; then
  fail "$FAILURES of $APPLIED_COUNT revert(s) were NOT caught by the journey -- see the rows above and $REPORT"
fi

echo
echo "PASS: every reverted fix this run attempted made the journey fail at the step it protects."
