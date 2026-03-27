#!/bin/bash
# Test suite for oracle colony promotion and template-aware synthesis
# Phase 11 Plan 03 -- validates promote_to_colony end-to-end, build_synthesis_prompt
# template branches, and validate-oracle-state template field

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
    echo "  Expected to contain: $expected"
    echo "  Actual: $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# Helper: Run negative test (expected NOT to contain)
run_test_not() {
  local name="$1"
  local not_expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" != *"$not_expected"* ]]; then
    echo -e "${GREEN}PASS${NC}: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo -e "${RED}FAIL${NC}: $name"
    echo "  Expected NOT to contain: $not_expected"
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

# Helper: Write state.json with template support
write_state() {
  local dir="$1"
  local status="${2:-complete}"
  local template="${3:-custom}"
  jq -n \
    --arg status "$status" \
    --arg template "$template" \
    '{
      version: "1.1",
      topic: "Test topic",
      scope: "both",
      phase: "verify",
      iteration: 5,
      max_iterations: 15,
      target_confidence: 95,
      overall_confidence: 85,
      started_at: "2026-03-13T00:00:00Z",
      last_updated: "2026-03-13T00:00:00Z",
      status: $status,
      strategy: "adaptive",
      focus_areas: [],
      template: $template
    }' > "$dir/state.json"
}

# Helper: Write plan.json with structured findings
write_plan_with_findings() {
  local dir="$1"
  local questions_json="$2"
  jq -n \
    --argjson q "$questions_json" \
    '{version: "1.1", sources: {}, questions: $q, created_at: "2026-03-13T00:00:00Z", last_updated: "2026-03-13T00:00:00Z"}' \
    > "$dir/plan.json"
}

# Helper: Create minimal colony state
write_colony_state() {
  local dir="$1"
  mkdir -p "$dir/.aether/data"
  jq -n '{
    goal: "test colony",
    state: "active",
    current_phase: 1,
    plan: {id: "test-plan", tasks: []},
    memory: {},
    errors: {records: []},
    events: [],
    session_id: "test-session",
    initialized_at: "2026-03-13T00:00:00Z"
  }' > "$dir/.aether/data/COLONY_STATE.json"
}

# Helper: Create mock aether-utils.sh that logs calls
write_mock_utils() {
  local dir="$1"
  mkdir -p "$dir/.aether"
  cat > "$dir/.aether/aether-utils.sh" <<MOCKEOF
#!/bin/bash
echo "\$@" >> "$dir/promotion-log.txt"
echo '{"ok":true}'
MOCKEOF
  chmod +x "$dir/.aether/aether-utils.sh"
}


# ---- Test Suite 1: promote_to_colony integration ----
test_promote_to_colony() {
  echo ""
  echo "--- promote_to_colony integration tests ---"

  # Test 1: Qualifying findings are promoted
  local tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is A?","status":"answered","confidence":85,"key_findings":[{"text":"Finding A","source_ids":["s1"],"iteration":2}],"iterations_touched":[1,2]},
    {"id":"q2","text":"What is B?","status":"answered","confidence":90,"key_findings":[{"text":"Finding B","source_ids":["s1","s2"],"iteration":3}],"iterations_touched":[1,2,3]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  local output
  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " 2>/dev/null)

  run_test "promote_qualifying_findings" "Promoting 2 high-confidence findings" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 2: No qualifying findings message
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is C?","status":"partial","confidence":70,"key_findings":[{"text":"Below threshold","source_ids":["s1"],"iteration":1}],"iterations_touched":[1]},
    {"id":"q2","text":"What is D?","status":"open","confidence":40,"key_findings":[],"iterations_touched":[1]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " 2>/dev/null)

  run_test "no_qualifying_findings" "No findings meet promotion threshold" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 3: Active status blocks promotion
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "active"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is E?","status":"answered","confidence":90,"key_findings":[{"text":"Active finding","source_ids":["s1"],"iteration":1}],"iterations_touched":[1]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " 2>/dev/null)

  run_test "active_status_blocks" "ERROR" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 4: instinct-create is called
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is F?","status":"answered","confidence":85,"key_findings":[{"text":"Instinct finding","source_ids":["s1"],"iteration":1}],"iterations_touched":[1]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " >/dev/null 2>/dev/null

  local log_content=""
  if [ -f "$tmpdir/promotion-log.txt" ]; then
    log_content=$(cat "$tmpdir/promotion-log.txt")
  fi
  run_test "calls_instinct_create" "instinct-create" "$log_content"
  cleanup_tmpdir "$tmpdir"

  # Test 5: learning-promote is called
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is G?","status":"answered","confidence":85,"key_findings":[{"text":"Learning finding","source_ids":["s1"],"iteration":1}],"iterations_touched":[1]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " >/dev/null 2>/dev/null

  log_content=""
  if [ -f "$tmpdir/promotion-log.txt" ]; then
    log_content=$(cat "$tmpdir/promotion-log.txt")
  fi
  run_test "calls_learning_promote" "learning-promote" "$log_content"
  cleanup_tmpdir "$tmpdir"

  # Test 6: memory-capture is called
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete"
  write_plan_with_findings "$tmpdir" '[
    {"id":"q1","text":"What is H?","status":"answered","confidence":85,"key_findings":[{"text":"Memory finding","source_ids":["s1"],"iteration":1}],"iterations_touched":[1]}
  ]'
  write_colony_state "$tmpdir"
  write_mock_utils "$tmpdir"

  bash -c "
    set +e
    eval \"\$(sed -n '/^promote_to_colony()/,/^}/p' '$ORACLE_SH')\"
    promote_to_colony '$tmpdir/plan.json' '$tmpdir/state.json' '$tmpdir'
  " >/dev/null 2>/dev/null

  log_content=""
  if [ -f "$tmpdir/promotion-log.txt" ]; then
    log_content=$(cat "$tmpdir/promotion-log.txt")
  fi
  run_test "calls_memory_capture" "memory-capture" "$log_content"
  cleanup_tmpdir "$tmpdir"
}


