#!/usr/bin/env bash
# State API Integration Tests
# Tests state-api.sh facade functions via aether-utils.sh subcommands

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
# Helper: Create isolated test environment
# ============================================================================
setup_state_env() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"

    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    chmod +x "$tmp_dir/.aether/aether-utils.sh"

    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    if [[ -d "$utils_source" ]]; then
        cp -r "$utils_source" "$tmp_dir/.aether/"
    fi

    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    if [[ -d "$exchange_source" ]]; then
        cp -r "$exchange_source" "$tmp_dir/.aether/"
    fi

    # Write a minimal valid COLONY_STATE.json
    cat > "$tmp_dir/.aether/data/COLONY_STATE.json" << 'EOF'
{
  "version": "3.0",
  "goal": "Test state API",
  "state": "READY",
  "current_phase": 1,
  "session_id": "test-session",
  "initialized_at": "2026-01-01T00:00:00Z",
  "build_started_at": null,
  "plan": { "phases": [{ "id": 1, "name": "Test Phase", "status": "pending" }] },
  "memory": { "phase_learnings": [], "decisions": [], "instincts": [] },
  "errors": { "records": [], "flagged_patterns": [] },
  "events": [],
  "signals": [],
  "graveyards": []
}
EOF

    echo "$tmp_dir"
}

run_cmd() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>/dev/null
}

run_cmd_stderr() {
    local tmp_dir="$1"
    shift
    AETHER_ROOT="$tmp_dir" DATA_DIR="$tmp_dir/.aether/data" bash "$tmp_dir/.aether/aether-utils.sh" "$@" 2>&1
}

# ============================================================================
# Test: state-read returns valid JSON with ok:true
# ============================================================================
test_state_read_valid() {
    local tmp_dir
    tmp_dir=$(setup_state_env)

    local output
    output=$(run_cmd "$tmp_dir" state-read)
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
        test_fail "ok:true" "ok not true"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify it contains the state data
    local goal
    goal=$(echo "$output" | jq -r '.result.goal')
    if [[ "$goal" != "Test state API" ]]; then
        test_fail "goal = 'Test state API'" "goal = '$goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: state-read returns error for missing file
# ============================================================================
test_state_read_missing() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    [[ -d "$utils_source" ]] && cp -r "$utils_source" "$tmp_dir/.aether/"
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    [[ -d "$exchange_source" ]] && cp -r "$exchange_source" "$tmp_dir/.aether/"

    # No COLONY_STATE.json -- should fail
    local output
    output=$(run_cmd_stderr "$tmp_dir" state-read) || true

    if echo "$output" | grep -q "E_FILE_NOT_FOUND"; then
        rm -rf "$tmp_dir"
        return 0
    fi

    test_fail "E_FILE_NOT_FOUND error" "$output"
    rm -rf "$tmp_dir"
    return 1
}

# ============================================================================
# Test: state-read-field extracts a specific field
# ============================================================================
test_state_read_field() {
    local tmp_dir
    tmp_dir=$(setup_state_env)

    local output
    output=$(run_cmd "$tmp_dir" state-read-field ".goal")

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON: $output"
        rm -rf "$tmp_dir"
        return 1
    fi

    local value
    value=$(echo "$output" | jq -r '.result')
    if [[ "$value" != "Test state API" ]]; then
        test_fail "result = 'Test state API'" "result = '$value'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: state-read-field returns error for missing file
# ============================================================================
test_state_read_field_missing() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    mkdir -p "$tmp_dir/.aether/data" "$tmp_dir/.aether/utils"
    cp "$AETHER_UTILS_SOURCE" "$tmp_dir/.aether/aether-utils.sh"
    local utils_source="$(dirname "$AETHER_UTILS_SOURCE")/utils"
    [[ -d "$utils_source" ]] && cp -r "$utils_source" "$tmp_dir/.aether/"
    local exchange_source="$(dirname "$AETHER_UTILS_SOURCE")/exchange"
    [[ -d "$exchange_source" ]] && cp -r "$exchange_source" "$tmp_dir/.aether/"

    local output
    output=$(run_cmd_stderr "$tmp_dir" state-read-field ".goal") || true

    if echo "$output" | grep -q "E_FILE_NOT_FOUND"; then
        rm -rf "$tmp_dir"
        return 0
    fi

    test_fail "E_FILE_NOT_FOUND error" "$output"
    rm -rf "$tmp_dir"
    return 1
}

# ============================================================================
# Test: state-mutate modifies state and persists
# ============================================================================
test_state_mutate() {
    local tmp_dir
    tmp_dir=$(setup_state_env)

    local output
    output=$(run_cmd "$tmp_dir" state-mutate '.state = "EXECUTING"')

    if ! assert_ok_true "$output"; then
        test_fail "ok:true on mutate" "not ok"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the change persists
    local verify
    verify=$(run_cmd "$tmp_dir" state-read-field ".state")
    local state_val
    state_val=$(echo "$verify" | jq -r '.result')

    if [[ "$state_val" != "EXECUTING" ]]; then
        test_fail "state = 'EXECUTING'" "state = '$state_val'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: state-mutate releases lock (no stale lock files)
# ============================================================================
test_state_mutate_no_stale_lock() {
    local tmp_dir
    tmp_dir=$(setup_state_env)

    run_cmd "$tmp_dir" state-mutate '.test_lock = true' >/dev/null

    # Check no lock files remain
    local lock_count
    lock_count=$(find "$tmp_dir/.aether" -name "*.lock" 2>/dev/null | wc -l | tr -d ' ')

    if [[ "$lock_count" -ne 0 ]]; then
        test_fail "0 lock files" "$lock_count lock files"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Test: state-write backward compatibility
# ============================================================================
test_state_write_compat() {
    local tmp_dir
    tmp_dir=$(setup_state_env)

    local new_state='{"version":"3.0","goal":"Updated via state-write","state":"READY","current_phase":2,"plan":{"phases":[]},"memory":{},"errors":{"records":[]},"events":[]}'
    local output
    output=$(run_cmd "$tmp_dir" state-write "$new_state")

    if ! assert_ok_true "$output"; then
        test_fail "ok:true on state-write" "not ok"
        rm -rf "$tmp_dir"
        return 1
    fi

    # Verify the written content
    local verify
    verify=$(run_cmd "$tmp_dir" state-read-field ".goal")
    local goal
    goal=$(echo "$verify" | jq -r '.result')

    if [[ "$goal" != "Updated via state-write" ]]; then
        test_fail "goal = 'Updated via state-write'" "goal = '$goal'"
        rm -rf "$tmp_dir"
        return 1
    fi

    rm -rf "$tmp_dir"
    return 0
}

# ============================================================================
# Run all tests
# ============================================================================
log_info "Running State API integration tests"

run_test test_state_read_valid "state-read returns valid JSON with ok:true"
run_test test_state_read_missing "state-read returns error for missing file"
run_test test_state_read_field "state-read-field extracts a specific field"
run_test test_state_read_field_missing "state-read-field returns error for missing file"
run_test test_state_mutate "state-mutate modifies state and persists"
run_test test_state_mutate_no_stale_lock "state-mutate releases lock (no stale files)"
run_test test_state_write_compat "state-write backward compatibility"

test_summary
