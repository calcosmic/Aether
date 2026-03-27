#!/bin/bash
# Test suite for oracle phase transitions and iteration counter
# Phase 07 Plan 02 — validates determine_phase logic, iteration increment, and state file updates

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORACLE_SH="$AETHER_ROOT/.aether/utils/oracle/oracle.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Colors for output (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Helper: Run test and track result
run_test() {
  local name="$1"
  local expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" == *"$expected"* ]]; then
    echo -e "${GREEN}PASS${NC}: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo -e "${RED}FAIL${NC}: $name"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# Helper: Setup temporary directory
setup_tmpdir() {
  mktemp -d
}

# Helper: Cleanup temporary directory
cleanup_tmpdir() {
  local dir="$1"
  rm -rf "$dir"
}

# Helper: Extract and run determine_phase from oracle.sh
run_determine_phase() {
  local state_file="$1"
  local plan_file="$2"

  bash -c "
    set +e
    eval \"\$(sed -n '/^determine_phase()/,/^}/p' '$ORACLE_SH')\"
    determine_phase '$state_file' '$plan_file'
  "
}

# Helper: Write state.json
write_state() {
  local dir="$1"
  local phase="$2"
  local iteration="${3:-0}"
  cat > "$dir/state.json" <<EOF
{
  "version": "1.0",
  "topic": "Test topic",
  "scope": "codebase",
  "phase": "$phase",
  "iteration": $iteration,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 0,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z",
  "status": "active"
}
EOF
}

# Helper: Write plan.json with custom questions (pass JSON array string)
write_plan() {
  local dir="$1"
  local questions_json="$2"
  cat > "$dir/plan.json" <<EOF
{
  "version": "1.0",
  "questions": $questions_json,
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z"
}
EOF
}


# ---- Test 1: Iteration counter increment ----
test_iteration_increment() {
  local tmpdir=$(setup_tmpdir)

  write_state "$tmpdir" "survey" 0

  # Run the same jq increment command that oracle.sh uses
  local ts="2026-03-13T01:00:00Z"
  jq --arg ts "$ts" '.iteration += 1 | .last_updated = $ts' "$tmpdir/state.json" > "$tmpdir/state.json.tmp" && mv "$tmpdir/state.json.tmp" "$tmpdir/state.json"

  local iteration
  iteration=$(jq -r '.iteration' "$tmpdir/state.json")
  run_test "iteration_increment_first" "1" "$iteration"

  # Increment again
  local ts2="2026-03-13T02:00:00Z"
  jq --arg ts "$ts2" '.iteration += 1 | .last_updated = $ts' "$tmpdir/state.json" > "$tmpdir/state.json.tmp" && mv "$tmpdir/state.json.tmp" "$tmpdir/state.json"

  iteration=$(jq -r '.iteration' "$tmpdir/state.json")
  run_test "iteration_increment_second" "2" "$iteration"

  # Verify last_updated is a valid ISO-8601 timestamp
  local last_updated
  last_updated=$(jq -r '.last_updated' "$tmpdir/state.json")
  if [[ "$last_updated" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
    run_test "iteration_last_updated_iso8601" "valid" "valid"
  else
    run_test "iteration_last_updated_iso8601" "valid ISO-8601" "$last_updated"
  fi

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 2: Phase transition survey -> investigate ----
test_phase_transition_survey_to_investigate() {
  local tmpdir=$(setup_tmpdir)

  # Scenario A: All questions touched -> investigate
  write_state "$tmpdir" "survey"
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"open","confidence":10,"key_findings":[],"iterations_touched":[1]},
    {"id":"q2","text":"Q2?","status":"open","confidence":15,"key_findings":[],"iterations_touched":[1]},
    {"id":"q3","text":"Q3?","status":"open","confidence":5,"key_findings":[],"iterations_touched":[1]}
  ]'

  local phase
  phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "survey_to_investigate_all_touched" "investigate" "$phase"

  # Scenario B: Avg confidence >= 25 (not all touched) -> investigate
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"open","confidence":40,"key_findings":["f1"],"iterations_touched":[1]},
    {"id":"q2","text":"Q2?","status":"open","confidence":30,"key_findings":[],"iterations_touched":[1]},
    {"id":"q3","text":"Q3?","status":"open","confidence":20,"key_findings":[],"iterations_touched":[]}
  ]'

  phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "survey_to_investigate_avg_25" "investigate" "$phase"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 3: Phase transition investigate -> synthesize ----
test_phase_transition_investigate_to_synthesize() {
  local tmpdir=$(setup_tmpdir)

  write_state "$tmpdir" "investigate"
  # Avg confidence 65%
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"partial","confidence":60,"key_findings":["f1"],"iterations_touched":[1,2]},
    {"id":"q2","text":"Q2?","status":"partial","confidence":70,"key_findings":["f2"],"iterations_touched":[1,2]},
    {"id":"q3","text":"Q3?","status":"partial","confidence":65,"key_findings":["f3"],"iterations_touched":[1,2]}
  ]'

  local phase
  phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "investigate_to_synthesize_avg_60" "synthesize" "$phase"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 4: Phase stays when threshold not met ----
