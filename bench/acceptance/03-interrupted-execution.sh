#!/usr/bin/env bash
# 03-interrupted-execution.sh — deterministic pass/fail judge for
# bench/tasks/03-interrupted-execution.md
#
# What this judges: the same seeded gorilla/mux bug fix as Task 01 (this
# category reuses that exact task on purpose — the only variable it tests is
# kill/resume behaviour), PLUS that the interruption itself left no damage:
# work done before the kill survived, and no duplicate/conflicting partial
# work exists.
#
# Usage: 03-interrupted-execution.sh RESULT_CLONE_PATH
#   RESULT_CLONE_PATH — a fresh clone of the run's resulting repository state,
#   taken AFTER the kill-and-resume cycle completed.
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
# verdict instead of the actual code. Recovery is a property of the
# resulting repository, so it is asserted here rather than inferred from the
# operator log.
set -euo pipefail

TASK_ID="03-interrupted-execution"
SEEDED_TEST_FILE="mux_test.go"
SEEDED_TEST_NAME="TestCleanPathPreservesNonSlashInput"

read -r -d '' SEEDED_TEST_FUNC <<'EOF' || true
// TestCleanPathPreservesNonSlashInput verifies that cleanPath does not append
// a trailing slash to a path that did not have one to begin with.
func TestCleanPathPreservesNonSlashInput(t *testing.T) {
	got := cleanPath("/foo/bar")
	want := "/foo/bar"
	if got != want {
		t.Fatalf("cleanPath(%q) = %q, want %q (a path with no trailing slash must not gain one)", "/foo/bar", got, want)
	}
}
EOF

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
# raw newlines/tabs from captured go test output).
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

# --- Everything Task 01 asserts (this is the same underlying task) ---

SINGLE_TEST_OUT=""
SINGLE_TEST_STATUS=0
SINGLE_TEST_OUT=$(go test -run "$SEEDED_TEST_NAME" -v ./... 2>&1) || SINGLE_TEST_STATUS=$?
record "seeded-test-passes" \
  "go test -run $SEEDED_TEST_NAME -v ./... exits 0" \
  "exit $SINGLE_TEST_STATUS: $(printf '%s' "$SINGLE_TEST_OUT" | tail -3 | tr '\n' ' ')" \
  "$SINGLE_TEST_STATUS"

if [ -f "$SEEDED_TEST_FILE" ] && grep -qF "$SEEDED_TEST_FUNC" "$SEEDED_TEST_FILE" 2>/dev/null; then
  TEST_FILE_STATUS=0
  TEST_FILE_FOUND="seeded function body present verbatim"
else
  TEST_FILE_STATUS=1
  TEST_FILE_FOUND="seeded function body missing or altered in $SEEDED_TEST_FILE"
fi
record "seeded-test-file-unchanged" \
  "$SEEDED_TEST_FILE contains the seeded $SEEDED_TEST_NAME function body byte-for-byte" \
  "$TEST_FILE_FOUND" \
  "$TEST_FILE_STATUS"

FULL_SUITE_OUT=""
FULL_SUITE_STATUS=0
FULL_SUITE_OUT=$(go test ./... 2>&1) || FULL_SUITE_STATUS=$?
record "full-suite-passes" \
  "go test ./... exits 0" \
  "exit $FULL_SUITE_STATUS: $(printf '%s' "$FULL_SUITE_OUT" | tail -3 | tr '\n' ' ')" \
  "$FULL_SUITE_STATUS"

# --- Recovery-specific assertions ---

# Assertion: the work performed before the kill is still present. The only
# file the fix touches is mux.go, so its presence and correctness (already
# proven by the full-suite and single-test assertions above) is direct
# evidence the pre-kill edit survived rather than being reverted. This
# assertion additionally confirms mux.go actually differs from the seeded
# defect line, so a resume that silently discarded all prior work and never
# re-applied the fix is caught even if some other path made the tests pass
# by coincidence.
DEFECT_LINE_STATUS=1
DEFECT_LINE_FOUND="the seeded defective line (checking p[0] instead of p[len(p)-1]) is still present in mux.go"
if [ -f mux.go ] && ! grep -q "p\[0\] == '/' && np != \"/\"" mux.go 2>/dev/null; then
  DEFECT_LINE_STATUS=0
  DEFECT_LINE_FOUND="the seeded defective line is no longer present — mux.go was actually edited, not reverted or skipped"
fi
record "pre-kill-work-survived" \
  "mux.go no longer contains the seeded defective condition (the pre-kill/post-kill edit survived the interruption)" \
  "$DEFECT_LINE_FOUND" \
  "$DEFECT_LINE_STATUS"

# Assertion: no duplicate or conflicting partial work exists — the fixed
# line (checking p[len(p)-1]) appears exactly once in cleanPath, not
# duplicated by a resume that re-applied the same edit on top of itself or
# left a conflicting second attempt nearby.
DUP_COUNT=0
if [ -f mux.go ]; then
  DUP_COUNT=$(grep -c "p\[len(p)-1\] == '/' && np != \"/\"" mux.go 2>/dev/null || true)
  DUP_COUNT=${DUP_COUNT:-0}
fi
if [ "$DUP_COUNT" -eq 1 ]; then
  DUP_STATUS=0
  DUP_FOUND="the corrected condition appears exactly once in mux.go"
else
  DUP_STATUS=1
  DUP_FOUND="the corrected condition appears $DUP_COUNT times in mux.go (expected exactly 1 — a resume left duplicate or missing partial work)"
fi
record "no-duplicate-partial-work" \
  "the corrected cleanPath condition appears exactly once in mux.go, with no duplicated or conflicting partial edits" \
  "$DUP_FOUND" \
  "$DUP_STATUS"

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
