#!/usr/bin/env bash
# 02-brownfield-feature.sh — deterministic pass/fail judge for
# bench/tasks/02-brownfield-feature.md
#
# What this judges: whether the zod `.lowercase()` string-validation feature
# was actually built and is reachable from the library's own public surface,
# by exercising it directly — not by checking for the presence of files.
#
# Usage: 02-brownfield-feature.sh RESULT_CLONE_PATH
#   RESULT_CLONE_PATH — a fresh clone of the run's resulting repository state.
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

TASK_ID="02-brownfield-feature"

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
# raw newlines/tabs from captured go test / npm test output).
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
  echo "FAIL argument-check: a path to a fresh clone of the run's resulting repository / no argument was given" >&2
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

cd "$RESULT_DIR"

# Dependencies must be installed to run the probe and the existing suite.
INSTALL_STATUS=0
INSTALL_OUT=""
if [ ! -d node_modules ]; then
  INSTALL_OUT=$(npm install 2>&1) || INSTALL_STATUS=$?
fi
if [ "$INSTALL_STATUS" -ne 0 ]; then
  echo "FAIL dependency-install: npm install exits 0 / exit $INSTALL_STATUS: $(printf '%s' "$INSTALL_OUT" | tail -5 | tr '\n' ' ')" >&2
  exit 1
fi

# Assertion 1: the pre-existing test suite still passes.
SUITE_OUT=""
SUITE_STATUS=0
SUITE_OUT=$(npm test 2>&1) || SUITE_STATUS=$?
record "existing-suite-passes" \
  "npm test exits 0" \
  "exit $SUITE_STATUS: $(printf '%s' "$SUITE_OUT" | tail -5 | tr '\n' ' ')" \
  "$SUITE_STATUS"

# Assertion 2 + 3: the feature's behaviour, exercised directly through the
# library's own public entry point (import from the package source, not an
# isolated file) — rejects a string with an uppercase character, accepts one
# with none. Run as a throwaway jest test (the same test runner the repo's
# own suite already uses via ts-jest) rather than a standalone ts-node
# script — ts-node's bin resolution proved unreliable in a fresh npm install
# on this environment, where jest's own binary (already exercised by the
# existing-suite assertion above) is proven to work.
PROBE_TEST_FILE="src/__tests__/acceptance-probe-lowercase.test.ts"
PROBE_WROTE_FILE=0
if [ ! -e "$PROBE_TEST_FILE" ]; then
  PROBE_WROTE_FILE=1
  mkdir -p src/__tests__
  cat >"$PROBE_TEST_FILE" <<'PROBE'
import { z } from "../index";

test("acceptance probe: lowercase reachability", () => {
  const schema = z.string().lowercase();
  const out = {
    badOk: schema.safeParse("HasUpper").success,
    goodOk: schema.safeParse("alllower").success,
    emptyOk: schema.safeParse("").success,
  };
  console.log("ACCEPTANCE_PROBE_RESULT " + JSON.stringify(out));
  expect(out.badOk).toBe(false);
  expect(out.goodOk).toBe(true);
  expect(out.emptyOk).toBe(true);
});
PROBE
fi

PROBE_OUT=""
PROBE_STATUS=0
PROBE_OUT=$(node_modules/.bin/jest acceptance-probe-lowercase --silent=false 2>&1) || PROBE_STATUS=$?
[ "$PROBE_WROTE_FILE" -eq 1 ] && rm -f "$PROBE_TEST_FILE"

record "lowercase-behaviour-correct" \
  "z.string().lowercase() rejects a string containing an uppercase letter, accepts an all-lowercase string, and accepts an empty string" \
  "exit $PROBE_STATUS, output: $(printf '%s' "$PROBE_OUT" | tail -8 | tr '\n' ' ')" \
  "$PROBE_STATUS"

# Assertion 4: the feature is reachable from the repository's own public
# surface (the .lowercase() method exists on the exported string schema
# class), not only defined in an isolated file never wired up.
SURFACE_STATUS=1
SURFACE_FOUND="no .lowercase( builder method found on the exported string schema"
if grep -rq "lowercase" src/types.ts 2>/dev/null && grep -q "lowercase(" src/types.ts 2>/dev/null; then
  SURFACE_STATUS=0
  SURFACE_FOUND=".lowercase( builder method found in src/types.ts"
fi
record "feature-on-public-surface" \
  "src/types.ts defines a .lowercase( builder method on the string schema class" \
  "$SURFACE_FOUND" \
  "$SURFACE_STATUS"

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
