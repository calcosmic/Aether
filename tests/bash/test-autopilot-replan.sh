#!/usr/bin/env bash
# Autopilot Replan Trigger Tests
# Tests autopilot-check-replan subcommand

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
AETHER_UTILS_SOURCE="$PROJECT_ROOT/.aether/aether-utils.sh"

# Source test helpers
source "$SCRIPT_DIR/test-helpers.sh"

# Verify jq is available
require_jq

# Verify aether-utils.sh exists
if [[ ! -f "$AETHER_UTILS_SOURCE" ]]; then
    log_error "aether-utils.sh not found at: $AETHER_UTILS_SOURCE"
    exit 1
fi

# ============================================================================
# Helper: Create isolated test environment with aether-utils.sh
# ============================================================================
setup_isolated_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data"

    # Copy aether-utils.sh to temp location so it uses temp data dir
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    # Copy utils directory if it exists (needed for acquire_lock, etc.)
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    # Copy exchange directory if it exists
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Copy schemas directory if it exists
    local schemas_source="$(dirname "$AETHER_UTILS_SOURCE")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# Helper: Create a COLONY_STATE.json with specified phase_learnings count
create_colony_state() {
    local tmp_dir="$1"
    local learnings_count="${2:-0}"

    local learnings_array="[]"
    if [[ "$learnings_count" -gt 0 ]]; then
        learnings_array="["
        for i in $(seq 1 "$learnings_count"); do
            if [[ $i -gt 1 ]]; then
                learnings_array+=","
            fi
            learnings_array+="{\"id\":\"learning_$i\",\"phase\":$i,\"phase_name\":\"Phase $i\",\"learnings\":[{\"claim\":\"test claim $i\",\"status\":\"validated\"}]}"
        done
        learnings_array+="]"
    fi

    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << EOF
{
  "goal": "test goal",
  "state": "active",
  "current_phase": 3,
  "plan": {"id": "test-plan", "phases": []},
  "memory": {
    "decisions": [],
    "instincts": [],
    "phase_learnings": $learnings_array
  },
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF
}

# ============================================================================
# Test: autopilot-check-replan returns should_replan=false when under interval
# ============================================================================
test_check_replan_no_trigger() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 0

    # Init autopilot with 6 phases, start at 1
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # Only 1 advance (under default interval of 2)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "false" ]]; then
        test_fail "should_replan=false" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan returns should_replan=true at interval
# ============================================================================
test_check_replan_triggers_at_interval() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 2

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # 2 advances (hits default interval of 2)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "true" ]]; then
        test_fail "should_replan=true" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan with custom interval
# ============================================================================
test_check_replan_custom_interval() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 3

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 9 --start-phase 1 2>&1 >/dev/null

    # 3 advances (hits custom interval of 3)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 4 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan --interval 3 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "true" ]]; then
        test_fail "should_replan=true (interval=3, completed=3)" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan custom interval - not triggered
# ============================================================================
test_check_replan_custom_interval_no_trigger() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 2

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 9 --start-phase 1 2>&1 >/dev/null

    # 2 advances (under custom interval of 3)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan --interval 3 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "false" ]]; then
        test_fail "should_replan=false (interval=3, completed=2)" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan counts learnings from COLONY_STATE
# ============================================================================
test_check_replan_counts_learnings() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 5

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # 2 advances
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local learnings_since_last
    learnings_since_last=$(echo "$output" | jq -r '.result.learnings_since_last')

    if [[ "$learnings_since_last" != "5" ]]; then
        test_fail "learnings_since_last=5" "learnings_since_last=$learnings_since_last"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan includes reason when triggered
# ============================================================================
test_check_replan_includes_reason() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 3

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # 2 advances
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local reason
    reason=$(echo "$output" | jq -r '.result.reason')

    # Reason should mention the number of phases
    if [[ "$reason" == "null" || -z "$reason" ]]; then
        test_fail "reason is non-empty" "reason=$reason"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Reason should contain "2" (the interval/count)
    if ! assert_contains "$reason" "2"; then
        test_fail "reason contains phase count" "reason=$reason"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan fails when no run-state.json
# ============================================================================
test_check_replan_no_state() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 0

    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit" "exit code 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Should return JSON error
    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan at 4 advances (second trigger at interval 2)
# ============================================================================
test_check_replan_second_trigger() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 4

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 8 --start-phase 1 2>&1 >/dev/null

    # 4 advances (second trigger at interval 2: 2, 4, 6...)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 4 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 5 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "true" ]]; then
        test_fail "should_replan=true (4 advances, interval 2)" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan at 3 advances (not triggered at interval 2)
# ============================================================================
test_check_replan_odd_no_trigger() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 3

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 8 --start-phase 1 2>&1 >/dev/null

    # 3 advances (not a multiple of 2)
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 4 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "false" ]]; then
        test_fail "should_replan=false (3 advances, interval 2)" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-check-replan returns valid JSON with all required fields
