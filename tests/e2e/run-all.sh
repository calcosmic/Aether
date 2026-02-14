#!/usr/bin/env bash
# Aether End-to-End Test Runner
# Runs all E2E tests and generates a summary report

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
declare -A TEST_RESULTS
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0
FAILED_SUITES=()

# Utility functions
log_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo ""
}

log_test_result() {
    local suite="$1"
    local result="$2"
    local passed="$3"
    local failed="$4"
    local skipped="${5:-0}"

    TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))
    TOTAL_PASSED=$((TOTAL_PASSED + passed))
    TOTAL_FAILED=$((TOTAL_FAILED + failed))
    TOTAL_SKIPPED=$((TOTAL_SKIPPED + skipped))

    if [[ "$result" == "PASS" ]]; then
        echo -e "${GREEN}✓ $suite: ALL TESTS PASSED${NC}"
    elif [[ "$result" == "FAIL" ]]; then
        echo -e "${RED}✗ $suite: SOME TESTS FAILED${NC}"
        FAILED_SUITES+=("$suite")
    elif [[ "$result" == "SKIP" ]]; then
        echo -e "${YELLOW}⊘ $suite: SKIPPED${NC}"
    else
        echo -e "${YELLOW}? $suite: UNKNOWN RESULT${NC}"
    fi

    echo "  Tests: $passed passed, $failed failed, $skipped skipped"
    echo ""
}

run_test() {
    local test_script="$1"
    local test_name=$(basename "$test_script" .sh)

    # Make executable
    chmod +x "$test_script"

    # Run the test
    if bash "$test_script" 2>&1; then
        return 0
    else
        return $?
    fi
}

# Main execution
main() {
    cd "$SCRIPT_DIR"

    log_header "Aether End-to-End Test Suite"

    echo "Running tests from: $SCRIPT_DIR"
    echo "Project root: $PROJECT_ROOT"
    echo ""

    # Check prerequisites
    if ! command -v node &>/dev/null; then
        echo -e "${RED}ERROR: node is required to run tests${NC}"
        exit 1
    fi

    if ! command -v jq &>/dev/null; then
        echo -e "${RED}ERROR: jq is required to run tests${NC}"
        exit 1
    fi

    # Run each test suite
    for test_script in test-*.sh; do
        if [[ -f "$test_script" ]]; then
            local test_name=$(basename "$test_script" .sh)
            log_header "$test_name"

            # Capture test output
            local output
            local exit_code

            output=$(bash "$test_script" 2>&1)
            exit_code=$?

            # Parse results from output
            local passed=$(echo "$output" | grep "Tests passed:" | awk '{print $3}' || echo "0")
            local failed=$(echo "$output" | grep "Tests failed:" | awk '{print $3}' || echo "0")

            if [[ "$exit_code" -eq 0 ]]; then
                log_test_result "$test_name" "PASS" "$passed" "$failed" "0"
            else
                log_test_result "$test_name" "FAIL" "$passed" "$failed" "0"
            fi
        fi
    done

    # Final summary
    log_header "FINAL SUMMARY"

    echo -e "Total tests run: ${BLUE}$TOTAL_TESTS${NC}"
    echo -e "Total passed:   ${GREEN}$TOTAL_PASSED${NC}"

    if [[ "$TOTAL_FAILED" -gt 0 ]]; then
        echo -e "Total failed:   ${RED}$TOTAL_FAILED${NC}"
    else
        echo "Total failed:   $TOTAL_FAILED"
    fi

    if [[ "$TOTAL_SKIPPED" -gt 0 ]]; then
        echo "Total skipped:  $TOTAL_SKIPPED"
    fi

    echo ""

    # Calculate success rate
    if [[ "$TOTAL_TESTS" -gt 0 ]]; then
        local success_rate=$(( TOTAL_PASSED * 100 / TOTAL_TESTS ))
        echo -e "Success rate:    ${success_rate}%${NC}"
    fi

    echo ""

    # List failed suites
    if [[ "${#FAILED_SUITES[@]}" -gt 0 ]]; then
        echo -e "${RED}Failed test suites:${NC}"
        for suite in "${FAILED_SUITES[@]}"; do
            echo -e "  ${RED}✗${NC} $suite"
        done
        echo ""
        exit 1
    else
        echo -e "${GREEN}All test suites passed!${NC}"
        echo ""
        exit 0
    fi
}

main "$@"
