#!/bin/bash
# Master test runner for Aether test suite
#
# Provides unified entry point for running all tests with:
# - Test discovery (finds all .test.sh files)
# - Colored output (green=pass, red=fail)
# - Summary reporting (total, passed, failed, duration)
# - Selective execution (all, integration, unit)
#
# Usage:
#   bash tests/test-orchestrator.sh --all
#   bash tests/test-orchestrator.sh --integration --verbose
#   bash tests/test-orchestrator.sh --clean --all

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASE_DIR="$(dirname "$SCRIPT_DIR")"

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Timing
START_TIME=$(date +%s)

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Flags
VERBOSE=false
CLEAN=false
RUN_ALL=false
RUN_INTEGRATION=false
RUN_UNIT=false

# Print usage
print_usage() {
    cat <<EOF
Usage: bash tests/test-orchestrator.sh [OPTIONS]

Options:
  --all              Run all tests (integration and unit)
  --integration      Run only integration tests
  --unit             Run only unit tests
  --verbose          Enable verbose TAP output with diagnostics
  --clean            Force cleanup before running tests
  --help             Show this help message

Examples:
  bash tests/test-orchestrator.sh --all
  bash tests/test-orchestrator.sh --integration --verbose
  bash tests/test-orchestrator.sh --clean --all

EOF
}

# Parse command line arguments
parse_args() {
    if [ $# -eq 0 ]; then
        print_usage
        exit 1
    fi

    while [ $# -gt 0 ]; do
        case "$1" in
            --all)
                RUN_ALL=true
                shift
                ;;
            --integration)
                RUN_INTEGRATION=true
                shift
                ;;
            --unit)
                RUN_UNIT=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --clean)
                CLEAN=true
                shift
                ;;
            --help|-h)
                print_usage
                exit 0
                ;;
            *)
                echo -e "${RED}Error: Unknown option: $1${NC}" >&2
                print_usage
                exit 1
                ;;
        esac
    done
}

# Force cleanup if requested
force_cleanup() {
    if [ "$CLEAN" = true ]; then
        echo -e "${BLUE} forcing cleanup before tests...${NC}"

        if [ -f "$SCRIPT_DIR/helpers/cleanup.sh" ]; then
            source "$SCRIPT_DIR/helpers/cleanup.sh"
            force_cleanup_test_colony
        else
            echo -e "${YELLOW}Warning: cleanup.sh not found${NC}" >&2
        fi
    fi
}

# Discover test files
discover_tests() {
    local test_files=()

    if [ "$RUN_ALL" = true ]; then
        # Find all .test.sh files
        while IFS= read -r -d '' file; do
            test_files+=("$file")
        done < <(find "$SCRIPT_DIR" -name "*.test.sh" -type f -print0 | sort -z)

    elif [ "$RUN_INTEGRATION" = true ]; then
        # Find integration tests only
        while IFS= read -r -d '' file; do
            test_files+=("$file")
        done < <(find "$SCRIPT_DIR/integration" -name "*.test.sh" -type f -print0 | sort -z)

    elif [ "$RUN_UNIT" = true ]; then
        # Find unit tests only
        while IFS= read -r -d '' file; do
            test_files+=("$file")
        done < <(find "$SCRIPT_DIR/unit" -name "*.test.sh" -type f -print0 | sort -z)
    fi

    echo "${test_files[@]}"
}

# Run a single test file
run_test() {
    local test_file="$1"
    local test_name=$(basename "$test_file" .test.sh)
    local start_time=$(date +%s)

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    echo ""
    echo -e "${BLUE}Running: $test_name${NC}"
    echo "File: $test_file"

    # Run test and capture output
    local output=""
    local exit_code=0

    if [ "$VERBOSE" = true ]; then
        # Verbose mode: show all output
        if bash "$test_file"; then
            exit_code=0
        else
            exit_code=$?
        fi
    else
        # Non-verbose: capture output but show on failure
        output=$(bash "$test_file" 2>&1)
        exit_code=$?
    fi

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [ $exit_code -eq 0 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}✓ PASSED${NC} (${duration}s)"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}✗ FAILED${NC} (${duration}s)"

        # Show output on failure
        if [ -n "$output" ]; then
            echo ""
            echo -e "${YELLOW}Test output:${NC}"
            echo "$output" | sed 's/^/  /'
        fi
    fi
}

# Print test summary
print_summary() {
    local end_time=$(date +%s)
    local total_duration=$((end_time - START_TIME))

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Test Suite Summary${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo ""
    echo "Total tests: $TESTS_TOTAL"
    echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
    echo ""
    echo "Duration: ${total_duration}s"
    echo ""

    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${YELLOW}Diagnostic hints:${NC}"
        echo "  1. Run with --verbose for detailed output"
        echo "  2. Check .aether/data for state dumps"
        echo "  3. Review test logs in tests/logs/"
        echo ""
    fi

    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
}

# Main execution
main() {
    parse_args "$@"
    force_cleanup

    # Discover tests
    TEST_FILES=($(discover_tests))

    if [ ${#TEST_FILES[@]} -eq 0 ]; then
        echo -e "${YELLOW}No tests found matching criteria${NC}" >&2
        exit 0
    fi

    echo -e "${BLUE}Aether Test Suite${NC}"
    echo "Running ${#TEST_FILES[@]} test(s)..."
    echo ""

    # Run each test
    for test_file in "${TEST_FILES[@]}"; do
        run_test "$test_file"
    done

    # Print summary
    print_summary

    # Exit with appropriate code
    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Run main
main "$@"
