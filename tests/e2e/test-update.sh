#!/usr/bin/env bash
# Aether End-to-End Test: aether update
# Tests that the update command properly updates a single repo

set -euo pipefail

# Test configuration
TEST_NAME="aether update"
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

get_file_hash() {
    if [[ -f "$1" ]]; then
        shasum -a 256 "$1" 2>/dev/null | awk '{print $1}' || echo "nohash"
    else
        echo "nofile"
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

log "${YELLOW}=== Aether Update E2E Test Suite ===${NC}"
log "Project root: $PROJECT_ROOT"
log "Test directory: $TEST_DIR"
log "Temporary HOME: $TMP_DIR"
log ""

# Create a test repo directory
TEST_REPO="$TMP_DIR/test-repo"
mkdir -p "$TEST_REPO"

# Initialize test repo with aether structure
mkdir -p "$TEST_REPO/.claude/commands/ant"
mkdir -p "$TEST_REPO/.opencode/commands/ant"
mkdir -p "$TEST_REPO/.opencode/agents"
mkdir -p "$TEST_REPO/.aether"

# Create initial version file
cat > "$TEST_REPO/.aether/version.json" <<EOF
{
  "version": "0.9.0",
  "updated_at": "2024-01-01T00:00:00Z"
}
EOF

# Create a test command file
cat > "$TEST_REPO/.claude/commands/ant/test-old.md" <<EOF
---
name: ant:test-old
description: "Test command - old version"
---
Old test command content.
EOF

# First run install to set up the hub
log "${YELLOW}Setup: Running install to create hub...${NC}"
cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

# Verify hub is created
if ! assert_dir_exists "$HOME/.aether"; then
    log "ERROR: Hub setup failed"
    exit 1
fi

log "Hub setup complete"
log ""

# ============================================================================
# Test 1: Update requires hub to exist
# ============================================================================

log_test_start "Update checks for hub existence"

# Temporarily rename hub to test error handling
mv "$HOME/.aether" "$HOME/.aether.bak" 2>/dev/null || true

cd "$TEST_REPO"
OUTPUT=$(node "$PROJECT_ROOT/bin/cli.js" update 2>&1 || true)

# Restore hub
mv "$HOME/.aether.bak" "$HOME/.aether" 2>/dev/null || true

if echo "$OUTPUT" | grep -q "No distribution hub found"; then
    log_test_pass "Update checks for hub" "Error message shown when hub missing"
else
    log_test_fail "Update checks for hub" "Error message about missing hub" "Got: $OUTPUT"
fi

# ============================================================================
# Test 2: Update requires .aether directory in repo
# ============================================================================

log_test_start "Update checks for .aether/ in target repo"

# Create a directory without .aether
NO_AETHER_DIR="$TMP_DIR/no-aether"
mkdir -p "$NO_AETHER_DIR"

cd "$NO_AETHER_DIR"
OUTPUT=$(node "$PROJECT_ROOT/bin/cli.js" update 2>&1 || true)

if echo "$OUTPUT" | grep -q "No .aether directory found"; then
    log_test_pass "Update checks for .aether/" "Error message shown when .aether/ missing"
else
    log_test_fail "Update checks for .aether/" "Error message about missing .aether/" "Got: $OUTPUT"
fi

# ============================================================================
# Test 3: Update copies system files
# ============================================================================

log_test_start "Update copies system files from hub"

cd "$TEST_REPO"
node "$PROJECT_ROOT/bin/cli.js" update --quiet 2>/dev/null || true

if assert_file_exists "$TEST_REPO/.aether/aether-utils.sh"; then
    log_test_pass "Update copies system files" "aether-utils.sh copied to repo"
else
    log_test_fail "Update copies system files" "aether-utils.sh exists in .aether/" "file not found"
fi

# ============================================================================
# Test 4: Update updates version.json
# ============================================================================

log_test_start "Update updates .aether/version.json"

VERSION_FILE="$TEST_REPO/.aether/version.json"
if assert_file_exists "$VERSION_FILE"; then
    NEW_VERSION=$(jq -r '.version' "$VERSION_FILE" 2>/dev/null || echo "unknown")
    HUB_VERSION=$(jq -r '.version' "$HOME/.aether/version.json" 2>/dev/null || echo "unknown")

    if [[ "$NEW_VERSION" == "$HUB_VERSION" ]]; then
        log_test_pass "Update updates version.json" "Version updated to $NEW_VERSION"
    else
        log_test_fail "Update updates version.json" "Version matches hub ($HUB_VERSION)" "Got: $NEW_VERSION"
    fi
else
    log_test_fail "Update updates version.json" "version.json exists" "file not found"
fi

# ============================================================================
# Test 5: Update syncs claude commands
# ============================================================================

log_test_start "Update syncs claude commands"

# Add a new command to hub that should be copied to repo
cat > "$HOME/.aether/commands/claude/test-new.md" <<EOF
---
name: ant:test-new
description: "Test command - new"
---
New test command from hub.
EOF

cd "$TEST_REPO"
node "$PROJECT_ROOT/bin/cli.js" update --quiet 2>/dev/null || true

if assert_file_exists "$TEST_REPO/.claude/commands/ant/test-new.md"; then
    log_test_pass "Update syncs claude commands" "New command test-new.md copied"
else
    log_test_fail "Update syncs claude commands" "test-new.md exists" "file not copied"
fi

# ============================================================================
# Test 6: Update removes stale files
# ============================================================================

log_test_start "Update removes stale command files"

# Verify old file was removed
if ! assert_file_exists "$TEST_REPO/.claude/commands/ant/test-old.md"; then
    log_test_pass "Update removes stale commands" "test-old.md removed (not in hub)"
else
    log_test_fail "Update removes stale commands" "test-old.md removed" "file still exists"
fi

# ============================================================================
# Test 7: Update preserves colony data
# ============================================================================

log_test_start "Update preserves colony data directory"

# Create colony data with a test file
mkdir -p "$TEST_REPO/.aether/data"
echo "test data" > "$TEST_REPO/.aether/data/test-preserve.txt"

cd "$TEST_REPO"
node "$PROJECT_ROOT/bin/cli.js" update --quiet 2>/dev/null || true

if assert_file_exists "$TEST_REPO/.aether/data/test-preserve.txt"; then
    CONTENT=$(cat "$TEST_REPO/.aether/data/test-preserve.txt")
    if [[ "$CONTENT" == "test data" ]]; then
        log_test_pass "Update preserves colony data" "test-preserve.txt unchanged"
    else
        log_test_fail "Update preserves colony data" "content preserved" "Content changed to: $CONTENT"
    fi
else
    log_test_fail "Update preserves colony data" "test-preserve.txt exists" "file was removed"
fi

# ============================================================================
# Test 8: Hash comparison prevents unnecessary writes
# ============================================================================

log_test_start "Hash comparison prevents unnecessary file writes"

# Get modification time before second update
if assert_file_exists "$TEST_REPO/.aether/aether-utils.sh"; then
    # Get mtime using platform-appropriate stat command
    MTIME_BEFORE=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%Y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null)

    # Run update again (no changes expected)
    cd "$TEST_REPO"
    node "$PROJECT_ROOT/bin/cli.js" update --quiet 2>/dev/null || true

    MTIME_AFTER=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%Y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null)

    if [[ "$MTIME_BEFORE" == "$MTIME_AFTER" ]]; then
        log_test_pass "Hash comparison prevents unnecessary writes" "File mtime unchanged"
    else
        log_test_fail "Hash comparison prevents unnecessary writes" "File mtime unchanged" "File was rewritten (mtime changed)"
    fi