# ---- Test Suite 2: build_synthesis_prompt templates ----
test_build_synthesis_prompt_templates() {
  echo ""
  echo "--- build_synthesis_prompt template tests ---"

  # Test 7: tech-eval template
  local tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete" "tech-eval"

  local output
  output=$(bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
    eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_synthesis_prompt 'converged'
  " 2>/dev/null)

  run_test "tech_eval_comparison_matrix" "Comparison Matrix" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 8: architecture-review template
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete" "architecture-review"

  output=$(bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
    eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_synthesis_prompt 'converged'
  " 2>/dev/null)

  run_test "architecture_review_component_map" "Component Map" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 9: bug-investigation template
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete" "bug-investigation"

  output=$(bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
    eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_synthesis_prompt 'converged'
  " 2>/dev/null)

  run_test "bug_investigation_root_cause" "Root Cause Analysis" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 10: best-practices template
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete" "best-practices"

  output=$(bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
    eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_synthesis_prompt 'converged'
  " 2>/dev/null)

  run_test "best_practices_gap_analysis" "Gap Analysis" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 11: custom template
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "complete" "custom"

  output=$(bash -c "
    set +e
    STATE_FILE='$tmpdir/state.json'
    SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
    eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_synthesis_prompt 'converged'
  " 2>/dev/null)

  run_test "custom_findings_by_question" "Findings by Question" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 12: All templates include confidence grouping
  local templates=("tech-eval" "architecture-review" "bug-investigation" "best-practices" "custom")
  local all_have_grouping=true
  tmpdir=$(setup_tmpdir)

  for tmpl in "${templates[@]}"; do
    write_state "$tmpdir" "complete" "$tmpl"

    output=$(bash -c "
      set +e
      STATE_FILE='$tmpdir/state.json'
      SCRIPT_DIR='$AETHER_ROOT/.aether/utils/oracle'
      eval \"\$(sed -n '/^build_synthesis_prompt()/,/^}/p' '$ORACLE_SH')\"
      build_synthesis_prompt 'converged'
    " 2>/dev/null)

    if [[ "$output" != *"Confidence Grouping"* ]]; then
      all_have_grouping=false
      break
    fi
  done

  run_test "all_templates_confidence_grouping" "true" "$all_have_grouping"
  cleanup_tmpdir "$tmpdir"
}


# ---- Test Suite 3: validate-oracle-state template field ----
test_validate_oracle_state_template() {
  echo ""
  echo "--- validate-oracle-state template tests ---"

  # Test 13: Valid template values accepted
  local tmpdir=$(setup_tmpdir)
  local valid_templates=("tech-eval" "architecture-review" "bug-investigation" "best-practices" "custom")
  local all_pass=true

  for tmpl in "${valid_templates[@]}"; do
    write_state "$tmpdir" "active" "$tmpl"
    local result
    result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
    local pass_val
    pass_val=$(echo "$result" | tr -d ' ' | grep -o '"pass":true')
    if [[ -z "$pass_val" ]]; then
      all_pass=false
      echo "  Template '$tmpl' did not pass"
      break
    fi
  done

  run_test "valid_template_values_accepted" "true" "$all_pass"
  cleanup_tmpdir "$tmpdir"

  # Test 14: Invalid template rejected
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "active" "invalid"

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
  run_test "invalid_template_rejected" '"pass":false' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"

  # Test 15: No template field -- backward compat
  tmpdir=$(setup_tmpdir)
  jq -n '{
    version: "1.1",
    topic: "Test",
    scope: "both",
    phase: "survey",
    iteration: 0,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 0,
    started_at: "2026-03-13T00:00:00Z",
    last_updated: "2026-03-13T00:00:00Z",
    status: "active"
  }' > "$tmpdir/state.json"

  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
  run_test "no_template_backward_compat" '"pass":true' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle Colony Integration Test Suite"
echo "========================================="

test_promote_to_colony
test_build_synthesis_prompt_templates
test_validate_oracle_state_template

echo ""
echo "==========================="
echo "$TESTS_PASSED / $TESTS_RUN passed ($TESTS_FAILED failed)"
echo "==========================="

if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
