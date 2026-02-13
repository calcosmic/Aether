#!/usr/bin/env bash
# Aether Bash Test Helpers
# Reusable test utilities for bash integration tests

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test state
TEST_NAME=""
TEST_OUTPUT=""
TEST_EXIT_CODE=0

# ============================================================================
# Logging Functions
# ============================================================================

log() {
    echo -e "${NC}[$(date +'%H:%M:%S')] $1${NC}"
}

log_info() {
    log "${BLUE}INFO${NC}: $1"
}

log_warn() {
    log "${YELLOW}WARN${NC}: $1"
}

log_error() {
    log "${RED}ERROR${NC}: $1"
}

# ============================================================================
# Test Execution Functions
# ============================================================================

test_start() {
    TEST_NAME="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    log "${YELLOW}TEST $TESTS_RUN: $TEST_NAME${NC}"
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log "${GREEN}✓ PASS${NC}: $TEST_NAME"
}

test_fail() {
    local expected="${1:-}"
    local got="${2:-}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log "${RED}✗ FAIL${NC}: $TEST_NAME"
    if [[ -n "$expected" ]]; then
        log "${RED}  Expected: $expected${NC}"
    fi
    if [[ -n "$got" ]]; then
        log "${RED}  Got: $got${NC}"
    fi
}

# ============================================================================
# Assertion Functions
# ============================================================================

