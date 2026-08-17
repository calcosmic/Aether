#!/usr/bin/env bash
# 01-bug-fix.sh — deterministic pass/fail judge for bench/tasks/01-bug-fix.md
#
# What this judges: whether the seeded gorilla/mux bug (cleanPath dropping
# the trailing-slash check on the wrong end of the string) was actually
# fixed in the code, not in the test that caught it.
#
# Usage: 01-bug-fix.sh RESULT_CLONE_PATH
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

TASK_ID="01-bug-fix"
SEEDED_TEST_FILE="mux_test.go"
SEEDED_TEST_NAME="TestCleanPathPreservesNonSlashInput"

# --- Byte-identical reference copy of the seeded test file, as written by
# the harness's setup step (see bench/tasks/01-bug-fix.md "Setup"). Used to
# assert the fix did not come from loosening the test.
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

# check NAME EXPECTED_DESC ACTUAL_DESC CONDITION_EXIT_CODE
# Records one assertion's pass/fail without exiting the script.
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

# emit_json — writes the assertions JSON to $ACCEPTANCE_JSON if set, else to
# file descriptor 3 if that fd is open (the runner opens it when it wants the
# machine-readable detail); silently does nothing further if neither target
# is available, since stdout is reserved for the human-readable PASS/FAIL
# lines only.
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

# Assertion 1: the seeded single-test command passes.
SINGLE_TEST_OUT=""
SINGLE_TEST_STATUS=0
SINGLE_TEST_OUT=$(go test -run "$SEEDED_TEST_NAME" -v ./... 2>&1) || SINGLE_TEST_STATUS=$?
record "seeded-test-passes" \
  "go test -run $SEEDED_TEST_NAME -v ./... exits 0" \
  "exit $SINGLE_TEST_STATUS: $(printf '%s' "$SINGLE_TEST_OUT" | tail -3 | tr '\n' ' ')" \
  "$SINGLE_TEST_STATUS"

# Assertion 2: the seeded test file is byte-identical to the seeded version —
# checked by confirming the exact seeded function body is present verbatim.
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

# Assertion 3: the seeded test name still exists (not renamed/skipped/deleted).
if grep -q "func $SEEDED_TEST_NAME(" "$SEEDED_TEST_FILE" 2>/dev/null; then
  NAME_STATUS=0
  NAME_FOUND="function $SEEDED_TEST_NAME still present"
else
  NAME_STATUS=1
  NAME_FOUND="function $SEEDED_TEST_NAME not found in $SEEDED_TEST_FILE"
fi
record "seeded-test-name-intact" \
  "$SEEDED_TEST_NAME exists in $SEEDED_TEST_FILE, not renamed/skipped/deleted" \
  "$NAME_FOUND" \
  "$NAME_STATUS"

# Assertion 4: the full test suite passes (no pre-existing test broken to
# make the seeded one pass).
FULL_SUITE_OUT=""
FULL_SUITE_STATUS=0
FULL_SUITE_OUT=$(go test ./... 2>&1) || FULL_SUITE_STATUS=$?
record "full-suite-passes" \
  "go test ./... exits 0" \
  "exit $FULL_SUITE_STATUS: $(printf '%s' "$FULL_SUITE_OUT" | tail -3 | tr '\n' ' ')" \
  "$FULL_SUITE_STATUS"

# Assertion 5: the defect's actual behaviour is corrected, exercised
# independently of the seeded test — write a throwaway probe program that
# calls the unexported cleanPath function directly via an internal test file,
# distinct from the seeded test, so a fix that special-cases the seeded input
# string alone is caught.
PROBE_FILE="acceptance_probe_test.go"
cat >"$PROBE_FILE" <<'PROBE'
package mux

import "testing"

// acceptance probe: independent of the seeded test, exercises cleanPath
// directly with different inputs to catch a fix that special-cases the
// seeded test's exact string rather than fixing the underlying check.
func TestAcceptanceProbeCleanPathIndependent(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/alpha/beta", "/alpha/beta"},
		{"/gamma", "/gamma"},
		{"/delta/", "/delta/"},
	}
	for _, c := range cases {
		got := cleanPath(c.in)
		if got != c.want {
			t.Fatalf("cleanPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
PROBE
PROBE_OUT=""
PROBE_STATUS=0
PROBE_OUT=$(go test -run TestAcceptanceProbeCleanPathIndependent -v ./... 2>&1) || PROBE_STATUS=$?
rm -f "$PROBE_FILE"
record "defect-fixed-independently" \
  "cleanPath does not append a trailing slash to inputs without one, verified independently of the seeded test's exact string" \
  "exit $PROBE_STATUS: $(printf '%s' "$PROBE_OUT" | tail -3 | tr '\n' ' ')" \
  "$PROBE_STATUS"

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
