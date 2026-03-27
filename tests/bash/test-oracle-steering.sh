#!/bin/bash
# Test suite for oracle steering integration functions
# Phase 10 Plan 02 -- validates read_steering_signals, build_oracle_prompt strategy,
# and validate-oracle-state steering extensions

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

# Helper: Write state.json with strategy and focus_areas support
write_state() {
  local dir="$1"
  local phase="${2:-survey}"
  local iteration="${3:-0}"
  local strategy="${4:-adaptive}"
  local focus_areas_json="${5:-[]}"
  jq -n \
    --arg phase "$phase" \
    --argjson iteration "$iteration" \
    --arg strategy "$strategy" \
    --argjson focus "$focus_areas_json" \
    '{
      version: "1.1",
      topic: "Test topic",
      scope: "both",
      phase: $phase,
      iteration: $iteration,
      max_iterations: 15,
      target_confidence: 95,
      overall_confidence: 0,
      started_at: "2026-03-13T00:00:00Z",
      last_updated: "2026-03-13T00:00:00Z",
      status: "active",
      strategy: $strategy,
      focus_areas: $focus
    }' > "$dir/state.json"
}

# Helper: Write plan.json
write_plan() {
  local dir="$1"
  local questions_json="$2"
  jq -n \
    --argjson q "$questions_json" \
    '{version: "1.1", sources: {}, questions: $q, created_at: "2026-03-13T00:00:00Z", last_updated: "2026-03-13T00:00:00Z"}' \
    > "$dir/plan.json"
}

# Helper: Write mock aether-utils.sh that responds to pheromone-read
write_mock_utils() {
  local dir="$1"
  local signals_json="$2"
  local aether_dir="$dir/.aether"
  mkdir -p "$aether_dir"

  # Create mock script that returns signals for pheromone-read
  cat > "$aether_dir/aether-utils.sh" <<MOCKEOF
#!/bin/bash
case "\$1" in
  pheromone-read) echo '{"ok":true,"result":{"signals":$signals_json}}';;
esac
MOCKEOF
  chmod +x "$aether_dir/aether-utils.sh"
}


# ---- Test Suite 1: read_steering_signals ----
test_read_steering_signals() {
  echo ""
  echo "--- read_steering_signals tests ---"

  # Test 1: No signals returns empty
  local tmpdir=$(setup_tmpdir)
  write_mock_utils "$tmpdir" '[]'

  local output
  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^read_steering_signals()/,/^}/p' '$ORACLE_SH')\"
    read_steering_signals '$tmpdir'
  " 2>/dev/null)

  run_test "no_signals_returns_empty" "" "$output"
  # Verify it is actually empty (not just whitespace)
  local trimmed
  trimmed=$(echo "$output" | tr -d '[:space:]')
  run_test "no_signals_truly_empty" "" "$trimmed"
  cleanup_tmpdir "$tmpdir"

  # Test 2: FOCUS formatting
  tmpdir=$(setup_tmpdir)
  write_mock_utils "$tmpdir" '[
    {"id":"sig_1","type":"FOCUS","content":{"text":"Security review"},"effective_strength":0.9,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"},
    {"id":"sig_2","type":"FOCUS","content":{"text":"Performance tuning"},"effective_strength":0.7,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"}
  ]'

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^read_steering_signals()/,/^}/p' '$ORACLE_SH')\"
    read_steering_signals '$tmpdir'
  " 2>/dev/null)

  run_test "focus_has_header" "FOCUS (Prioritize these areas)" "$output"
  run_test "focus_has_signal_text_1" "Security review" "$output"
  run_test "focus_has_signal_text_2" "Performance tuning" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 3: REDIRECT formatting
  tmpdir=$(setup_tmpdir)
  write_mock_utils "$tmpdir" '[
    {"id":"sig_r1","type":"REDIRECT","content":{"text":"Avoid deprecated v2 API"},"effective_strength":0.95,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"}
  ]'

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^read_steering_signals()/,/^}/p' '$ORACLE_SH')\"
    read_steering_signals '$tmpdir'
  " 2>/dev/null)

  run_test "redirect_has_header" "REDIRECT (Hard constraints -- MUST follow)" "$output"
  run_test "redirect_has_signal_text" "Avoid deprecated v2 API" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 4: Signal limit enforcement (max 3 FOCUS)
  tmpdir=$(setup_tmpdir)
  write_mock_utils "$tmpdir" '[
    {"id":"sig_a","type":"FOCUS","content":{"text":"Area Alpha"},"effective_strength":0.9,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"},
    {"id":"sig_b","type":"FOCUS","content":{"text":"Area Beta"},"effective_strength":0.8,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"},
    {"id":"sig_c","type":"FOCUS","content":{"text":"Area Gamma"},"effective_strength":0.7,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"},
    {"id":"sig_d","type":"FOCUS","content":{"text":"Area Delta"},"effective_strength":0.6,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"},
    {"id":"sig_e","type":"FOCUS","content":{"text":"Area Epsilon"},"effective_strength":0.5,"created_at":"2026-03-13T00:00:00Z","expires_at":"2026-03-14T00:00:00Z"}
  ]'

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^read_steering_signals()/,/^}/p' '$ORACLE_SH')\"
    read_steering_signals '$tmpdir'
  " 2>/dev/null)

  # Count FOCUS signal lines (each starts with "- [")
  local focus_count
  focus_count=$(echo "$output" | grep -c '^\- \[')
  run_test "signal_limit_max_3" "3" "$focus_count"
  run_test_not "signal_limit_excludes_delta" "Area Delta" "$output"
  run_test_not "signal_limit_excludes_epsilon" "Area Epsilon" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 5: Graceful degradation with nonexistent path
  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^read_steering_signals()/,/^}/p' '$ORACLE_SH')\"
    read_steering_signals '/nonexistent/path/does/not/exist'
  " 2>/dev/null)
  local exit_code=$?

  trimmed=$(echo "$output" | tr -d '[:space:]')
  run_test "graceful_degradation_empty" "" "$trimmed"
  cleanup_tmpdir "$tmpdir"
}