test_phase_stays_when_threshold_not_met() {
  local tmpdir=$(setup_tmpdir)

  # Survey stays survey: no questions touched, avg confidence 10%
  write_state "$tmpdir" "survey"
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"open","confidence":10,"key_findings":[],"iterations_touched":[]},
    {"id":"q2","text":"Q2?","status":"open","confidence":10,"key_findings":[],"iterations_touched":[]},
    {"id":"q3","text":"Q3?","status":"open","confidence":10,"key_findings":[],"iterations_touched":[]}
  ]'

  local phase
  phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "survey_stays_survey" "survey" "$phase"

  # Investigate stays investigate: avg confidence 40%, 3 questions below 50%
  write_state "$tmpdir" "investigate"
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"open","confidence":40,"key_findings":[],"iterations_touched":[1,2]},
    {"id":"q2","text":"Q2?","status":"open","confidence":35,"key_findings":[],"iterations_touched":[1,2]},
    {"id":"q3","text":"Q3?","status":"open","confidence":45,"key_findings":[],"iterations_touched":[1]},
    {"id":"q4","text":"Q4?","status":"partial","confidence":65,"key_findings":["f1"],"iterations_touched":[1,2]}
  ]'

  phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "investigate_stays_investigate" "investigate" "$phase"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 5: Phase transition updates state file (full read-compare-write cycle) ----
test_phase_transition_updates_state_file() {
  local tmpdir=$(setup_tmpdir)

  # Start in survey, plan data triggers investigate transition
  write_state "$tmpdir" "survey"
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"open","confidence":30,"key_findings":["f1"],"iterations_touched":[1]},
    {"id":"q2","text":"Q2?","status":"open","confidence":30,"key_findings":[],"iterations_touched":[1]},
    {"id":"q3","text":"Q3?","status":"open","confidence":30,"key_findings":[],"iterations_touched":[1]}
  ]'

  # Determine new phase
  local new_phase
  new_phase=$(run_determine_phase "$tmpdir/state.json" "$tmpdir/plan.json")
  run_test "transition_determines_investigate" "investigate" "$new_phase"

  # Apply the transition (same jq command as oracle.sh)
  local current_phase
  current_phase=$(jq -r '.phase' "$tmpdir/state.json")
  if [ "$new_phase" != "$current_phase" ]; then
    jq --arg phase "$new_phase" '.phase = $phase' "$tmpdir/state.json" > "$tmpdir/state.json.tmp" && mv "$tmpdir/state.json.tmp" "$tmpdir/state.json"
  fi

  # Verify state.json was updated
  local file_phase
  file_phase=$(jq -r '.phase' "$tmpdir/state.json")
  run_test "transition_updates_state_file" "investigate" "$file_phase"

  # Verify state.json is still valid JSON
  if jq -e . "$tmpdir/state.json" > /dev/null 2>&1; then
    run_test "transition_state_remains_valid_json" "valid" "valid"
  else
    run_test "transition_state_remains_valid_json" "valid JSON" "invalid JSON"
  fi

  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle Phase Transition Test Suite"
echo "========================================="
echo ""

test_iteration_increment
test_phase_transition_survey_to_investigate
test_phase_transition_investigate_to_synthesize
test_phase_stays_when_threshold_not_met
test_phase_transition_updates_state_file

echo ""
echo "========================================="
echo "Oracle Phase Tests: $TESTS_PASSED passed, $TESTS_FAILED failed out of $TESTS_RUN"
echo "========================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
