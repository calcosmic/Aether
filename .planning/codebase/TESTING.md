# Testing Patterns

**Analysis Date:** 2026-02-13

## Test Framework

**Runner:**
- No formal test framework (Node.js native assertions + shell scripts)
- Unit tests: Node.js scripts in `test/` directory
- E2E tests: Bash scripts in `tests/e2e/` directory

**Assertion Library:**
- Node.js: Custom assertion functions with `console.log` pass/fail output
- Shell: Custom `assert_file_exists`, `assert_dir_exists`, `get_file_hash` functions

**Run Commands:**
```bash
# Unit tests (hash comparison)
node test/sync-dir-hash.test.js

# Unit tests (user modification detection)
node test/user-modification-detection.test.js

# Unit tests (namespace isolation)
node test/namespace-isolation.test.js

# E2E tests - run all
cd tests/e2e && ./run-all.sh

# E2E tests - individual suites
cd tests/e2e && ./test-install.sh
cd tests/e2e && ./test-update.sh
cd tests/e2e && ./test-update-all.sh

# Linting (part of test verification)
npm run lint
```

## Test File Organization

**Location:**
- Unit tests: `test/` directory (separate from source)
- E2E tests: `tests/e2e/` directory

**Naming:**
- Unit tests: `*.test.js` suffix (e.g., `sync-dir-hash.test.js`)
- E2E tests: `test-*.sh` prefix (e.g., `test-install.sh`)

**Structure:**
```
test/                                    # Unit tests
├── sync-dir-hash.test.js               # Hash comparison unit tests
├── user-modification-detection.test.js # User modification detection tests
└── namespace-isolation.test.js         # Namespace isolation verification

tests/e2e/                              # End-to-end tests
├── README.md                           # Test documentation
├── run-all.sh                          # Test runner (runs all test-*.sh)
├── test-install.sh                     # aether install E2E tests
├── test-update.sh                      # aether update E2E tests
└── test-update-all.sh                  # aether update --all E2E tests
```

## Test Structure

**Suite Organization (Node.js unit tests):**
```javascript
// Test utilities - setup/teardown with temp directories
function setupTestDirs() {
  const testDir = path.join(__dirname, 'test-sync-temp');
  const srcDir = path.join(testDir, 'src');
  const destDir = path.join(testDir, 'dest');

  // Clean up any existing test dirs
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }

  fs.mkdirSync(srcDir, { recursive: true });
  fs.mkdirSync(destDir, { recursive: true });

  return { testDir, srcDir, destDir };
}

function cleanupTestDirs(testDir) {
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
}

// Test case pattern
console.log('Test 1: Same content in src and dest - should skip copy');
{
  const { testDir, srcDir, destDir } = setupTestDirs();
  try {
    // Setup test data
    fs.writeFileSync(path.join(srcDir, 'file.txt'), 'hello world');
    fs.writeFileSync(path.join(destDir, 'file.txt'), 'hello world');

    // Execute function under test
    const result = syncDirWithCleanupNew(srcDir, destDir);

    // Assert expected outcome
    if (result.copied === 0 && result.skipped === 1) {
      console.log('  PASS: No copy when hashes match\n');
      passed++;
    } else {
      console.log(`  FAIL: Expected copied=0, skipped=1, got copied=${result.copied}, skipped=${result.skipped}\n`);
      failed++;
    }
  } finally {
    cleanupTestDirs(testDir);
  }
}

// Summary output
console.log(`\n=== Results: ${passed} passed, ${failed} failed ===`);
process.exit(failed > 0 ? 1 : 0);
```

**Suite Organization (Bash E2E tests):**
```bash
#!/usr/bin/env bash
set -euo pipefail

# Test configuration with isolated HOME
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/../.." && pwd)"
TMP_DIR=$(mktemp -d)
export HOME="$TMP_DIR"  # Isolate test environment
CLEANUP_DONE=false

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Utility functions
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

# Assertion helpers
assert_file_exists() {
    [[ -f "$1" ]]
}

assert_dir_exists() {
    [[ -d "$1" ]]
}

# Cleanup trap
cleanup() {
    if [[ "$CLEANUP_DONE" == "true" ]]; then return; fi
    CLEANUP_DONE=true
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

# Test execution pattern
log_test_start "Install creates ~/.aether/ directory"
if assert_dir_exists "$HOME/.aether"; then
    log_test_pass "Install creates ~/.aether/" "~/.aether/ exists"
else
    log_test_fail "Install creates ~/.aether/" "~/.aether/ exists" "directory not found"
fi

# Summary
log "Tests run:    $TESTS_RUN"
log "Tests passed: $TESTS_PASSED"
if [[ "$TESTS_FAILED" -gt 0 ]]; then
    log "Tests failed:  $TESTS_FAILED"
    exit 1
fi
exit 0
```

**Patterns:**
- Setup: Create isolated temp directories with `mktemp -d`
- Teardown: `trap cleanup EXIT INT TERM` pattern for guaranteed cleanup
- Assertion: Custom pass/fail logging with counters
- Isolation: `export HOME="$TMP_DIR"` for E2E tests to avoid affecting real installation

## Mocking

**Framework:** None (custom implementations)

**Patterns:**
```javascript
// Copy implementation under test into test file for unit testing
function hashFileSync(filePath) {
  const content = fs.readFileSync(filePath);
  return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
}

// Create mock data structures instead of mocking
const manifest = {
  generated_at: '2026-01-01T00:00:00Z',
  files: {
    'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
  }
};
```

