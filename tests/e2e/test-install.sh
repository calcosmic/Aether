#!/usr/bin/env bash
# Aether End-to-End Test: aether install
# Tests that the install command properly sets up the distribution hub

set -euo pipefail

# Test configuration
TEST_NAME="aether install"
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/../.." && pwd)"
TMP_DIR=$(mktemp -d)
export HOME="$TMP_DIR"  # Isolate test environment
CLEANUP_DONE=false

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Utility functions
log() {
    echo -e "${NC}[$(date +'%H:%M:%S')] $1"
}

log_test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    log "${YELLOW}TEST $TESTS_RUN: $1${NC}"
}

log_test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log "${GREEN}✓ PASS${NC}: $1"
}

log_test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log "${RED}✗ FAIL${NC}: $1"
    log "${RED}  Expected: $2${NC}"
    log "${RED}  Got: $3${NC}"
}

assert_file_exists() {
    if [[ -f "$1" ]]; then
        return 0
    else
        return 1
    fi
}

assert_dir_exists() {
    if [[ -d "$1" ]]; then
        return 0
    else
        return 1
    fi
}

assert_file_count() {
    local dir="$1"
    local expected="$2"
    local actual=$(find "$dir" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$actual" -eq "$expected" ]]; then
        return 0
    else
        return 1
    fi
}

# Cleanup
cleanup() {
    if [[ "$CLEANUP_DONE" == "true" ]]; then
        return
    fi
    CLEANUP_DONE=true

    log "Cleaning up test environment..."
    if [[ -d "$TMP_DIR" ]]; then
        rm -rf "$TMP_DIR"
    fi
}

trap cleanup EXIT INT TERM

# ============================================================================
# Test Setup
# ============================================================================

log "${YELLOW}=== Aether Install E2E Test Suite ===${NC}"
log "Project root: $PROJECT_ROOT"
log "Test directory: $TEST_DIR"
log "Temporary HOME: $TMP_DIR"
log ""

# ============================================================================
# Test 1: Install command creates distribution hub
# ============================================================================

log_test_start "Install creates ~/.aether/ directory"

cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

if assert_dir_exists "$HOME/.aether"; then
    log_test_pass "Install creates ~/.aether/" "~/.aether/ exists"
else
    log_test_fail "Install creates ~/.aether/" "~/.aether/ exists" "directory not found"
fi

# ============================================================================
# Test 2: Install creates system directory
# ============================================================================

log_test_start "Install creates ~/.aether/system/ directory"

if assert_dir_exists "$HOME/.aether/system"; then
    log_test_pass "Install creates ~/.aether/system/" "~/.aether/system/ exists"
else
    log_test_fail "Install creates ~/.aether/system/" "~/.aether/system/ exists" "directory not found"
fi

# ============================================================================
# Test 3: Install creates claude commands directory
# ============================================================================

log_test_start "Install creates ~/.aether/commands/claude/ directory"

if assert_dir_exists "$HOME/.aether/commands/claude"; then
    log_test_pass "Install creates ~/.aether/commands/claude/" "~/.aether/commands/claude/ exists"
else
    log_test_fail "Install creates ~/.aether/commands/claude/" "~/.aether/commands/claude/ exists" "directory not found"
fi

# ============================================================================
# Test 4: Install creates opencode commands directory
# ============================================================================

log_test_start "Install creates ~/.aether/commands/opencode/ directory"

if assert_dir_exists "$HOME/.aether/commands/opencode"; then
    log_test_pass "Install creates ~/.aether/commands/opencode/" "~/.aether/commands/opencode/ exists"
else
    log_test_fail "Install creates ~/.aether/commands/opencode/" "~/.aether/commands/opencode/ exists" "directory not found"
fi

# ============================================================================
# Test 5: Install creates agents directory
# ============================================================================

log_test_start "Install creates ~/.aether/agents/ directory"

if assert_dir_exists "$HOME/.aether/agents"; then
    log_test_pass "Install creates ~/.aether/agents/" "~/.aether/agents/ exists"
else
    log_test_fail "Install creates ~/.aether/agents/" "~/.aether/agents/ exists" "directory not found"
fi

# ============================================================================
# Test 6: Install copies claude commands
# ============================================================================

log_test_start "Install copies claude commands to global hub"

# Count command files in source
SOURCE_CMD_COUNT=$(find "$PROJECT_ROOT/.claude/commands/ant" -type f 2>/dev/null | wc -l | tr -d ' ')
DEST_CMD_COUNT=$(find "$HOME/.aether/commands/claude" -type f 2>/dev/null | wc -l | tr -d ' ')

if [[ "$DEST_CMD_COUNT" -ge "$SOURCE_CMD_COUNT" ]]; then
    log_test_pass "Install copies claude commands" "$DEST_CMD_COUNT commands copied (source: $SOURCE_CMD_COUNT)"
else
    log_test_fail "Install copies claude commands" ">= $SOURCE_CMD_COUNT files" "$DEST_CMD_COUNT files found"
fi

# ============================================================================
# Test 7: Install creates version.json
# ============================================================================

log_test_start "Install creates ~/.aether/version.json"

if assert_file_exists "$HOME/.aether/version.json"; then
    VERSION_CONTENT=$(cat "$HOME/.aether/version.json")
    if echo "$VERSION_CONTENT" | grep -q '"version"'; then
        log_test_pass "Install creates ~/.aether/version.json" "version.json created with version field"
    else
        log_test_fail "Install creates ~/.aether/version.json" "version field present" "invalid JSON structure"
    fi
else
    log_test_fail "Install creates ~/.aether/version.json" "file exists" "file not found"
fi

# ============================================================================
# Test 8: Install creates registry.json
# ============================================================================

log_test_start "Install creates ~/.aether/registry.json"

if assert_file_exists "$HOME/.aether/registry.json"; then
    REGISTRY_CONTENT=$(cat "$HOME/.aether/registry.json")
    if echo "$REGISTRY_CONTENT" | grep -q '"repos"'; then
        log_test_pass "Install creates ~/.aether/registry.json" "registry.json created with repos field"
    else
        log_test_fail "Install creates ~/.aether/registry.json" "repos field present" "invalid JSON structure"
    fi
else
    log_test_fail "Install creates ~/.aether/registry.json" "file exists" "file not found"
fi

# ============================================================================
# Test 9: Install creates manifest.json
# ============================================================================

log_test_start "Install creates ~/.aether/manifest.json"

if assert_file_exists "$HOME/.aether/manifest.json"; then
    MANIFEST_CONTENT=$(cat "$HOME/.aether/manifest.json")
    if echo "$MANIFEST_CONTENT" | grep -q '"files"' && echo "$MANIFEST_CONTENT" | grep -q '"generated_at"'; then
        log_test_pass "Install creates ~/.aether/manifest.json" "manifest.json created with required fields"
    else
        log_test_fail "Install creates ~/.aether/manifest.json" "files and generated_at fields present" "invalid JSON structure"
    fi
else
    log_test_fail "Install creates ~/.aether/manifest.json" "file exists" "file not found"
fi

# ============================================================================
# Test 10: Install copies system files
# ============================================================================

log_test_start "Install copies aether-utils.sh to system directory"

if assert_file_exists "$HOME/.aether/system/aether-utils.sh"; then
    # Check if file is executable
    if [[ -x "$HOME/.aether/system/aether-utils.sh" ]]; then
        log_test_pass "Install copies aether-utils.sh with exec bit" "aether-utils.sh is executable"
    else
        log_test_fail "Install copies aether-utils.sh with exec bit" "executable" "not executable"
    fi
else
    log_test_fail "Install copies aether-utils.sh" "file exists" "file not found"
fi

# ============================================================================
# Test 11: Idempotency - running install multiple times is safe
# ============================================================================

log_test_start "Install is idempotent (safe to run multiple times)"

# Get file count before second install
BEFORE_COUNT=$(find "$HOME/.aether" -type f 2>/dev/null | wc -l | tr -d ' ')

# Run install again
cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

# Get file count after second install
AFTER_COUNT=$(find "$HOME/.aether" -type f 2>/dev/null | wc -l | tr -d ' ')

if [[ "$AFTER_COUNT" -ge "$BEFORE_COUNT" ]]; then
    log_test_pass "Install is idempotent" "Files: $BEFORE_COUNT -> $AFTER_COUNT (no data loss)"
else
    log_test_fail "Install is idempotent" "file count stable or increased" "lost files: $BEFORE_COUNT -> $AFTER_COUNT"
fi

# ============================================================================
# Test 12: Shell scripts have executable bit
# ============================================================================

log_test_start "All shell scripts in hub are executable"

# Count non-executable shell scripts
# find with ! -executable returns exit code 1 when no results, so we use || true
FIND_OUTPUT=$(find "$HOME/.aether/system" -name "*.sh" -type f ! -executable 2>/dev/null || true)

if [[ -z "$FIND_OUTPUT" ]]; then
    # No non-executable files found
    NON_EXEC=0
else
    # Count non-executable files
    NON_EXEC=$(echo "$FIND_OUTPUT" | wc -l | tr -d ' ')
fi

if [[ "$NON_EXEC" -eq 0 ]]; then
    log_test_pass "All shell scripts are executable" "All .sh files have exec bit"
else
    log_test_fail "All shell scripts are executable" "0 non-executable" "$NON_EXEC non-executable files"
fi

# ============================================================================
# Test Summary
# ============================================================================

log ""
log "${YELLOW}=== Test Summary ===${NC}"
log "Tests run:    $TESTS_RUN"
log -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
if [[ "$TESTS_FAILED" -gt 0 ]]; then
    log -e "${RED}Tests failed:  $TESTS_FAILED${NC}"
    exit 1
else
    log "Tests failed:  $TESTS_FAILED"
    exit 0
fi