# assert_json_valid - Verify output is valid JSON using jq
# Usage: assert_json_valid "$output"
assert_json_valid() {
    local json="$1"
    if echo "$json" | jq empty 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# assert_json_field_equals - Check a specific JSON field value
# Usage: assert_json_field_equals "$json" ".field" "expected_value"
assert_json_field_equals() {
    local json="$1"
    local field="$2"
    local expected="$3"
    local actual

    actual=$(echo "$json" | jq -r "$field" 2>/dev/null || echo "null")
    if [[ "$actual" == "$expected" ]]; then
        return 0
    else
        return 1
    fi
}

# assert_ok_true - Verify {"ok":true} response
# Usage: assert_ok_true "$json"
assert_ok_true() {
    local json="$1"
    assert_json_field_equals "$json" ".ok" "true"
}

# assert_ok_false - Verify {"ok":false} response
# Usage: assert_ok_false "$json"
assert_ok_false() {
    local json="$1"
    assert_json_field_equals "$json" ".ok" "false"
}

# assert_exit_code - Verify command exit status
# Usage: assert_exit_code $exit_code 0
assert_exit_code() {
    local actual="$1"
    local expected="$2"
    if [[ "$actual" -eq "$expected" ]]; then
        return 0
    else
        return 1
    fi
}

# assert_json_has_field - Verify JSON has a specific field
# Usage: assert_json_has_field "$json" "field_name"
assert_json_has_field() {
    local json="$1"
    local field="$2"
    if echo "$json" | jq -e "has(\"$field\")" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# assert_json_array_length - Verify JSON array has expected length
# Usage: assert_json_array_length "$json" ".array_field" 5
assert_json_array_length() {
    local json="$1"
    local array_path="$2"
    local expected="$3"
    local actual

    actual=$(echo "$json" | jq "${array_path} | length" 2>/dev/null || echo "0")
    if [[ "$actual" -eq "$expected" ]]; then
        return 0
    else
        return 1
    fi
}

# assert_contains - Verify string contains substring
# Usage: assert_contains "$haystack" "needle"
assert_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" == *"$needle"* ]]; then
        return 0
    else
        return 1
    fi
}

# assert_file_exists - Verify file exists
assert_file_exists() {
    [[ -f "$1" ]]
}

# assert_dir_exists - Verify directory exists
assert_dir_exists() {
    [[ -d "$1" ]]
}

# ============================================================================
# Test Environment Functions
# ============================================================================

# setup_test_env - Create temporary test environment
# Sets TEST_TMP_DIR, TEST_DATA_DIR, and exports them
setup_test_env() {
    TEST_TMP_DIR=$(mktemp -d)
    TEST_DATA_DIR="$TEST_TMP_DIR/.aether/data"
    mkdir -p "$TEST_DATA_DIR"

    # Create minimal COLONY_STATE.json
    cat > "$TEST_DATA_DIR/COLONY_STATE.json" << 'EOF'
{
  "goal": "test",
  "state": "active",
  "current_phase": 1,
  "plan": {"id": "test-plan", "tasks": []},
  "memory": {},
  "errors": {"records": []},
  "events": [],
  "session_id": "test-session",
  "initialized_at": "2026-02-13T16:00:00Z"
}
EOF

    # Create minimal constraints.json
    cat > "$TEST_DATA_DIR/constraints.json" << 'EOF'
{
  "focus": ["testing"],
  "constraints": ["test constraint"]
}
EOF

    export TEST_TMP_DIR
    export TEST_DATA_DIR
    log_info "Test environment created at: $TEST_TMP_DIR"
}

# teardown_test_env - Clean up temporary files
teardown_test_env() {
    if [[ -n "${TEST_TMP_DIR:-}" && -d "$TEST_TMP_DIR" ]]; then
        rm -rf "$TEST_TMP_DIR"
        log_info "Test environment cleaned up"
    fi
    unset TEST_TMP_DIR
    unset TEST_DATA_DIR
}

# run_test - Execute a test function with setup/teardown
# Usage: run_test "test_function_name"
run_test() {
    local test_func="$1"
    local test_name="${2:-$test_func}"

    test_start "$test_name"

    # Run the test function
    if "$test_func"; then
        test_pass
    else
        test_fail "" ""
    fi
}

# run_test_with_env - Execute a test with full environment setup
# Usage: run_test_with_env "test_function_name"
run_test_with_env() {
    local test_func="$1"
    local test_name="${2:-$test_func}"
    local original_dir
    original_dir=$(pwd)

    setup_test_env
    cd "$TEST_TMP_DIR"

    # Run the test function
    test_start "$test_name"
    if "$test_func"; then
        test_pass
    else
        test_fail "" ""
    fi

    cd "$original_dir"
    teardown_test_env
}

# ============================================================================
# Test Summary
# ============================================================================

test_summary() {
    log ""
    log "${YELLOW}=== Test Summary ===${NC}"
    log "Tests run:    $TESTS_RUN"
    log -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
    if [[ "$TESTS_FAILED" -gt 0 ]]; then
        log -e "${RED}Tests failed: $TESTS_FAILED${NC}"
        return 1
    else
        log "Tests failed: $TESTS_FAILED"
        return 0
    fi
}

# ============================================================================
# Utility Functions
# ============================================================================

# run_aether_utils - Run aether-utils.sh subcommand and capture output
# Usage: output=$(run_aether_utils "subcommand" "arg1" "arg2")
# Sets: TEST_OUTPUT and TEST_EXIT_CODE
run_aether_utils() {
    local aether_utils_path="${AETHER_UTILS_PATH:-.aether/aether-utils.sh}"
    local subcommand="$1"
    shift

    # Capture both stdout and stderr
    TEST_OUTPUT=$(bash "$aether_utils_path" "$subcommand" "$@" 2>&1) || TEST_EXIT_CODE=$?
    TEST_EXIT_CODE=${TEST_EXIT_CODE:-0}

    echo "$TEST_OUTPUT"
}

# require_jq - Check that jq is installed
require_jq() {
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        exit 1
    fi
}

# Export all functions for use in other scripts
export -f log log_info log_warn log_error
export -f test_start test_pass test_fail
export -f assert_json_valid assert_json_field_equals assert_ok_true assert_ok_false
export -f assert_exit_code assert_json_has_field assert_json_array_length
export -f assert_contains assert_file_exists assert_dir_exists
export -f setup_test_env teardown_test_env run_test run_test_with_env
export -f test_summary run_aether_utils require_jq
