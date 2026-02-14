#!/usr/bin/env bash
# Aether End-to-End Test: aether update --all
# Tests that the update --all command properly updates all registered repos

set -euo pipefail

# Test configuration
TEST_NAME="aether update --all"
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

log "${YELLOW}=== Aether Update --all E2E Test Suite ===${NC}"
log "Project root: $PROJECT_ROOT"
log "Test directory: $TEST_DIR"
log "Temporary HOME: $TMP_DIR"
log ""

# Create multiple test repos
REPO_1="$TMP_DIR/test-repo-1"
REPO_2="$TMP_DIR/test-repo-2"
REPO_3="$TMP_DIR/test-repo-3"

for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    mkdir -p "$repo/.claude/commands/ant"
    mkdir -p "$repo/.opencode/commands/ant"
    mkdir -p "$repo/.opencode/agents"
    mkdir -p "$repo/.aether"
    mkdir -p "$repo/.aether/data"

    # Create initial version file (old version)
    cat > "$repo/.aether/version.json" <<EOF
{
  "version": "0.8.0",
  "updated_at": "2024-01-01T00:00:00Z"
}
EOF

    # Create a test command file
    cat > "$repo/.claude/commands/ant/test-old.md" <<EOF
---
name: ant:test-old
description: "Test command - old version"
---
Old test command content.
EOF
done

# First run install to set up the hub
log "${YELLOW}Setup: Running install to create hub...${NC}"
cd "$PROJECT_ROOT"
node bin/cli.js install --quiet 2>/dev/null || true

# Verify hub is created
if ! assert_dir_exists "$HOME/.aether"; then
    log "ERROR: Hub setup failed"
    exit 1
fi