# ============================================================================
test_check_replan_response_schema() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 2

    # Init autopilot
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # 2 advances
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 3 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Check all required fields exist in result
    local has_should_replan has_reason has_learnings
    has_should_replan=$(echo "$output" | jq '.result | has("should_replan")')
    has_reason=$(echo "$output" | jq '.result | has("reason")')
    has_learnings=$(echo "$output" | jq '.result | has("learnings_since_last")')

    local failed=false
    [[ "$has_should_replan" == "true" ]] || { test_fail "has should_replan field" "missing"; failed=true; }
    [[ "$has_reason" == "true" ]] || { test_fail "has reason field" "missing"; failed=true; }
    [[ "$has_learnings" == "true" ]] || { test_fail "has learnings_since_last field" "missing"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-check-replan appears in help output
# ============================================================================
test_check_replan_in_help() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)

    local has_check_replan
    has_check_replan=$(echo "$output" | jq '.commands | index("autopilot-check-replan") != null')

    if [[ "$has_check_replan" != "true" ]]; then
        test_fail "autopilot-check-replan in commands" "not found"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: autopilot-check-replan zero advances returns false
# ============================================================================
test_check_replan_zero_advances() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)
    create_colony_state "$tmp_dir" 0

    # Init autopilot but no advances
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-check-replan 2>&1)

    local should_replan
    should_replan=$(echo "$output" | jq -r '.result.should_replan')

    if [[ "$should_replan" != "false" ]]; then
        test_fail "should_replan=false (0 advances)" "should_replan=$should_replan"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Main test runner
# ============================================================================
main() {
    log "${YELLOW}=== Autopilot Replan Trigger Tests ===${NC}"

    # Core replan logic tests
    run_test "test_check_replan_no_trigger" "check-replan returns false when under interval"
    run_test "test_check_replan_triggers_at_interval" "check-replan returns true at default interval (2)"
    run_test "test_check_replan_custom_interval" "check-replan respects custom interval (3)"
    run_test "test_check_replan_custom_interval_no_trigger" "check-replan custom interval - no trigger"
    run_test "test_check_replan_second_trigger" "check-replan triggers at second multiple (4)"
    run_test "test_check_replan_odd_no_trigger" "check-replan does not trigger at non-multiple (3)"
    run_test "test_check_replan_zero_advances" "check-replan returns false with zero advances"

    # Response shape tests
    run_test "test_check_replan_response_schema" "check-replan response has all required fields"
    run_test "test_check_replan_counts_learnings" "check-replan counts learnings from COLONY_STATE"
    run_test "test_check_replan_includes_reason" "check-replan includes descriptive reason"

    # Error handling
    run_test "test_check_replan_no_state" "check-replan fails when no run-state.json"

    # Help registration
    run_test "test_check_replan_in_help" "autopilot-check-replan appears in help output"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