**What to Mock:**
- File system operations: Use real temp directories instead of mocking
- External processes: Avoid if possible; use real commands in E2E tests

**What NOT to Mock:**
- File system: Tests use real temp directories for realistic testing
- Node.js built-ins: Use actual `fs`, `path`, `crypto` modules

## Fixtures and Factories

**Test Data:**
```javascript
// Inline fixture creation
fs.writeFileSync(path.join(srcDir, 'config.txt'), 'v1');
fs.writeFileSync(path.join(destDir, 'config.txt'), 'user-changes');

// Manifest fixture
const manifest = {
  generated_at: '2026-01-01T00:00:00Z',
  files: {
    'config.txt': hashFileSync(path.join(srcDir, 'config.txt'))
  }
};
```

```bash
# Bash heredoc fixtures
cat > "$repo/.aether/version.json" <<EOF
{
  "version": "0.9.0",
  "updated_at": "2024-01-01T00:00:00Z"
}
EOF

cat > "$repo/.claude/commands/ant/test-old.md" <<EOF
---
name: ant:test-old
description: "Test command - old version"
---
Old test command content.
EOF
```

**Location:**
- Fixtures created inline within test functions
- No separate fixture files or directories
- Temp directories cleaned up after each test

## Coverage

**Requirements:** None enforced

**View Coverage:**
- Not configured - tests focus on pass/fail verification
- Manual coverage: Review test output for "Tests passed" count

## Test Types

**Unit Tests:**
- Scope: Individual functions (hash comparison, sync logic, user modification detection)
- Location: `test/*.test.js`
- Approach: Test function implementations copied into test files, isolated temp directories
- Focus: Single responsibility - one function/feature per test file

**Integration Tests:**
- Scope: CLI commands (`aether install`, `aether update`, `aether update --all`)
- Location: `tests/e2e/*.sh`
- Approach: Execute actual CLI commands against isolated environment
- Focus: Full command workflows with real file system operations

**E2E Tests:**
- Framework: Bash scripts with isolated `$HOME` environment
- Coverage:
  - `test-install.sh`: 12 tests covering hub setup, directory creation, file copying, idempotency
  - `test-update.sh`: 10 tests covering hub checks, version updates, command sync, hash comparison, dry-run
  - `test-update-all.sh`: 8 tests covering multi-repo updates, registry management, batch operations

## Common Patterns

**Async Testing:**
```javascript
// Not used - tests are synchronous with Node.js sync APIs
// If async needed, use callbacks or async/await with try/catch
```

**Error Testing:**
```javascript
// Test error conditions by checking output
const result = syncDirWithCleanup(srcDir, destDir);
if (result.userModifications === undefined) {
  console.log('  PASS: Backward compatible without manifest\n');
  passed++;
}
```

**Idempotency Testing:**
```bash
# Test that running command twice produces same result
BEFORE_COUNT=$(find "$HOME/.aether" -type f 2>/dev/null | wc -l)
node bin/cli.js install --quiet
AFTER_COUNT=$(find "$HOME/.aether" -type f 2>/dev/null | wc -l)

if [[ "$AFTER_COUNT" -ge "$BEFORE_COUNT" ]]; then
    log_test_pass "Install is idempotent" "Files: $BEFORE_COUNT -> $AFTER_COUNT (no data loss)"
fi
```

**Hash Comparison Testing:**
```bash
# Verify files aren't unnecessarily rewritten
MTIME_BEFORE=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh")
node "$PROJECT_ROOT/bin/cli.js" update --quiet
MTIME_AFTER=$(stat -f "%m" "$TEST_REPO/.aether/aether-utils.sh")

if [[ "$MTIME_BEFORE" == "$MTIME_AFTER" ]]; then
    log_test_pass "Hash comparison prevents unnecessary writes"
fi
```

**Dry-Run Testing:**
```bash
# Verify --dry-run doesn't modify files
MTIME_BEFORE=$(stat -f "%m" "$FILE")
node bin/cli.js update --dry-run
MTIME_AFTER=$(stat -f "%m" "$FILE")

if [[ "$MTIME_BEFORE" == "$MTIME_AFTER" ]]; then
    log_test_pass "--dry-run doesn't modify files"
fi
```

## Test Prerequisites

**Required tools:**
- `node` - Node.js runtime (v16+)
- `jq` - JSON processor for parsing version.json files
- `shasum` or `sha256sum` - File hash verification
- `shellcheck` - Shell script linting (via `npm run lint:shell`)

## Linting as Testing

**Shell script verification:**
```bash
npm run lint:shell
# Runs: shellcheck --severity=error on all .sh files
```

**JSON validation:**
```bash
npm run lint:json
# Validates: .aether/data/constraints.json, .aether/data/COLONY_STATE.json
```

**Sync verification:**
```bash
npm run lint:sync
# Runs: bash bin/generate-commands.sh check
```

## Key Testing Principles

1. **Isolation**: Every test uses isolated temp directories (`mktemp -d`) and isolated `$HOME`
2. **Cleanup**: `trap cleanup EXIT INT TERM` ensures temp directories are always removed
3. **Real operations**: Tests use real file system operations, not mocks
4. **Counters**: Track `TESTS_PASSED`, `TESTS_FAILED`, `TESTS_RUN` for summary reporting
5. **Exit codes**: Tests exit with code 1 on any failure, 0 on success
6. **Descriptive output**: Each test logs what it's testing, expected vs actual results

---

*Testing analysis: 2026-02-13*
