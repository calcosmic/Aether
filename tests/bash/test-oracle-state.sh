#!/bin/bash
# Test suite for oracle state file lifecycle
# Phase 06 Plan 02 — validates session management with new oracle state files

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
UTILS_SCRIPT="$AETHER_ROOT/.aether/aether-utils.sh"

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

# Helper: Create all 5 oracle state files in a directory
create_oracle_files() {
  local dir="$1"
  echo '{"version":"1.0","topic":"Test","scope":"codebase","phase":"survey","iteration":0,"max_iterations":15,"target_confidence":95,"overall_confidence":0,"started_at":"2026-03-13T00:00:00Z","last_updated":"2026-03-13T00:00:00Z","status":"active"}' > "$dir/state.json"
  echo '{"version":"1.0","questions":[{"id":"q1","text":"Test?","status":"open","confidence":0,"key_findings":[],"iterations_touched":[]}],"created_at":"2026-03-13T00:00:00Z","last_updated":"2026-03-13T00:00:00Z"}' > "$dir/plan.json"
  echo "# Knowledge Gaps" > "$dir/gaps.md"
  echo "# Research Synthesis" > "$dir/synthesis.md"
  echo "# Research Plan" > "$dir/research-plan.md"
}

# ---- Test 1: session-verify-fresh finds new oracle files ----
test_verify_fresh_finds_new_files() {
  local tmpdir=$(setup_tmpdir)
  local start_time=$(date +%s)

  # Create all 5 state files (they are fresh since just created)
  create_oracle_files "$tmpdir"

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command oracle "" "$start_time")

  # Should find 5 fresh files
  local fresh_count
  fresh_count=$(echo "$result" | jq -r '.fresh | length')
  run_test "verify_fresh_finds_new_files_count" "5" "$fresh_count"

  run_test "verify_fresh_finds_new_files_ok" '"ok":true' "$result"

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 2: session-verify-fresh reports missing files ----
test_verify_fresh_reports_missing() {
  local tmpdir=$(setup_tmpdir)
  local start_time=$(date +%s)

  # Only create state.json (missing 4 other files)
  echo '{"version":"1.0","topic":"Test"}' > "$tmpdir/state.json"

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command oracle "" "$start_time")

  # Should report missing files
  local missing_count
  missing_count=$(echo "$result" | jq -r '.missing | length')

  # We expect 4 missing files: plan.json, gaps.md, synthesis.md, research-plan.md
  if [[ "$missing_count" -ge 4 ]]; then
    run_test "verify_fresh_reports_missing" "4+ missing" "4+ missing"
  else
    run_test "verify_fresh_reports_missing" "4+ missing" "$missing_count missing"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 3: session-clear removes oracle files ----
