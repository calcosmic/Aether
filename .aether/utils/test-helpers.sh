#!/bin/bash
# Aether Test Helper Utility
# Provides test isolation, cleanup, and assertion utilities
#
# Usage:
#   source .aether/utils/test-helpers.sh
#   test_setup "test_name"
#   # ... run tests ...
#   test_teardown

# Global variables for test isolation
TEST_BACKUP_DIR="/tmp/aether_test_backups"
TEST_CURRENT_BACKUP=""

# Colors for test output
export TEST_COLOR_GREEN='\033[0;32m'
export TEST_COLOR_RED='\033[0;31m'
export TEST_COLOR_YELLOW='\033[1;33m'
export TEST_COLOR_BLUE='\033[0;34m'
export TEST_COLOR_RESET='\033[0m'

# Setup test environment
# Arguments: test_name
# Creates backup of existing state and prepares fresh test environment
test_setup() {
    local test_name="$1"
    local timestamp=$(date +%s)
    TEST_CURRENT_BACKUP="${TEST_BACKUP_DIR}/${test_name}_${timestamp}"

    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}Setting up test: ${test_name}${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"

    # Create backup directory
    mkdir -p "$TEST_BACKUP_DIR"

    # Backup events.json if it exists
    if [ -f "$EVENTS_FILE" ]; then
        cp "$EVENTS_FILE" "${TEST_CURRENT_BACKUP}_events.json"
        echo -e "${TEST_COLOR_YELLOW}INFO:${TEST_COLOR_RESET} Backed up ${EVENTS_FILE}"
    fi

    # Backup COLONY_STATE.json if it exists
    local colony_state="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/data/COLONY_STATE.json"
    if [ -f "$colony_state" ]; then
        cp "$colony_state" "${TEST_CURRENT_BACKUP}_colony_state.json"
        echo -e "${TEST_COLOR_YELLOW}INFO:${TEST_COLOR_RESET} Backed up ${colony_state}"
    fi

    # Create fresh event bus
    rm -f "$EVENTS_FILE" 2>/dev/null
    source "$(dirname "${BASH_SOURCE[0]}")/event-bus.sh"
    initialize_event_bus > /dev/null 2>&1

    echo -e "${TEST_COLOR_GREEN}✓${TEST_COLOR_RESET} Test environment ready"
    echo
}

# Teardown test environment
# Restores backed up state and cleans up temp files
test_teardown() {
    local test_name="${1:-test}"
    local exit_code="${2:-0}"

    echo
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}Tearing down test: ${test_name}${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"

    local restored=false

    # Restore events.json if backup exists
    if [ -f "${TEST_CURRENT_BACKUP}_events.json" ]; then
        mv "${TEST_CURRENT_BACKUP}_events.json" "$EVENTS_FILE"
        echo -e "${TEST_COLOR_YELLOW}INFO:${TEST_COLOR_RESET} Restored ${EVENTS_FILE}"
        restored=true
    fi

    # Restore COLONY_STATE.json if backup exists
    local colony_state="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")/.aether/data/COLONY_STATE.json"
    if [ -f "${TEST_CURRENT_BACKUP}_colony_state.json" ]; then
        mv "${TEST_CURRENT_BACKUP}_colony_state.json" "$colony_state"
        echo -e "${TEST_COLOR_YELLOW}INFO:${TEST_COLOR_RESET} Restored ${colony_state}"
        restored=true
    fi

    # Clean up backup file
    if [ -n "$TEST_CURRENT_BACKUP" ]; then
        rm -f "${TEST_CURRENT_BACKUP}_"* 2>/dev/null
    fi

    if [ "$restored" = true ]; then
        echo -e "${TEST_COLOR_GREEN}✓${TEST_COLOR_RESET} Restored original state"
    else
        echo -e "${TEST_COLOR_YELLOW}INFO:${TEST_COLOR_RESET} No backup to restore"
    fi
    echo
}

# Assert that a command succeeds
# Arguments: description, command...
# Returns: 0 if assertion passes, exits with 1 if fails
assert_success() {
    local description="$1"
    shift
    local command="$@"

    if eval "$command" > /dev/null 2>&1; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        return 1
    fi
}

