# Testing Patterns

**Analysis Date:** 2026-02-17

## Test Framework

**JavaScript Runner:**
- AVA (`ava`) v6.0.0
- Config: `package.json` ava section
- Timeout: 30 seconds per test

**Shell Runner:**
- shellcheck for linting
- Custom bash test framework in `tests/bash/test-helpers.sh`

**Run Commands:**
```bash
npm test              # Run all tests (unit + bash)
npm run test:unit     # Run AVA unit tests only
npm run test:bash     # Run shell tests only
npm run lint          # Run all linters (shell, json, sync)
npm run lint:shell    # Run shellcheck only
```

## Test File Organization

**Location:**
- Unit tests: `tests/unit/*.test.js`
- Integration tests: `tests/integration/*.test.js`
- E2E tests: `tests/e2e/*.test.js`
- Shell tests: `tests/bash/*.sh`

**Naming:**
- JavaScript: `{feature-name}.test.js` (e.g., `cli-hash.test.js`, `state-guard.test.js`)
- Shell: `test-{feature}.sh` (e.g., `test-aether-utils.sh`)

**Structure:**
```
tests/
├── unit/
│   ├── cli-hash.test.js
│   ├── state-guard.test.js
│   ├── helpers/
│   │   └── mock-fs.js
│   └── ...
├── integration/
│   ├── state-guard-integration.test.js
│   └── file-lock-integration.test.js
├── e2e/
│   ├── update-rollback.test.js
│   └── checkpoint-update-build.test.js
└── bash/
    ├── test-helpers.sh
    ├── test-aether-utils.sh
    └── test-session-freshness.sh
```

## Test Structure

**AVA Test Pattern:**
```javascript
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

test.before(() => {
  // Setup mocks once before all tests
});

test.afterEach(() => {
  // Reset state between tests
});

test.after(() => {
  sinon.restore();
});

// Test with serial execution when order matters
test.serial('test name', async t => {
  // Arrange
  const input = 'test';

  // Act
  const result = functionUnderTest(input);

  // Assert
  t.is(result, expected);
});
```

**Shell Test Pattern:**
```bash
#!/usr/bin/env bash
set -euo pipefail

source "$SCRIPT_DIR/test-helpers.sh"

test_help() {
    local output
    output=$(bash "$SCRIPT" help 2>&1)
    local exit_code=$?

    if ! assert_exit_code $exit_code 0; then
        test_fail "exit code 0" "exit code $exit_code"
        return 1
    fi

    if ! assert_json_valid "$output"; then
        test_fail "valid JSON" "invalid JSON"
        return 1
    fi

    return 0
}
```

## Mocking

**Framework:** sinon + proxyquire

**What to Mock:**
- File system (`fs`): Use `proxyquire` to inject mock `fs`
- Child processes: Use `sinon.stub` on `execSync`
- Time/Date: Use `sinon.useFakeTimers()` for deterministic dates

**Mocking Example from `/Users/callumcowie/repos/Aether/tests/unit/cli-hash.test.js`:**
```javascript
const sinon = require('sinon');
const proxyquire = require('proxyquire');

let mockFs;
let cli;

test.before(() => {
  // Create mock fs with sinon stubs
  mockFs = {
    readFileSync: sinon.stub()
  };

  // Load cli.js with mocked fs
  cli = proxyquire('../../bin/cli.js', {
    fs: mockFs
  });
});

test('hashFileSync returns correct SHA-256 hash', t => {
  const content = 'hello world';
  mockFs.readFileSync.withArgs('/test/file.txt').returns(Buffer.from(content));

  const result = cli.hashFileSync('/test/file.txt');

  t.is(result, 'sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9');
});
```

**Test Helper: Mock FS from `/Users/callumcowie/repos/Aether/tests/unit/helpers/mock-fs.js`:**
```javascript
function createMockFs() {
  return {
    readFileSync: sinon.stub(),
    writeFileSync: sinon.stub(),
    existsSync: sinon.stub(),
    mkdirSync: sinon.stub(),
    // ... more stubs
  };
}

function setupMockFiles(mockFs, files) {
  // Set up mock files with content
  for (const [path, content] of Object.entries(files)) {
    mockFs.readFileSync.withArgs(path).returns(content);
    mockFs.existsSync.withArgs(path).returns(true);
  }
}
```

**What NOT to Mock:**
- Core business logic being tested
- Error handling paths (unless specifically testing error cases)
- Simple pure functions

## Fixtures and Factories

**Test Data:**
- Inline factory functions within test files
- Helper functions for creating valid state objects

**Example from `/Users/callumcowie/repos/Aether/tests/unit/state-guard.test.js`:**
```javascript
function createValidState(overrides = {}) {
  return {
    version: '3.0',
    current_phase: overrides.current_phase ?? 5,
    initialized_at: overrides.initialized_at ?? '2026-02-14T10:00:00Z',
    last_updated: '2026-02-14T10:00:00Z',
    goal: 'Test goal',
    state: 'ACTIVE',
    memory: { phase_learnings: [], decisions: [], instincts: [] },
    errors: { records: [], flagged_patterns: [] },
    signals: [],
    graveyards: [],
    events: overrides.events ?? [],
    ...overrides
  };
}

function createValidEvidence(overrides = {}) {
  return {
    checkpoint_hash: 'sha256:abc123def456',
    test_results: { passed: true, count: 10 },
    timestamp: overrides.timestamp ?? '2026-02-14T12:00:00Z',
    ...overrides
  };
}
```

**Location:**
- Co-located with tests in `tests/unit/helpers/`
- Inline factory functions for specific test files

## Coverage

**Requirements:** None explicitly enforced

**View Coverage:** No coverage tool configured (no nyc/istanbul)

**Note:** Tests focus on behavior verification rather than coverage metrics

## Test Types

**Unit Tests:**
- Scope: Individual functions, classes, modules
- Location: `tests/unit/`
- Use mocks to isolate from dependencies
- Example: `cli-hash.test.js` tests `hashFileSync` in isolation

**Integration Tests:**
- Scope: Multiple modules working together
- Location: `tests/integration/`
- Real file system operations in temp directories
- Example: `state-guard-integration.test.js` tests StateGuard with real locks

**E2E Tests:**
- Scope: Full CLI commands end-to-end
- Location: `tests/e2e/`
- Tests actual command execution
- Example: `checkpoint-update-build.test.js` tests full update flow

**Shell Tests:**
- Scope: Shell script functionality
- Location: `tests/bash/`
- Validates JSON output, exit codes
- Example: `test-aether-utils.sh` tests all aether-utils.sh subcommands

## Common Patterns

**Async Testing:**
```javascript
test.serial('advancePhase succeeds', async t => {
  const result = await guard.advancePhase(5, 6, evidence);
  t.is(result.status, 'transitioned');
});
```

**Error Testing:**
```javascript
test.serial('advancePhase throws without evidence', async t => {
  const error = await t.throwsAsync(
    async () => await guard.advancePhase(5, 6, null),
    { instanceOf: StateGuardError }
  );

  t.is(error.code, StateGuardErrorCodes.E_IRON_LAW_VIOLATION);
});
```

**Serial Tests (when order matters):**
```javascript
test.serial('test that must run after previous', t => {
  // Sequential execution
});
```

**Shell JSON Validation:**
```bash
assert_json_valid() {
    local output="$1"
    echo "$output" | jq -e . >/dev/null 2>&1
}
```

---

*Testing analysis: 2026-02-17*
