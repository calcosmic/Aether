# Testing Patterns

**Analysis Date:** 2026-02-17

## Test Framework Overview

| Type | Framework | Location | Run Command |
|------|-----------|----------|-------------|
| Unit Tests | AVA | `tests/unit/` | `npm run test:unit` or `npm test` |
| Shell Tests | Custom bash runner | `tests/bash/` | `npm run test:bash` |
| Integration | AVA | `tests/integration/` | `npm test` |
| E2E Tests | AVA | `tests/e2e/` | Manual execution |

## Unit Testing (AVA)

### Framework Details

**Version:** AVA 6.x
**Config:** `package.json` ava section
**Timeout:** 30 seconds per test
**Assertion Library:** AVA built-in

### Run Commands

```bash
npm test              # Run all tests (unit + bash)
npm run test:unit    # Run only AVA unit tests
npm run test:bash    # Run shell script tests
```

### Test File Organization

**Location Pattern:** Tests are co-located with their module conceptually:
- `bin/lib/state-guard.js` → `tests/unit/state-guard.test.js`
- `bin/lib/errors.js` → tested inline in `tests/unit/update-errors.test.js`

**Naming:** `*.test.js`

### Test Structure

```javascript
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

test.before(() => {
  // Setup shared mocks or state
});

test.after(() => {
  // Cleanup after all tests
});

test.afterEach(() => {
  // Reset mocks between tests
});

test.serial('test description', async t => {
  // Arrange
  const input = 'test value';

  // Act
  const result = await functionUnderTest(input);

  // Assert
  t.is(result.expected, actual);
});
```

### Mocking Strategy

**Primary Tools:**
- `sinon` - Stubs, spies, and mocks
- `proxyquire` - Module dependency injection
- Custom `mock-fs` helper - Filesystem mocking

**Example: Using proxyquire for Module Mocking**

```javascript
// tests/unit/state-guard.test.js
const proxyquire = require('proxyquire');
const sinon = require('sinon');

let mockFs;
let StateGuard;

test.before(() => {
  // Create mock fs with sinon stubs
  mockFs = {
    existsSync: sinon.stub(),
    readFileSync: sinon.stub(),
    writeFileSync: sinon.stub(),
    // ... other fs methods
  };

  // Load module with mocked fs
  const module = proxyquire('../../bin/lib/state-guard.js', {
    fs: mockFs
  });

  StateGuard = module.StateGuard;
});
```

**Custom Mock FS Helper**

The project provides `tests/unit/helpers/mock-fs.js` for common filesystem operations:

```javascript
const { createMockFs, setupMockFiles, resetMockFs } = require('./helpers/mock-fs');

test.before(() => {
  mockFs = createMockFs();

  // Setup test files
  setupMockFiles(mockFs, {
    '/test/COLONY_STATE.json': JSON.stringify(validState),
    '/test/locks': null  // directory
  });
});

test.afterEach(() => {
  resetMockFs(mockFs);
});
```

### Test Fixtures

**Helper Functions:**

Tests use helper functions to create valid test data:

```javascript
function createValidState(overrides = {}) {
  return {
    version: '3.0',
    current_phase: overrides.current_phase ?? 5,
    initialized_at: overrides.initialized_at ?? '2026-02-14T10:00:00Z',
    last_updated: '2026-02-14T10:00:00Z',
    goal: 'Test goal',
    state: 'ACTIVE',
    memory: {
      phase_learnings: [],
      decisions: [],
      instincts: []
    },
    errors: { records: [], flagged_patterns: [] },
    signals: [],
    graveyards: [],
    events: overrides.events ?? [],
    ...overrides
  };
}
```

### Assertions Reference

AVA provides these assertion methods:

| Method | Use For |
|--------|---------|
| `t.is(a, b)` | Strict equality |
| `t.deepEqual(a, b)` | Deep equality (objects/arrays) |
| `t.truthy(value)` | Truthy check |
| `t.falsy(value)` | Falsy check |
| `t.true(value)` | Strict true |
| `t.false(value)` | Strict false |
| `t.throws(fn)` | Exception thrown |
| `t.throwsAsync(fn)` | Async exception thrown |
| `t.notThrows(fn)` | No exception |
| `t.notThrowsAsync(fn)` | No async exception |
| `t.regex(str, regex)` | Regex match |
| `t.snapshot(actual)` | Snapshot comparison |

### Serial vs Parallel Tests

Use `test.serial()` when tests modify shared state (files, globals):

```javascript
test.serial('advancePhase succeeds with valid evidence', async t => {
  // Test that modifies COLONY_STATE.json
});
```

Use regular `test()` for truly independent tests.

## Shell Testing

### Test Runner

Custom bash test runner in `tests/bash/`:

```bash
#!/bin/bash
# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper: Run test and track result
run_test() {
  local name="$1"
  local expected="$2"
  local actual="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ "$actual" == *"$expected"* ]]; then
    echo "PASS: $name"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    echo "FAIL: $name"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}
```

### Shell Test Location

| Test File | Tests |
|-----------|-------|
| `tests/bash/test-session-freshness.sh` | Session freshness detection |
| `tests/bash/test-aether-utils.sh` | Main utilities |
| `tests/bash/test-xml-utils.sh` | XML utilities |
| `tests/bash/test-pheromone-xml.sh` | Pheromone XML parsing |

### Shell Test Patterns

**Setup/Temp Directory:**

```bash
setup_tmpdir() {
  mktemp -d
}

cleanup_tmpdir() {
  local dir="$1"
  rm -rf "$dir"
}
```

**Environment Variables for Test Context:**

```bash
test_verify_fresh_stale() {
  local tmpdir=$(setup_tmpdir)

  # Create test files
  touch -t 202501010000 "$tmpdir/test-file.md"

  # Run test with environment variable
  local result
  result=$(SURVEY_DIR="$tmpdir" bash "$UTILS_SCRIPT" session-verify-fresh --command survey "" "$(date +%s)")

  # Assert
  run_test "verify_fresh_stale_ok_false" '"ok":false' "$result"

  cleanup_tmpdir "$tmpdir"
}
```

## Integration Testing

**Location:** `tests/integration/`

Integration tests verify interactions between modules:

```javascript
// tests/integration/state-guard-integration.test.js
test('StateGuard integrates with FileLock correctly', async t => {
  // Full integration scenario
});
```

## E2E Testing

**Location:** `tests/e2e/`

End-to-end tests verify complete workflows:

```javascript
// tests/e2e/checkpoint-update-build.test.js
test('complete checkpoint update build flow', async t => {
  // Full workflow from start to finish
});
```

## Linting

**Shell Linting:**
```bash
npm run lint:shell    # shellcheck on .aether scripts
```

**JSON Validation:**
```bash
npm run lint:json     # Validate JSON files
```

**Sync Verification:**
```bash
npm run lint:sync     # Verify command sync between Claude Code and OpenCode
```

## Coverage

**Current Approach:** No enforced coverage target, but tests expected for:
- New code features
- Bug fixes (regression tests)
- CLI commands

## Best Practices

1. **Test Independence:** Each test should run independently
2. **Cleanup:** Use `teardown` or `afterEach` for resource cleanup
3. **Mock External Dependencies:** Don't rely on actual filesystem or network
4. **Clear Assertions:** Use descriptive assertion messages
5. **Arrange-Act-Assert:** Structure tests clearly
6. **Fixture Reuse:** Create helper functions for common test data

---

*Testing analysis: 2026-02-17*
