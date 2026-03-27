#!/usr/bin/env bash
# Autopilot State Tracking Tests
# Tests autopilot-init, autopilot-update, autopilot-status, autopilot-stop subcommands

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

    # Copy exchange directory if it exists (needed for XML functions)
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Copy schemas directory if it exists (needed for XML validation)
    local schemas_source="$(dirname "$AETHER_UTILS_SOURCE")/schemas"
    if [[ -d "$schemas_source" ]]; then
        cp -r "$schemas_source" "$tmp_dir/.aether/"
    fi

    echo "$tmp_dir"
}

# ============================================================================
# Test: autopilot-init creates run-state.json with correct schema
# ============================================================================
test_autopilot_init_creates_file() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify run-state.json was created
    if [[ ! -f "$tmp_dir/.aether/data/run-state.json" ]]; then
        test_fail "run-state.json created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-init creates correct schema fields
# ============================================================================
test_autopilot_init_schema() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 2 2>&1 >/dev/null

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")

    # Check all required fields
    local version status total_phases start_phase current_phase max_phases pause_reason last_action phases_completed total_auto_advanced
    version=$(echo "$state" | jq -r '.version')
    status=$(echo "$state" | jq -r '.status')
    total_phases=$(echo "$state" | jq -r '.total_phases')
    start_phase=$(echo "$state" | jq -r '.start_phase')
    current_phase=$(echo "$state" | jq -r '.current_phase')
    max_phases=$(echo "$state" | jq -r '.max_phases')
    pause_reason=$(echo "$state" | jq -r '.pause_reason')
    last_action=$(echo "$state" | jq -r '.last_action')
    phases_completed=$(echo "$state" | jq -r '.phases_completed_in_run')
    total_auto_advanced=$(echo "$state" | jq -r '.total_auto_advanced')

    local failed=false

    [[ "$version" == "1.0" ]] || { test_fail "version=1.0" "version=$version"; failed=true; }
    [[ "$status" == "running" ]] || { test_fail "status=running" "status=$status"; failed=true; }
    [[ "$total_phases" == "6" ]] || { test_fail "total_phases=6" "total_phases=$total_phases"; failed=true; }
    [[ "$start_phase" == "2" ]] || { test_fail "start_phase=2" "start_phase=$start_phase"; failed=true; }
    [[ "$current_phase" == "2" ]] || { test_fail "current_phase=2" "current_phase=$current_phase"; failed=true; }
    [[ "$max_phases" == "null" ]] || { test_fail "max_phases=null" "max_phases=$max_phases"; failed=true; }
    [[ "$pause_reason" == "null" ]] || { test_fail "pause_reason=null" "pause_reason=$pause_reason"; failed=true; }
    [[ "$last_action" == "null" ]] || { test_fail "last_action=null" "last_action=$last_action"; failed=true; }
    [[ "$phases_completed" == "0" ]] || { test_fail "phases_completed_in_run=0" "phases_completed_in_run=$phases_completed"; failed=true; }
    [[ "$total_auto_advanced" == "0" ]] || { test_fail "total_auto_advanced=0" "total_auto_advanced=$total_auto_advanced"; failed=true; }

    # Verify started_at is an ISO-8601 timestamp
    local started_at
    started_at=$(echo "$state" | jq -r '.started_at')
    if [[ ! "$started_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T ]]; then
        test_fail "started_at is ISO-8601" "started_at=$started_at"
        failed=true
    fi

    # Verify phase_results is empty array
    local phase_results_len
    phase_results_len=$(echo "$state" | jq '.phase_results | length')
    [[ "$phase_results_len" == "0" ]] || { test_fail "phase_results empty" "length=$phase_results_len"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-init with --max-phases option
# ============================================================================
test_autopilot_init_max_phases() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 --max-phases 3 2>&1 >/dev/null

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local max_phases
    max_phases=$(echo "$state" | jq -r '.max_phases')

    if [[ "$max_phases" != "3" ]]; then
        test_fail "max_phases=3" "max_phases=$max_phases"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-update increments phases_completed_in_run
# ============================================================================
test_autopilot_update_increments() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # Update with action=advance, phase=2 (only advance increments phases_completed)
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1)
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

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local phases_completed current_phase last_action
    phases_completed=$(echo "$state" | jq -r '.phases_completed_in_run')
    current_phase=$(echo "$state" | jq -r '.current_phase')
    last_action=$(echo "$state" | jq -r '.last_action')

    local failed=false
    [[ "$phases_completed" == "1" ]] || { test_fail "phases_completed=1" "phases_completed=$phases_completed"; failed=true; }
    [[ "$current_phase" == "2" ]] || { test_fail "current_phase=2" "current_phase=$current_phase"; failed=true; }
    [[ "$last_action" == "advance" ]] || { test_fail "last_action=advance" "last_action=$last_action"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-update records phase result
# ============================================================================
test_autopilot_update_records_result() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # Update with phase result
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action build --phase 2 --result success 2>&1 >/dev/null

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local result_count result_phase result_action result_status
    result_count=$(echo "$state" | jq '.phase_results | length')
    result_phase=$(echo "$state" | jq -r '.phase_results[0].phase')
    result_action=$(echo "$state" | jq -r '.phase_results[0].action')
    result_status=$(echo "$state" | jq -r '.phase_results[0].result')

    local failed=false
    [[ "$result_count" == "1" ]] || { test_fail "result_count=1" "result_count=$result_count"; failed=true; }
    [[ "$result_phase" == "2" ]] || { test_fail "result_phase=2" "result_phase=$result_phase"; failed=true; }
    [[ "$result_action" == "build" ]] || { test_fail "result_action=build" "result_action=$result_action"; failed=true; }
    [[ "$result_status" == "success" ]] || { test_fail "result_status=success" "result_status=$result_status"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-update fails when no run-state.json exists
# ============================================================================
test_autopilot_update_no_state() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action build --phase 2 2>&1)
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
# Test: autopilot-update increments total_auto_advanced on advance action
# ============================================================================
test_autopilot_update_auto_advanced() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    # Update with action=advance
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-update --action advance --phase 2 2>&1 >/dev/null

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local total_auto_advanced
    total_auto_advanced=$(echo "$state" | jq -r '.total_auto_advanced')

    if [[ "$total_auto_advanced" != "1" ]]; then
        test_fail "total_auto_advanced=1" "total_auto_advanced=$total_auto_advanced"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-status returns current state
# ============================================================================
test_autopilot_status() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-status 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code — output: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify result contains expected fields
    local status current_phase
    status=$(echo "$output" | jq -r '.result.status')
    current_phase=$(echo "$output" | jq -r '.result.current_phase')

    local failed=false
    [[ "$status" == "running" ]] || { test_fail "status=running" "status=$status"; failed=true; }
    [[ "$current_phase" == "1" ]] || { test_fail "current_phase=1" "current_phase=$current_phase"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-status returns not_active when no run-state.json
# ============================================================================
test_autopilot_status_no_state() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-status 2>&1)
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

    local status
    status=$(echo "$output" | jq -r '.result.status')
    if [[ "$status" != "not_active" ]]; then
        test_fail "status=not_active" "status=$status"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-stop sets status to stopped with reason
# ============================================================================
test_autopilot_stop() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-stop --reason "user requested stop" 2>&1)
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

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local status pause_reason
    status=$(echo "$state" | jq -r '.status')
    pause_reason=$(echo "$state" | jq -r '.pause_reason')

    local failed=false
    [[ "$status" == "stopped" ]] || { test_fail "status=stopped" "status=$status"; failed=true; }
    [[ "$pause_reason" == "user requested stop" ]] || { test_fail "pause_reason='user requested stop'" "pause_reason=$pause_reason"; failed=true; }

    rm -rf "$tmp_dir"
    [[ "$failed" == "false" ]]
}

# ============================================================================
# Test: autopilot-stop fails when no run-state.json
# ============================================================================
test_autopilot_stop_no_state() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    set +e
    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-stop --reason "test" 2>&1)
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit" "exit code 0"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: autopilot-stop with completed status
# ============================================================================
test_autopilot_stop_completed() {
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Init first
    bash "$tmp_dir/.aether/aether-utils.sh" autopilot-init --total-phases 6 --start-phase 1 2>&1 >/dev/null

    local output
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" autopilot-stop --reason "all phases done" --status completed 2>&1)

    local state
    state=$(cat "$tmp_dir/.aether/data/run-state.json")
    local status
    status=$(echo "$state" | jq -r '.status')

    if [[ "$status" != "completed" ]]; then
        test_fail "status=completed" "status=$status"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: existing subcommands still work (regression)
# ============================================================================
test_existing_help_unaffected() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: autopilot commands appear in help output
# ============================================================================
test_autopilot_in_help() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" help 2>&1)

    local has_init has_update has_status has_stop
    has_init=$(echo "$output" | jq '.commands | index("autopilot-init") != null')
    has_update=$(echo "$output" | jq '.commands | index("autopilot-update") != null')
    has_status=$(echo "$output" | jq '.commands | index("autopilot-status") != null')
    has_stop=$(echo "$output" | jq '.commands | index("autopilot-stop") != null')

    local failed=false
    [[ "$has_init" == "true" ]] || { test_fail "autopilot-init in commands" "not found"; failed=true; }
    [[ "$has_update" == "true" ]] || { test_fail "autopilot-update in commands" "not found"; failed=true; }
    [[ "$has_status" == "true" ]] || { test_fail "autopilot-status in commands" "not found"; failed=true; }
    [[ "$has_stop" == "true" ]] || { test_fail "autopilot-stop in commands" "not found"; failed=true; }

    [[ "$failed" == "false" ]]
}