# ---- Test Suite 2: build_oracle_prompt strategy ----
test_build_oracle_prompt_strategy() {
  echo ""
  echo "--- build_oracle_prompt strategy tests ---"

  # Test 1: Breadth-first modifier
  local tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "survey" 0 "breadth-first"
  echo "BASE_PROMPT_MARKER" > "$tmpdir/oracle.md"

  local output
  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^build_oracle_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_oracle_prompt '$tmpdir/state.json' '$tmpdir/oracle.md'
  " 2>/dev/null)

  run_test "breadth_first_modifier" "Breadth-first" "$output"
  run_test "breadth_first_strategy_note" "STRATEGY NOTE" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 2: Depth-first modifier
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "investigate" 3 "depth-first"
  echo "BASE_PROMPT_MARKER" > "$tmpdir/oracle.md"

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^build_oracle_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_oracle_prompt '$tmpdir/state.json' '$tmpdir/oracle.md'
  " 2>/dev/null)

  run_test "depth_first_modifier" "Depth-first" "$output"
  run_test "depth_first_exhaustively" "exhaustively" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 3: Adaptive no modifier
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "survey" 0 "adaptive"
  echo "BASE_PROMPT_MARKER" > "$tmpdir/oracle.md"

  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^build_oracle_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_oracle_prompt '$tmpdir/state.json' '$tmpdir/oracle.md'
  " 2>/dev/null)

  run_test_not "adaptive_no_strategy_note" "STRATEGY NOTE" "$output"
  cleanup_tmpdir "$tmpdir"

  # Test 4: Steering directive injection
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "survey" 0 "adaptive"
  echo "BASE_PROMPT_MARKER" > "$tmpdir/oracle.md"

  local steering_text="CUSTOM STEERING: focus on database queries"
  output=$(bash -c "
    set +e
    eval \"\$(sed -n '/^build_oracle_prompt()/,/^}/p' '$ORACLE_SH')\"
    build_oracle_prompt '$tmpdir/state.json' '$tmpdir/oracle.md' '$steering_text'
  " 2>/dev/null)

  run_test "steering_directive_injected" "CUSTOM STEERING: focus on database queries" "$output"
  run_test "steering_base_prompt_present" "BASE_PROMPT_MARKER" "$output"
  cleanup_tmpdir "$tmpdir"
}


# ---- Test Suite 3: validate-oracle-state steering extensions ----
test_validate_oracle_state_steering() {
  echo ""
  echo "--- validate-oracle-state steering tests ---"

  # Test 1: Accepts strategy field
  local tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "survey" 0 "breadth-first"

  local result
  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
  run_test "accepts_strategy_field" '"pass": true' "$result"
  # Also check with alternate format
  run_test "accepts_strategy_breadth_first" '"pass":true' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"

  # Test 2: Accepts focus_areas field
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "investigate" 2 "adaptive" '["security","performance"]'

  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
  run_test "accepts_focus_areas" '"pass":true' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"

  # Test 3: Backward compat (no strategy/focus_areas)
  tmpdir=$(setup_tmpdir)
  # Write state without strategy or focus_areas
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
  run_test "backward_compat_no_strategy" '"pass":true' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"

  # Test 4: Rejects invalid strategy
  tmpdir=$(setup_tmpdir)
  write_state "$tmpdir" "survey" 0 "invalid"

  result=$(ORACLE_DIR="$tmpdir" bash "$AETHER_ROOT/.aether/aether-utils.sh" validate-oracle-state state 2>/dev/null)
  run_test "rejects_invalid_strategy" '"pass":false' "$(echo "$result" | tr -d ' ')"
  cleanup_tmpdir "$tmpdir"
}


# Run all tests
echo "========================================="
echo "Oracle Steering Integration Test Suite"
echo "========================================="

test_read_steering_signals
test_build_oracle_prompt_strategy
test_validate_oracle_state_steering

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