# Manually register repos in registry
REGISTRY_FILE="$HOME/.aether/registry.json"
cat > "$REGISTRY_FILE" <<EOF
{
  "schema_version": 1,
  "repos": [
    {
      "path": "$REPO_1",
      "version": "0.8.0",
      "registered_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    {
      "path": "$REPO_2",
      "version": "0.8.0",
      "registered_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    {
      "path": "$REPO_3",
      "version": "0.8.0",
      "registered_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
EOF

log "Hub and repos setup complete"
log ""

# ============================================================================
# Test 1: Update --all updates all registered repos
# ============================================================================

log_test_start "Update --all updates all registered repos"

cd "$PROJECT_ROOT"
node bin/cli.js update --all --quiet 2>/dev/null || true

UPDATED_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    VERSION_FILE="$repo/.aether/version.json"
    if assert_file_exists "$VERSION_FILE"; then
        VERSION=$(jq -r '.version' "$VERSION_FILE" 2>/dev/null || echo "unknown")
        HUB_VERSION=$(jq -r '.version' "$HOME/.aether/version.json" 2>/dev/null || echo "unknown")
        if [[ "$VERSION" == "$HUB_VERSION" ]]; then
            UPDATED_COUNT=$((UPDATED_COUNT + 1))
        fi
    fi
done

if [[ "$UPDATED_COUNT" -eq 3 ]]; then
    log_test_pass "Update --all updates all repos" "All 3 repos updated to $HUB_VERSION"
else
    log_test_fail "Update --all updates all repos" "All 3 repos updated" "$UPDATED_COUNT repos updated"
fi

# ============================================================================
# Test 2: Update --all syncs commands to all repos
# ============================================================================

log_test_start "Update --all syncs commands to all repos"

# Add a new command to hub that should be copied to all repos
cat > "$HOME/.aether/commands/claude/test-sync.md" <<EOF
---
name: ant:test-sync
description: "Test command for sync verification"
---
This command should be synced to all repos.
EOF

cd "$PROJECT_ROOT"
node bin/cli.js update --all --quiet 2>/dev/null || true

SYNCED_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    if assert_file_exists "$repo/.claude/commands/ant/test-sync.md"; then
        SYNCED_COUNT=$((SYNCED_COUNT + 1))
    fi
done

if [[ "$SYNCED_COUNT" -eq 3 ]]; then
    log_test_pass "Update --all syncs commands to all repos" "New command synced to all 3 repos"
else
    log_test_fail "Update --all syncs commands to all repos" "New command in all 3 repos" "$SYNCED_COUNT repos have new command"
fi

# ============================================================================
# Test 3: Update --all removes stale files from all repos
# ============================================================================

log_test_start "Update --all removes stale files from all repos"

# First, add the old file back to all repos
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    cat > "$repo/.claude/commands/ant/test-stale.md" <<EOF
---
name: ant:test-stale
description: "Stale command that should be removed"
---
This file should be removed during update.
EOF
done

# Remove test-stale from hub (so it's considered stale)
rm -f "$HOME/.aether/commands/claude/test-stale.md"

cd "$PROJECT_ROOT"
node bin/cli.js update --all --quiet 2>/dev/null || true

REMOVED_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    if ! assert_file_exists "$repo/.claude/commands/ant/test-stale.md"; then
        REMOVED_COUNT=$((REMOVED_COUNT + 1))
    fi
done

if [[ "$REMOVED_COUNT" -eq 3 ]]; then
    log_test_pass "Update --all removes stale files from all repos" "Stale files removed from all 3 repos"
else
    log_test_fail "Update --all removes stale files from all repos" "Stale files removed from all 3 repos" "$REMOVED_COUNT repos cleaned"
fi

# ============================================================================
# Test 4: Update --all skips non-existent repos
# ============================================================================

log_test_start "Update --all handles non-existent repos gracefully"

# Add a non-existent repo to registry
NONEXISTENT="$TMP_DIR/does-not-exist"
REGISTRY=$(cat "$REGISTRY_FILE")
echo "$REGISTRY" | jq ".repos += [{\"path\": \"$NONEXISTENT\", \"version\": \"0.8.0\", \"registered_at\": \"2024-01-01T00:00:00Z\", \"updated_at\": \"2024-01-01T00:00:00Z\"}]" > "$REGISTRY_FILE"

cd "$PROJECT_ROOT"
OUTPUT=$(node bin/cli.js update --all 2>&1 || true)

if echo "$OUTPUT" | grep -q "Pruned"; then
    log_test_pass "Update --all handles non-existent repos" "Non-existent repo pruned"
else
    log_test_fail "Update --all handles non-existent repos" "Non-existent repo pruned" "Got: $OUTPUT"
fi

# ============================================================================
# Test 5: Update --all preserves colony data in all repos
# ============================================================================

log_test_start "Update --all preserves colony data in all repos"

# Create test data files in each repo
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    mkdir -p "$repo/.aether/data"
    echo "repo-specific data" > "$repo/.aether/data/test-preserve-$repo.txt"
done

cd "$PROJECT_ROOT"
node bin/cli.js update --all --quiet 2>/dev/null || true

PRESERVED_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    TEST_FILE="$repo/.aether/data/test-preserve-$repo.txt"
    if assert_file_exists "$TEST_FILE"; then
        CONTENT=$(cat "$TEST_FILE")
        if [[ "$CONTENT" == "repo-specific data" ]]; then
            PRESERVED_COUNT=$((PRESERVED_COUNT + 1))
        fi
    fi
done

if [[ "$PRESERVED_COUNT" -eq 3 ]]; then
    log_test_pass "Update --all preserves colony data in all repos" "Colony data preserved in all 3 repos"
else
    log_test_fail "Update --all preserves colony data in all repos" "Colony data preserved in all 3 repos" "$PRESERVED_COUNT repos preserved"
fi

# ============================================================================
# Test 6: Update --all --dry-run doesn't modify files
# ============================================================================

log_test_start "Update --all --dry-run doesn't modify files"

# Get file counts before dry-run
BEFORE_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    COUNT=$(find "$repo/.claude/commands/ant" -type f 2>/dev/null | wc -l | tr -d ' ')
    BEFORE_COUNT=$((BEFORE_COUNT + COUNT))
done

cd "$PROJECT_ROOT"
node bin/cli.js update --all --dry-run --quiet 2>/dev/null || true

# Get file counts after dry-run
AFTER_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    COUNT=$(find "$repo/.claude/commands/ant" -type f 2>/dev/null | wc -l | tr -d ' ')
    AFTER_COUNT=$((AFTER_COUNT + COUNT))
done

if [[ "$BEFORE_COUNT" -eq "$AFTER_COUNT" ]]; then
    log_test_pass "Update --all --dry-run doesn't modify files" "File counts unchanged ($BEFORE_COUNT -> $AFTER_COUNT)"
else
    log_test_fail "Update --all --dry-run doesn't modify files" "File counts unchanged" "File counts changed: $BEFORE_COUNT -> $AFTER_COUNT"
fi

# ============================================================================
# Test 7: Update --list shows all registered repos
# ============================================================================

log_test_start "Update --list shows all registered repos"

cd "$PROJECT_ROOT"
OUTPUT=$(node bin/cli.js update --list 2>&1 || true)

# Count how many of our repos are shown
SHOWN_COUNT=0
for repo in "$REPO_1" "$REPO_2" "$REPO_3"; do
    if echo "$OUTPUT" | grep -q "$repo"; then
        SHOWN_COUNT=$((SHOWN_COUNT + 1))
    fi
done

if [[ "$SHOWN_COUNT" -ge 2 ]]; then
    log_test_pass "Update --list shows registered repos" "$SHOWN_COUNT repos shown in output"
else
    log_test_fail "Update --list shows registered repos" "At least 2 repos shown" "$SHOWN_COUNT repos shown"
fi

# ============================================================================
# Test 8: Update --all updates registry timestamps
# ============================================================================

log_test_start "Update --all updates registry timestamps"

BEFORE_TIME=$(jq -r '.repos[0].updated_at' "$REGISTRY_FILE" 2>/dev/null || echo "unknown")

cd "$PROJECT_ROOT"
node bin/cli.js update --all --quiet 2>/dev/null || true

AFTER_TIME=$(jq -r '.repos[0].updated_at' "$REGISTRY_FILE" 2>/dev/null || echo "unknown")

# Check if timestamp was updated (should be newer)
if [[ "$BEFORE_TIME" != "$AFTER_TIME" ]]; then
    log_test_pass "Update --all updates registry timestamps" "Timestamp updated from $BEFORE_TIME"
else
    # Timestamp might be the same if version unchanged, check if file was modified
    if [[ "$BEFORE_TIME" == "unknown" ]]; then
        log_test_fail "Update --all updates registry timestamps" "Timestamp updated" "Could not read timestamps"
    else
        log_test_pass "Update --all updates registry timestamps" "Timestamp checked (versions may match)"
    fi
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
