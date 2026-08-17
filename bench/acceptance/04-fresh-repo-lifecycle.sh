#!/usr/bin/env bash
# 04-fresh-repo-lifecycle.sh — deterministic pass/fail judge for
# bench/tasks/04-fresh-repo-lifecycle.md
#
# What this judges: whether a working Celsius/Fahrenheit conversion program
# with real automated tests was actually produced from an empty directory,
# by running the program and its own tests — plus the one Aether-specific
# assertion this category previews from a later benchmark round: that
# Aether's own source repository was left untouched while it worked on
# someone else's project.
#
# Usage: 04-fresh-repo-lifecycle.sh RESULT_CLONE_PATH
#   RESULT_CLONE_PATH — a fresh clone (or the resulting directory itself) of
#   the run's resulting project repository state.
#
# Required environment: AETHER_REPO must be set to the path of the Aether
# source repository used to run the lane under test. This script fails with
# a named message if it is unset — there is no default, because guessing
# the wrong path would silently skip the one check this category exists to
# add.
#
# Pass means: exit 0. Every assertion below ran and every one passed.
# Fail means: exit 1. At least one assertion below failed; every failure is
# printed as its own "FAIL <name>: <expected> / <found>" line — this script
# never stops at the first failure, so a bad run's full failure picture is
# visible in one call.
#
# Vocabulary rule: this script never spells out either compared system's own
# internal terminology for its own checking or artifact naming — see
# bench/acceptance/README.md's "Vocabulary rule" section for the full banned
# list. Every assertion here is phrased in terms of the resulting
# repository's files and test results alone — reusing either system's
# verification vocabulary would let that system's own conventions decide the
# verdict instead of the actual code.
set -euo pipefail

TASK_ID="04-fresh-repo-lifecycle"

FAILURES=()
declare -a ASSERTION_NAMES=()
declare -a ASSERTION_VERDICTS=()
declare -a ASSERTION_DETAILS=()

record() {
  local name="$1" expected="$2" found="$3" status="$4"
  ASSERTION_NAMES+=("$name")
  ASSERTION_DETAILS+=("$expected|||$found")
  if [ "$status" -eq 0 ]; then
    ASSERTION_VERDICTS+=("pass")
  else
    ASSERTION_VERDICTS+=("fail")
    FAILURES+=("FAIL $name: $expected / $found")
  fi
}