# ============================================================================
# Main test runner
# ============================================================================
main() {
    log "${YELLOW}=== Autopilot State Tracking Tests ===${NC}"

    # autopilot-init tests
    run_test "test_autopilot_init_creates_file" "autopilot-init creates run-state.json"
    run_test "test_autopilot_init_schema" "autopilot-init creates correct schema fields"
    run_test "test_autopilot_init_max_phases" "autopilot-init respects --max-phases option"

    # autopilot-update tests
    run_test "test_autopilot_update_increments" "autopilot-update increments phases_completed_in_run"
    run_test "test_autopilot_update_records_result" "autopilot-update records phase result"
    run_test "test_autopilot_update_no_state" "autopilot-update fails when no run-state.json"
    run_test "test_autopilot_update_auto_advanced" "autopilot-update increments total_auto_advanced on advance"

    # autopilot-status tests
    run_test "test_autopilot_status" "autopilot-status returns current state"
    run_test "test_autopilot_status_no_state" "autopilot-status returns not_active when no state"

    # autopilot-stop tests
    run_test "test_autopilot_stop" "autopilot-stop sets status to stopped with reason"
    run_test "test_autopilot_stop_no_state" "autopilot-stop fails when no run-state.json"
    run_test "test_autopilot_stop_completed" "autopilot-stop supports completed status"

    # Regression tests
    run_test "test_existing_help_unaffected" "existing help subcommand still works"
    run_test "test_autopilot_in_help" "autopilot commands appear in help output"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