test_clear_removes_oracle_files() {
  local tmpdir=$(setup_tmpdir)

  # Create all 5 state files + .stop
  create_oracle_files "$tmpdir"
  touch "$tmpdir/.stop"

  # Run session-clear
  ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-clear --command oracle > /dev/null 2>&1

  # Check that state files are removed
  local remaining=0
  for f in state.json plan.json gaps.md synthesis.md research-plan.md .stop; do
    [[ -f "$tmpdir/$f" ]] && remaining=$((remaining + 1))
  done

  if [[ "$remaining" -eq 0 ]]; then
    run_test "clear_removes_oracle_files" "all removed" "all removed"
  else
    run_test "clear_removes_oracle_files" "all removed" "$remaining files still exist"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 4: session-clear preserves archive directory ----
test_clear_preserves_archive() {
  local tmpdir=$(setup_tmpdir)

  # Create state files + archive directory with content
  create_oracle_files "$tmpdir"
  mkdir -p "$tmpdir/archive/2026-03-13-120000"
  echo '{"archived":true}' > "$tmpdir/archive/2026-03-13-120000/state.json"

  # Run session-clear
  ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-clear --command oracle > /dev/null 2>&1

  # Verify archive directory still exists with contents
  if [[ -d "$tmpdir/archive" && -f "$tmpdir/archive/2026-03-13-120000/state.json" ]]; then
    run_test "clear_preserves_archive" "archive preserved" "archive preserved"
  else
    run_test "clear_preserves_archive" "archive preserved" "archive missing or empty"
  fi

  cleanup_tmpdir "$tmpdir"
}

# ---- Test 5: research-plan.md generation from valid state ----
test_research_plan_structure() {
  local tmpdir=$(setup_tmpdir)

  # Create valid state.json and plan.json
  cat > "$tmpdir/state.json" <<'STATE_EOF'
{
  "version": "1.0",
  "topic": "How caching works",
  "scope": "codebase",
  "phase": "survey",
  "iteration": 2,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 40,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T01:00:00Z",
  "status": "active"
}
STATE_EOF

  cat > "$tmpdir/plan.json" <<'PLAN_EOF'
{
  "version": "1.0",
  "questions": [
    {"id": "q1", "text": "What caching layers exist?", "status": "partial", "confidence": 60, "key_findings": ["Redis used"], "iterations_touched": [1]},
    {"id": "q2", "text": "How are cache keys generated?", "status": "open", "confidence": 0, "key_findings": [], "iterations_touched": []}
  ],
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T01:00:00Z"
}
PLAN_EOF

  # Generate research-plan.md inline (replicating the wizard's generation logic)
  local topic iteration max_iter confidence
  topic=$(jq -r '.topic' "$tmpdir/state.json")
  iteration=$(jq -r '.iteration' "$tmpdir/state.json")
  max_iter=$(jq -r '.max_iterations' "$tmpdir/state.json")
  confidence=$(jq -r '.overall_confidence' "$tmpdir/state.json")

  {
    echo "# Research Plan"
    echo ""
    echo "**Topic:** $topic"
    echo "**Status:** active | **Iteration:** $iteration of $max_iter"
    echo "**Overall Confidence:** ${confidence}%"
    echo ""
    echo "## Questions"
    echo "| # | Question | Status | Confidence |"
    echo "|---|----------|--------|------------|"
    jq -r '.questions[] | "| \(.id) | \(.text) | \(.status) | \(.confidence)% |"' "$tmpdir/plan.json"
    echo ""
    echo "## Next Steps"
    local next_q
    next_q=$(jq -r '[.questions[] | select(.status != "answered")][0].text // "All questions answered"' "$tmpdir/plan.json")
    echo "Next investigation: $next_q"
    echo ""
    echo "---"
    echo "*Generated from plan.json -- do not edit directly*"
  } > "$tmpdir/research-plan.md"

  # Verify the generated file has expected content
  local content
  content=$(cat "$tmpdir/research-plan.md")

  run_test "research_plan_has_topic" "How caching works" "$content"
  run_test "research_plan_has_table_header" "| # | Question | Status | Confidence |" "$content"
  run_test "research_plan_has_question" "What caching layers exist?" "$content"
  run_test "research_plan_has_next_steps" "Next investigation:" "$content"

  # Validate state files pass jq validation after simulated update
  # (Simulating what happens after an iteration updates state.json)
  jq '.iteration = 3 | .overall_confidence = 55 | .last_updated = "2026-03-13T02:00:00Z"' "$tmpdir/state.json" > "$tmpdir/state_updated.json"
  mv "$tmpdir/state_updated.json" "$tmpdir/state.json"

  local validate_result
  validate_result=$(ORACLE_DIR="$tmpdir" bash "$UTILS_SCRIPT" validate-oracle-state all 2>/dev/null)
  local all_pass
  all_pass=$(echo "$validate_result" | jq -r '.result.pass')
  run_test "research_plan_state_valid_after_update" "true" "$all_pass"

  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle State File Test Suite"
echo "========================================="
echo ""

test_verify_fresh_finds_new_files
test_verify_fresh_reports_missing
test_clear_removes_oracle_files
test_clear_preserves_archive
test_research_plan_structure

echo ""
echo "========================================="
echo "Oracle State Tests: $TESTS_PASSED passed, $TESTS_FAILED failed out of $TESTS_RUN"
echo "========================================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