else
    log_test_fail "Hash comparison setup" "aether-utils.sh exists" "file not found"
fi

# ============================================================================
# Test 9: Update detects up-to-date repos
# ============================================================================

log_test_start "Update detects up-to-date repos"

# Set version to match hub
HUB_VERSION=$(jq -r '.version' "$HOME/.aether/version.json" 2>/dev/null)
jq ".version = \"$HUB_VERSION\"" "$TEST_REPO/.aether/version.json" > "$TEST_REPO/.aether/version.json.tmp"
mv "$TEST_REPO/.aether/version.json.tmp" "$TEST_REPO/.aether/version.json"

cd "$TEST_REPO"
OUTPUT=$(node "$PROJECT_ROOT/bin/cli.js" update 2>&1 || true)

if echo "$OUTPUT" | grep -q "Already up-to-date"; then
    log_test_pass "Update detects up-to-date repos" "Up-to-date message shown"
else
    log_test_fail "Update detects up-to-date repos" "Up-to-date message" "Got: $OUTPUT"
fi

# ============================================================================
# Test 10: Update --dry-run doesn't modify files
# ============================================================================

log_test_start "Update --dry-run doesn't modify files"

# Create a file that would be updated
echo "old content" > "$TEST_REPO/.aether/aether-utils.sh"
MTIME_BEFORE=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%Y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null)

cd "$TEST_REPO"
OUTPUT=$(node "$PROJECT_ROOT/bin/cli.js" update --dry-run 2>&1 || true)

MTIME_AFTER=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%Y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null || stat -c "%y" "$TEST_REPO/.aether/aether-utils.sh" 2>/dev/null)

if [[ "$MTIME_BEFORE" == "$MTIME_AFTER" ]]; then
    log_test_pass "Update --dry-run doesn't modify files" "File mtime unchanged"
else
    log_test_fail "Update --dry-run doesn't modify files" "File mtime unchanged" "File was modified"
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