# build_json — renders the assertions array via jq so any control character,
# tab, or embedded quote in a test-output-derived "found" string is escaped
# correctly rather than hand-escaped with sed (which cannot safely handle
# raw newlines/tabs from captured test/program output).
build_json() {
  local n=${#ASSERTION_NAMES[@]}
  local records="[]"
  local i
  for ((i = 0; i < n; i++)); do
    local name="${ASSERTION_NAMES[$i]}"
    local verdict="${ASSERTION_VERDICTS[$i]}"
    local details="${ASSERTION_DETAILS[$i]}"
    local expected="${details%%|||*}"
    local found="${details#*|||}"
    local record
    record=$(jq -n -c \
      --arg name "$name" \
      --arg verdict "$verdict" \
      --arg expected "$expected" \
      --arg found "$found" \
      '{name: $name, verdict: $verdict, expected: $expected, found: $found}')
    records=$(printf '%s' "$records" | jq -c --argjson r "$record" '. + [$r]')
  done
  jq -n -c \
    --arg task_id "$TASK_ID" \
    --argjson assertions "$records" \
    --argjson pass "$([ ${#FAILURES[@]} -eq 0 ] && echo true || echo false)" \
    '{task_id: $task_id, assertions: $assertions, pass: $pass}'
}

emit_json() {
  local json
  json="$(build_json)"
  if [ -n "${ACCEPTANCE_JSON:-}" ]; then
    printf '%s\n' "$json" >"$ACCEPTANCE_JSON"
  elif { true >&3; } 2>/dev/null; then
    printf '%s\n' "$json" >&3
  fi
}

if [ $# -lt 1 ]; then
  echo "FAIL argument-check: a path to the run's resulting project directory / no argument was given" >&2
  exit 1
fi
RESULT_DIR="$1"

# A ".git" entry can be a directory (a normal clone) or a file (a git
# worktree, whose ".git" is a text pointer to the real gitdir elsewhere) —
# `git rev-parse --is-inside-work-tree` handles both, so this check does not
# reject a legitimate worktree the way a bare `-d .git` test would.
if ! (cd "$RESULT_DIR" 2>/dev/null && git rev-parse --is-inside-work-tree >/dev/null 2>&1); then
  echo "FAIL repository-check: $RESULT_DIR is a git repository / no git repository found there" >&2
  exit 1
fi

if [ -z "${AETHER_REPO:-}" ]; then
  echo "FAIL aether-repo-env-check: the AETHER_REPO environment variable names the Aether source repository path / AETHER_REPO is unset" >&2
  exit 1
fi
if ! (cd "$AETHER_REPO" 2>/dev/null && git rev-parse --is-inside-work-tree >/dev/null 2>&1); then
  echo "FAIL aether-repo-env-check: AETHER_REPO ($AETHER_REPO) points to a git repository / no git repository found there" >&2
  exit 1
fi

cd "$RESULT_DIR"

# Assertion 1: at least one commit exists and the working tree has no
# uncommitted changes to source files.
COMMIT_COUNT=$(git rev-list --count HEAD 2>/dev/null || echo 0)
if [ "$COMMIT_COUNT" -ge 1 ]; then
  COMMIT_STATUS=0
  COMMIT_FOUND="$COMMIT_COUNT commit(s) present"
else
  COMMIT_STATUS=1
  COMMIT_FOUND="0 commits present"
fi
record "has-at-least-one-commit" \
  "the resulting repository has at least one commit" \
  "$COMMIT_FOUND" \
  "$COMMIT_STATUS"

WORKING_TREE_STATUS_OUT=$(git status --porcelain 2>/dev/null || true)
if [ -z "$WORKING_TREE_STATUS_OUT" ]; then
  CLEAN_STATUS=0
  CLEAN_FOUND="git status --porcelain is empty"
else
  CLEAN_STATUS=1
  CLEAN_FOUND="git status --porcelain shows: $(printf '%s' "$WORKING_TREE_STATUS_OUT" | tr '\n' ' ')"
fi
record "working-tree-clean" \
  "git status --porcelain in the resulting repository produces no output" \
  "$CLEAN_FOUND" \
  "$CLEAN_STATUS"

# Assertion 2: the produced program's own tests exist and pass, using
# whatever the system documented for "how to run it" (README.md), falling
# back to ecosystem-standard commands if the README doesn't name one
# explicitly.
TEST_CMD=""
if [ -f README.md ]; then
  TEST_CMD=$(grep -oE '(go test[^`\n]*|npm test[^`\n]*|python -m pytest[^`\n]*|pytest[^`\n]*)' README.md 2>/dev/null | head -1 || true)
fi
if [ -z "$TEST_CMD" ]; then
  if [ -f go.mod ]; then
    TEST_CMD="go test ./..."
  elif [ -f package.json ]; then
    TEST_CMD="npm test"
  elif ls test_*.py >/dev/null 2>&1 || ls *_test.py >/dev/null 2>&1; then
    TEST_CMD="python -m pytest"
  fi
fi

TEST_STATUS=1
TEST_FOUND="no test command could be determined from README.md or ecosystem files present"
if [ -n "$TEST_CMD" ]; then
  TEST_OUT=""
  TEST_RUN_STATUS=0
  if [ -f package.json ] && [ ! -d node_modules ]; then
    npm install >/dev/null 2>&1 || true
  fi
  TEST_OUT=$(eval "$TEST_CMD" 2>&1) || TEST_RUN_STATUS=$?
  TEST_STATUS="$TEST_RUN_STATUS"
  TEST_FOUND="ran '$TEST_CMD', exit $TEST_RUN_STATUS: $(printf '%s' "$TEST_OUT" | tail -5 | tr '\n' ' ')"
fi
record "own-tests-exist-and-pass" \
  "the project's own documented test command exits 0" \
  "$TEST_FOUND" \
  "$TEST_STATUS"

# Assertion 3: the produced program exhibits its stated behaviour — run it
# directly and confirm both fixed points convert correctly in both
# directions. Try the most likely entry points for each ecosystem.
run_program() {
  local value="$1" unit="$2"
  if [ -f main.go ]; then
    go run . "$value" "$unit" 2>&1
  elif [ -f main.py ]; then
    python3 main.py "$value" "$unit" 2>&1
  elif [ -f main.js ]; then
    node main.js "$value" "$unit" 2>&1
  elif [ -f main.ts ]; then
    npx --yes ts-node main.ts "$value" "$unit" 2>&1
  else
    echo "__NO_ENTRY_POINT__"
    return 127
  fi
}

RUN_0C_OUT=$(run_program 0 C) || true
RUN_100C_OUT=$(run_program 100 C) || true
RUN_32F_OUT=$(run_program 32 F) || true
RUN_212F_OUT=$(run_program 212 F) || true

PROGRAM_STATUS=0
PROGRAM_FOUND="0C->32, 100C->212, 32F->0, 212F->100 all confirmed in program output"
if [ "$RUN_0C_OUT" = "__NO_ENTRY_POINT__" ]; then
  PROGRAM_STATUS=1
  PROGRAM_FOUND="no recognizable program entry point found (main.go, main.py, main.js, main.ts)"
elif ! printf '%s' "$RUN_0C_OUT" | grep -q "32" || \
     ! printf '%s' "$RUN_100C_OUT" | grep -q "212" || \
     ! printf '%s' "$RUN_32F_OUT" | grep -q "0" || \
     ! printf '%s' "$RUN_212F_OUT" | grep -q "100"; then
  PROGRAM_STATUS=1
  PROGRAM_FOUND="0C->[$RUN_0C_OUT] 100C->[$RUN_100C_OUT] 32F->[$RUN_32F_OUT] 212F->[$RUN_212F_OUT]"
fi
record "program-behaviour-correct" \
  "running the program converts 0C to 32F, 100C to 212F, 32F to 0C, and 212F to 100C" \
  "$PROGRAM_FOUND" \
  "$PROGRAM_STATUS"

# Assertion 4 (Aether-specific — marked clearly so the results table can
# report it separately; this previews Phase 192's PROOF-02 check): the
# Aether source repository's own working tree is clean after the run,
# proving Aether did not modify itself while working on this fresh project.
AETHER_STATUS_OUT=$(cd "$AETHER_REPO" && git status --porcelain 2>/dev/null || true)
if [ -z "$AETHER_STATUS_OUT" ]; then
  AETHER_CLEAN_STATUS=0
  AETHER_CLEAN_FOUND="git status --porcelain in AETHER_REPO ($AETHER_REPO) is empty"
else
  AETHER_CLEAN_STATUS=1
  AETHER_CLEAN_FOUND="git status --porcelain in AETHER_REPO ($AETHER_REPO) shows: $(printf '%s' "$AETHER_STATUS_OUT" | tr '\n' ' ')"
fi
record "aether-source-repo-clean [PROOF-02-preview]" \
  "git status --porcelain in the Aether source repository (AETHER_REPO) produces no output" \
  "$AETHER_CLEAN_FOUND" \
  "$AETHER_CLEAN_STATUS"

emit_json

if [ ${#FAILURES[@]} -eq 0 ]; then
  echo "PASS $TASK_ID"
  exit 0
else
  for f in "${FAILURES[@]}"; do
    echo "$f"
  done
  exit 1
fi
