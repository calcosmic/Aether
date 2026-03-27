#!/bin/bash
# Test suite for oracle convergence functions
# Phase 08 Plan 02 — validates compute_convergence, update_convergence_metrics,
# detect_diminishing_returns, validate_and_recover, and check_convergence

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

# Helper: Write state.json with convergence object
# Args: dir, phase, history_json, prev_findings, composite_score
write_state_with_convergence() {
  local dir="$1"
  local phase="$2"
  local history_json="$3"
  local prev_findings="${4:-0}"
  local composite_score="${5:-0}"
  cat > "$dir/state.json" <<EOF
{
  "version": "1.0",
  "topic": "Test topic",
  "scope": "codebase",
  "phase": "$phase",
  "iteration": 5,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 50,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z",
  "status": "active",
  "convergence": {
    "prev_findings_count": $prev_findings,
    "prev_overall_confidence": 40,
    "composite_score": $composite_score,
    "converged": false,
    "history": $history_json
  }
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


# ---- Test 1: compute_convergence metrics ----
test_compute_convergence_metrics() {
  local tmpdir=$(setup_tmpdir)

  # Setup: 3 questions - 1 answered, 1 partial at 75% (resolved), 1 open at 20% (not resolved)
  # Expected: gap_resolution = 2/3 = 66%, coverage = 2/3 touched = 66%
  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"answered","confidence":90,"key_findings":["f1","f2"],"iterations_touched":[1,2]},
    {"id":"q2","text":"Q2?","status":"partial","confidence":75,"key_findings":["f3","f4"],"iterations_touched":[1,2]},
    {"id":"q3","text":"Q3?","status":"open","confidence":20,"key_findings":["f5"],"iterations_touched":[]}
  ]'
  write_state_with_convergence "$tmpdir" "investigate" "[]" 2 0

  # Extract and run compute_convergence
  local result
  result=$(bash -c "
    set +e
    eval \"\$(sed -n '/^compute_convergence()/,/^}/p' '$ORACLE_SH')\"
    compute_convergence '$tmpdir/plan.json' '$tmpdir/state.json'
  ")

  # Assertion 1: gap_resolution_pct should be 66 (2/3 resolved)
  local gap_pct
  gap_pct=$(echo "$result" | jq '.gap_resolution_pct')
  run_test "compute_convergence_gap_resolution" "66" "$gap_pct"

  # Assertion 2: coverage_pct should be 66 (2/3 touched)
  local cov_pct
  cov_pct=$(echo "$result" | jq '.coverage_pct')
  run_test "compute_convergence_coverage" "66" "$cov_pct"

  # Assertion 3: novelty_delta should be 3 (5 current findings - 2 previous)
  local novelty
  novelty=$(echo "$result" | jq '.novelty_delta')
  run_test "compute_convergence_novelty_delta" "3" "$novelty"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 2: update_convergence_metrics writes state ----
test_update_convergence_metrics_writes_state() {
  local tmpdir=$(setup_tmpdir)

  write_plan "$tmpdir" '[
    {"id":"q1","text":"Q1?","status":"partial","confidence":60,"key_findings":["f1","f2"],"iterations_touched":[1,2]},
    {"id":"q2","text":"Q2?","status":"answered","confidence":85,"key_findings":["f3","f4","f5"],"iterations_touched":[1,2,3]}
  ]'
  write_state "$tmpdir" "synthesize" 3

  # Extract both functions (update_convergence_metrics depends on compute_convergence)
  bash -c "
    set +e
    eval \"\$(sed -n '/^compute_convergence()/,/^}/p' '$ORACLE_SH')\"
    eval \"\$(sed -n '/^update_convergence_metrics()/,/^}/p' '$ORACLE_SH')\"
    update_convergence_metrics '$tmpdir/state.json' '$tmpdir/plan.json'
  "

  # Assertion 1: state.json should have convergence.history with at least 1 entry
  local history_len
  history_len=$(jq '(.convergence.history // []) | length' "$tmpdir/state.json")
  if [ "$history_len" -ge 1 ]; then
    run_test "update_metrics_history_populated" "true" "true"
  else
    run_test "update_metrics_history_populated" ">=1 entries" "$history_len entries"
  fi

  # Assertion 2: prev_findings_count should be updated to current findings count (5)
  local prev_findings
  prev_findings=$(jq '.convergence.prev_findings_count' "$tmpdir/state.json")
  run_test "update_metrics_prev_findings" "5" "$prev_findings"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 3: diminishing returns detection ----
test_diminishing_returns_detection() {
  local tmpdir=$(setup_tmpdir)

  # Assertion 1: 3 entries with novelty_delta=0 in survey -> "strategy_change"
  write_state_with_convergence "$tmpdir" "survey" '[
    {"iteration":1,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":30,"coverage_pct":50,"phase":"survey"},
    {"iteration":2,"novelty_delta":1,"confidence_delta":0,"gap_resolution_pct":30,"coverage_pct":50,"phase":"survey"},
    {"iteration":3,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":30,"coverage_pct":50,"phase":"survey"}
  ]' 0 40

  local result
  result=$(bash -c "
    set +e
    eval \"\$(sed -n '/^detect_diminishing_returns()/,/^}/p' '$ORACLE_SH')\"
    detect_diminishing_returns '$tmpdir/state.json'
  ")
  run_test "diminishing_returns_survey_strategy_change" "strategy_change" "$result"

  # Assertion 2: 3 entries with novelty_delta=0 in verify -> "synthesize_now"
  write_state_with_convergence "$tmpdir" "verify" '[
    {"iteration":1,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":90,"coverage_pct":100,"phase":"verify"},
    {"iteration":2,"novelty_delta":1,"confidence_delta":0,"gap_resolution_pct":90,"coverage_pct":100,"phase":"verify"},
    {"iteration":3,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":90,"coverage_pct":100,"phase":"verify"}
  ]' 10 85

  result=$(bash -c "
    set +e
    eval \"\$(sed -n '/^detect_diminishing_returns()/,/^}/p' '$ORACLE_SH')\"
    detect_diminishing_returns '$tmpdir/state.json'
  ")
  run_test "diminishing_returns_verify_synthesize_now" "synthesize_now" "$result"

  # Assertion 3: 3 entries with high novelty -> "continue"
  write_state_with_convergence "$tmpdir" "survey" '[
    {"iteration":1,"novelty_delta":5,"confidence_delta":10,"gap_resolution_pct":30,"coverage_pct":50,"phase":"survey"},
    {"iteration":2,"novelty_delta":5,"confidence_delta":8,"gap_resolution_pct":40,"coverage_pct":60,"phase":"survey"},
    {"iteration":3,"novelty_delta":5,"confidence_delta":5,"gap_resolution_pct":50,"coverage_pct":70,"phase":"survey"}
  ]' 15 50

  result=$(bash -c "
    set +e
    eval \"\$(sed -n '/^detect_diminishing_returns()/,/^}/p' '$ORACLE_SH')\"
    detect_diminishing_returns '$tmpdir/state.json'
  ")
  run_test "diminishing_returns_high_novelty_continue" "continue" "$result"

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 4: validate_and_recover ----
test_validate_and_recover() {
  local tmpdir=$(setup_tmpdir)

  # Assertion 1: Valid JSON file returns 0
  echo '{"valid": true}' > "$tmpdir/test.json"

  local exit_code
  bash -c "
    set +e
    AETHER_ROOT='$AETHER_ROOT'
    eval \"\$(sed -n '/^validate_and_recover()/,/^}/p' '$ORACLE_SH')\"
    validate_and_recover '$tmpdir/test.json' 2>/dev/null
  "
  exit_code=$?
  run_test "validate_recover_valid_json" "0" "$exit_code"

  # Assertion 2: Invalid JSON with valid .pre-iteration backup -> recovers
  echo '{invalid json broken}' > "$tmpdir/state.json"
  echo '{"recovered": true, "version": "1.0"}' > "$tmpdir/state.json.pre-iteration"

  bash -c "
    set +e
    AETHER_ROOT='$AETHER_ROOT'
    eval \"\$(sed -n '/^validate_and_recover()/,/^}/p' '$ORACLE_SH')\"
    validate_and_recover '$tmpdir/state.json' 2>/dev/null
  "
  exit_code=$?
  run_test "validate_recover_from_backup" "0" "$exit_code"

  # Verify the file was actually restored
  local recovered
  recovered=$(jq -r '.recovered // false' "$tmpdir/state.json" 2>/dev/null || echo "false")
  if [ "$recovered" = "true" ]; then
    run_test "validate_recover_file_restored" "true" "true"
  else
    run_test "validate_recover_file_restored" "true" "$recovered"
  fi

  cleanup_tmpdir "$tmpdir"
}


# ---- Test 5: convergence check ----
test_convergence_check() {
  local tmpdir=$(setup_tmpdir)

  # Assertion 1: composite_score=90 with 2+ low-novelty history entries -> returns 0
  write_state_with_convergence "$tmpdir" "verify" '[
    {"iteration":1,"novelty_delta":5,"confidence_delta":10,"gap_resolution_pct":60,"coverage_pct":80,"phase":"investigate"},
    {"iteration":2,"novelty_delta":0,"confidence_delta":2,"gap_resolution_pct":90,"coverage_pct":100,"phase":"verify"},
    {"iteration":3,"novelty_delta":1,"confidence_delta":0,"gap_resolution_pct":90,"coverage_pct":100,"phase":"verify"}
  ]' 10 90

  local exit_code
  bash -c "
    set +e
    eval \"\$(sed -n '/^check_convergence()/,/^}/p' '$ORACLE_SH')\"
    check_convergence '$tmpdir/state.json'
  "
  exit_code=$?
  run_test "convergence_check_converged" "0" "$exit_code"

  # Assertion 2: composite_score=70 -> returns 1
  write_state_with_convergence "$tmpdir" "investigate" '[
    {"iteration":1,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":50,"coverage_pct":70,"phase":"investigate"},
    {"iteration":2,"novelty_delta":0,"confidence_delta":0,"gap_resolution_pct":50,"coverage_pct":70,"phase":"investigate"}
  ]' 5 70

  bash -c "
    set +e
    eval \"\$(sed -n '/^check_convergence()/,/^}/p' '$ORACLE_SH')\"
    check_convergence '$tmpdir/state.json'
  "
  exit_code=$?
  run_test "convergence_check_not_converged" "1" "$exit_code"

  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle Convergence Test Suite"
echo "========================================="
echo ""

test_compute_convergence_metrics
test_update_convergence_metrics_writes_state
test_diminishing_returns_detection
test_validate_and_recover
test_convergence_check

echo ""
echo "========================================="
echo "Oracle Convergence Tests: $TESTS_PASSED passed, $TESTS_FAILED failed out of $TESTS_RUN"
echo "========================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