# Assert that a command fails
# Arguments: description, command...
# Returns: 0 if assertion passes, exits with 1 if fails
assert_failure() {
    local description="$1"
    shift
    local command="$@"

    if eval "$command" > /dev/null 2>&1; then
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description (should have failed)"
        return 1
    else
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    fi
}

# Assert that two values are equal
# Arguments: description, expected, actual
# Returns: 0 if assertion passes, exits with 1 if fails
assert_equals() {
    local description="$1"
    local expected="$2"
    local actual="$3"

    if [ "$expected" = "$actual" ]; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        echo "  Expected: $expected"
        echo "  Got: $actual"
        return 1
    fi
}

# Assert that a numeric comparison is true
# Arguments: description, operator, expected, actual
# Operators: -eq, -ne, -gt, -ge, -lt, -le
# Returns: 0 if assertion passes, exits with 1 if fails
assert_numeric() {
    local description="$1"
    local operator="$2"
    local expected="$3"
    local actual="$4"

    if [ "$actual" "$operator" "$expected" ]; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        echo "  Expected: $actual $operator $expected"
        return 1
    fi
}

# Assert that a string contains a substring
# Arguments: description, haystack, needle
# Returns: 0 if assertion passes, exits with 1 if fails
assert_contains() {
    local description="$1"
    local haystack="$2"
    local needle="$3"

    if echo "$haystack" | grep -q "$needle"; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        echo "  String does not contain: $needle"
        return 1
    fi
}

# Assert that a file exists
# Arguments: description, filepath
# Returns: 0 if assertion passes, exits with 1 if fails
assert_file_exists() {
    local description="$1"
    local filepath="$2"

    if [ -f "$filepath" ]; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        echo "  File not found: $filepath"
        return 1
    fi
}

# Assert that JSON field has expected value
# Arguments: description, json_input, field_path, expected_value
# Returns: 0 if assertion passes, exits with 1 if fails
assert_json_field() {
    local description="$1"
    local json_input="$2"
    local field_path="$3"
    local expected_value="$4"

    local actual_value=$(echo "$json_input" | jq -r "$field_path")

    if [ "$actual_value" = "$expected_value" ]; then
        echo -e "${TEST_COLOR_GREEN}PASS:${TEST_COLOR_RESET} $description"
        return 0
    else
        echo -e "${TEST_COLOR_RED}FAIL:${TEST_COLOR_RESET} $description"
        echo "  Field: $field_path"
        echo "  Expected: $expected_value"
        echo "  Got: $actual_value"
        return 1
    fi
}

# Print test section header
# Arguments: section_name
test_section() {
    local section_name="$1"
    echo
    echo -e "${TEST_COLOR_BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}  ${section_name}${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${TEST_COLOR_RESET}"
    echo
}

# Print test summary
# Arguments: total_tests, passed_tests
test_summary() {
    local total="$1"
    local passed="$2"
    local failed=$((total - passed))
    local pass_rate=$(echo "scale=2; $passed * 100 / $total" | bc 2>/dev/null || echo "N/A")

    echo
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}Test Summary${TEST_COLOR_RESET}"
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"
    echo "Tests Run:    $total"
    echo -e "Tests Passed: ${TEST_COLOR_GREEN}${passed}${TEST_COLOR_RESET}"
    if [ "$failed" -gt 0 ]; then
        echo -e "Tests Failed: ${TEST_COLOR_RED}${failed}${TEST_COLOR_RESET}"
    else
        echo "Tests Failed: $failed"
    fi
    echo "Pass Rate:    ${pass_rate}%"
    echo -e "${TEST_COLOR_BLUE}========================================${TEST_COLOR_RESET}"

    if [ "$failed" -eq 0 ]; then
        echo -e "${TEST_COLOR_GREEN}All tests passed!${TEST_COLOR_RESET}"
    else
        echo -e "${TEST_COLOR_RED}Some tests failed!${TEST_COLOR_RESET}"
    fi
}

# Cleanup temp test files
cleanup_test_files() {
    rm -f /tmp/test_events*.json 2>/dev/null
    rm -f /tmp/test_events*.txt 2>/dev/null
    rm -f /tmp/events_*.tmp 2>/dev/null
}

# Export functions
export -f test_setup test_teardown
export -f assert_success assert_failure assert_equals assert_numeric assert_contains assert_file_exists assert_json_field
export -f test_section test_summary cleanup_test_files
