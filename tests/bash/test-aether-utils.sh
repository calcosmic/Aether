#!/usr/bin/env bash
# Aether Utils Integration Tests
# Tests aether-utils.sh subcommands for valid JSON output and correct behavior

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

    echo "$tmp_dir"
}

# ============================================================================
# Test: help subcommand
# ============================================================================
test_help() {
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

    if ! assert_json_has_field "$output" "commands"; then
        test_fail "has 'commands' field" "field missing"
        return 1
    fi

    if ! assert_json_has_field "$output" "description"; then
        test_fail "has 'description' field" "field missing"
        return 1
    fi

    # Verify commands array is not empty
    local cmd_count
    cmd_count=$(echo "$output" | jq '.commands | length')
    if [[ "$cmd_count" -eq 0 ]]; then
        test_fail "non-empty commands array" "empty array"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: version subcommand
# ============================================================================
test_version() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" version 2>&1)
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

    if ! assert_json_field_equals "$output" ".result" "1.0.0"; then
        test_fail '"1.0.0"' "$(echo "$output" | jq -r '.result')"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state colony
# ============================================================================
test_validate_state_colony() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create valid COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test"},
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state colony 2>&1) || true
    rm -rf "$tmp_dir"

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state constraints
# ============================================================================
test_validate_state_constraints() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create valid constraints.json
    cat > "$tmp_dir/.aether/data/constraints.json" << 'EOF'
{
  "focus": ["testing"],
  "constraints": ["test"]
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state constraints 2>&1) || true
    rm -rf "$tmp_dir"

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: validate-state missing file
# ============================================================================
test_validate_state_missing() {
    local output
    local exit_code
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Don't create any data files - test missing file handling
    set +e
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" validate-state colony 2>&1)
    exit_code=$?
    set -e
    rm -rf "$tmp_dir"

    # Should return non-zero exit code for error
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: activity-log-init
# ============================================================================
test_activity_log_init() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" activity-log-init 1 "Test Phase" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
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

    # Verify activity.log was created
    if [[ ! -f "$tmp_dir/.aether/data/activity.log" ]]; then
        test_fail "activity.log created" "file not found"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: activity-log-read
# ============================================================================
test_activity_log_read() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create an activity log
    echo "[12:00:00] Test entry" > "$tmp_dir/.aether/data/activity.log"

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" activity-log-read 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
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

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: flag-list (empty)
# ============================================================================
test_flag_list_empty() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-list 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
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

    # Should return empty flags array
    local count
    count=$(echo "$output" | jq '.result.flags | length')
    if [[ "$count" -ne 0 ]]; then
        test_fail "0 flags" "$count flags"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: flag-add and flag-list
# ============================================================================
test_flag_add_and_list() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Add a flag
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-add issue "Test Issue" "Test description" manual 1 2>&1)

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON from flag-add" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        rm -rf "$tmp_dir"
        return 1
    fi

    # List flags
    output=$(bash "$tmp_dir/.aether/aether-utils.sh" flag-list 2>&1)

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON from flag-list" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local count
    count=$(echo "$output" | jq '.result.flags | length')
    if [[ "$count" -ne 1 ]]; then
        test_fail "1 flag" "$count flags"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: generate-ant-name
# ============================================================================
test_generate_ant_name() {
    local output
    output=$(bash "$AETHER_UTILS_SOURCE" generate-ant-name builder 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_true "$output"; then
        test_fail '{"ok":true}' "$output"
        return 1
    fi

    # Verify result is a non-empty string with expected format (Prefix-Number)
    local name
    name=$(echo "$output" | jq -r '.result')
    if [[ ! "$name" =~ ^[A-Za-z]+-[0-9]+$ ]]; then
        test_fail "name matching Pattern-Number format" "$name"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: error-summary (empty state)
# ============================================================================
test_error_summary_empty() {
    local output
    local tmp_dir
    tmp_dir=$(setup_isolated_env)

    # Create COLONY_STATE.json with empty errors
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "goal": "test",
  "state": "active",
  "current_phase": 1,
  "plan": {},
  "memory": {},
  "errors": {"records": []},
  "events": []
}
EOF

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" error-summary 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
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

    # Verify total is 0
    local total
    total=$(echo "$output" | jq '.result.total')
    if [[ "$total" -ne 0 ]]; then
        test_fail "total: 0" "total: $total"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: invalid subcommand
# ============================================================================
test_invalid_subcommand() {
    local output
    local exit_code

    set +e
    output=$(bash "$AETHER_UTILS_SOURCE" invalid-command 2>&1)
    exit_code=$?
    set -e

    # Should return non-zero exit code
    if [[ "$exit_code" -eq 0 ]]; then
        test_fail "non-zero exit code" "exit code 0"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON error" "invalid JSON: $output"
        return 1
    fi

    if ! assert_ok_false "$output"; then
        test_fail '{"ok":false}' "$output"
        return 1
    fi

    return 0
}

# ============================================================================
# Test: check-antipattern
# ============================================================================
test_check_antipattern() {
    local output
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Create a test file with a TODO
    echo "// TODO: fix this" > "$tmp_dir/test.js"

    output=$(bash "$AETHER_UTILS_SOURCE" check-antipattern "$tmp_dir/test.js" 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
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

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: bootstrap-system (requires hub)
# ============================================================================
test_bootstrap_system() {
    local output
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether"

    # Create mock hub system directory
    mkdir -p "$tmp_dir/.aether-hub/system"
    echo "# test" > "$tmp_dir/.aether-hub/system/aether-utils.sh"

    # Copy script to temp location
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"

    # Override HOME to point to mock hub
    export HOME="$tmp_dir"

    output=$(bash "$tmp_dir/.aether/aether-utils.sh" bootstrap-system 2>&1) || true

    unset HOME

    # This may fail if hub doesn't exist, that's OK - just verify JSON output
    if [[ -n "$output" ]]; then
        if ! assert_json_valid "$output"; then
            test_fail "valid JSON" "invalid JSON: $output"
            rm -rf "$tmp_dir"
            return 1
        fi
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
    log "${YELLOW}=== Aether Utils Integration Tests ===${NC}"
    log "Testing: $AETHER_UTILS_SOURCE"
    log ""

    # Run all tests
    run_test "test_help" "help returns valid JSON with commands"
    run_test "test_version" "version returns ok:true with 1.0.0"
    run_test "test_validate_state_colony" "validate-state colony validates COLONY_STATE.json"
    run_test "test_validate_state_constraints" "validate-state constraints validates constraints.json"
    run_test "test_validate_state_missing" "validate-state handles missing files"
    run_test "test_activity_log_init" "activity-log-init creates activity.log"
    run_test "test_activity_log_read" "activity-log-read returns log content"
    run_test "test_flag_list_empty" "flag-list returns empty array when no flags"
    run_test "test_flag_add_and_list" "flag-add creates flag, flag-list retrieves it"
    run_test "test_generate_ant_name" "generate-ant-name returns valid name"
    run_test "test_error_summary_empty" "error-summary with empty state"
    run_test "test_invalid_subcommand" "invalid subcommand returns error"
    run_test "test_check_antipattern" "check-antipattern analyzes files"
    run_test "test_bootstrap_system" "bootstrap-system handles missing hub gracefully"

    # Print summary
    test_summary
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
