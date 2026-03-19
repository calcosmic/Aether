# Testing Patterns

**Analysis Date:** 2026-03-19

## Test Framework

**Runner:**
- AVA v6.0.0 — JavaScript test runner
- Config: `package.json` (lines 40-44)
- 47 test files covering unit, integration, bash, and e2e scenarios

**Assertion Library:**
- AVA's built-in assertions (`.is()`, `.deepEqual()`, `.true()`, etc.)
- No external assertion library needed

**Run Commands:**
```bash
npm test              # Run all unit tests (ava)
npm run test:unit    # Run AVA tests only
npm run test:all     # Run unit tests + bash integration tests
npm run test:bash    # Run bash integration tests only
```

**Test Configuration (`package.json` lines 40-44):**
```json
"ava": {
  "files": ["tests/unit/**/*.test.js"],
  "timeout": "30s"
}
```

## Test File Organization

**Location:**
- Unit tests: `tests/unit/**/*.test.js` — 47 test files as of analysis
- Bash integration tests: `tests/bash/test-aether-utils.sh` — shell tests
- Test helpers: `tests/unit/helpers/` — reusable mocking utilities
- E2E tests directory exists: `tests/e2e/` (structure present, content may be minimal)
- Integration tests directory: `tests/integration/` (present)

**Naming:**
- Pattern: `{module-name}.test.js` (example: `file-lock.test.js`, `state-sync.test.js`)
- Co-located with source: test file reflects source file it tests

**Structure Examples:**
```
tests/unit/
├── file-lock.test.js              # Tests for bin/lib/file-lock.js
├── state-sync.test.js             # Tests for bin/lib/state-sync.js
├── update-transaction.test.js      # Tests for bin/lib/update-transaction.js
├── model-profiles.test.js          # Tests for bin/lib/model-profiles.js
└── helpers/
    └── mock-fs.js                  # Reusable mocking utilities
```

## Test Structure

**Suite Organization:**
Each test file uses AVA's flat structure (no nested describe blocks). Tests organized by section comments.

Example from `tests/unit/file-lock.test.js` (lines 14-87):
```javascript
test.before(() => {
  sandbox = sinon.createSandbox();
});

test.beforeEach((t) => {
  sandbox.restore();
  t.context.mockFs = createMockFs();
  // Default setup
  t.context.mockFs.existsSync.withArgs('.aether/locks').returns(true);
  // Load module with mocks
  const { FileLock } = loadFileLock(t.context.mockFs);
  t.context.FileLock = FileLock;
});

test.afterEach(() => {
  sandbox.restore();
});

// Test cases
test.serial('acquire creates lock file atomically', (t) => {
  // Setup
  // Execute
  // Assert
});
```

**Patterns:**
1. **Setup:** `test.beforeEach()` runs before each test (resets stubs, creates fresh mocks)
2. **Isolation:** `test.serial()` runs tests sequentially to avoid stub conflicts
3. **Cleanup:** `test.afterEach()` restores stubs to prevent cross-test pollution
4. **Context:** `t.context` stores test-specific state (mocks, fixtures)

## Mocking

**Framework:** Sinon v19.0.5 + Proxyquire v2.1.3

**Patterns:**

### Sinon Stubs
Create isolated units by stubbing dependencies. Example from `tests/unit/file-lock.test.js` (lines 18-29):
```javascript
function createMockFs() {
  return {
    existsSync: sandbox.stub(),
    readFileSync: sandbox.stub(),
    writeFileSync: sandbox.stub(),
    openSync: sandbox.stub(),
    closeSync: sandbox.stub(),
    unlinkSync: sandbox.stub(),
    mkdirSync: sandbox.stub(),
    readdirSync: sandbox.stub(),
  };
}
```

### Proxyquire for Dependency Injection
Inject mocked dependencies when loading modules. Example from `tests/unit/file-lock.test.js` (lines 32-36):
```javascript
function loadFileLock(mockFs) {
  return proxyquire('../../bin/lib/file-lock.js', {
    fs: mockFs,
  });
}
```

### Mock Configuration
Configure stub behavior with chainable API:
```javascript
mockFs.existsSync.withArgs('.aether/locks').returns(true);
mockFs.openSync.withArgs('.aether/locks/state.json.lock', 'wx').returns(1);
mockFs.readFileSync.withArgs('.aether/locks/state.json.lock.pid', 'utf8').returns('12345');
```

### Assertions on Stubs
Verify calls were made correctly:
```javascript
t.true(mockFs.openSync.calledWith('.aether/locks/state.json.lock', 'wx'));
t.true(mockFs.writeFileSync.calledWith(1, process.pid.toString(), 'utf8'));
```

**What to Mock:**
- File system operations (`fs` module)
- Child process calls (`execSync`, `execFileSync`)
- External services (HTTP, database)
- Time-dependent operations (use fake timers)
- Process-level operations (environment variables, process.env)

**What NOT to Mock:**
- Core business logic (test actual implementation)
- Data validation and transformation
- Error handling and recovery paths
- Format conversion (JSON, YAML)

## Fixtures and Factories

**Test Data:**
State fixtures created inline or via factory functions. Example from `tests/unit/state-loader.test.js` (lines 13-22):
```javascript
const MINIMAL_VALID_STATE = JSON.stringify({
  version: "3.0",
  goal: "CI test colony",
  state: "READY",
  current_phase: 1,
  plan: { phases: [{ id: 1, name: "Test Phase", status: "pending" }] },
  events: [],
  errors: { records: [] },
  memory: { decisions: [], learnings: [] }
}, null, 2);
```

**Factories:**
Helper functions create test objects. From `tests/unit/update-transaction.test.js` (lines 6-34):
```javascript
const createMockFs = () => ({
  existsSync: sinon.stub(),
  readFileSync: sinon.stub(),
  // ... all fs methods
});

const createMockCrypto = () => ({
  createHash: sinon.stub().returns({
    update: sinon.stub().returns({
      digest: sinon.stub().returns('abc123hash'),
    }),
  }),
});
```

**Location:**
- Fixtures and factories defined at top of test file (after imports)
- Organized by logical grouping (e.g., all filesystem mocks together)
- Comments explain complex fixtures

**Backup/Restore Pattern:**
For tests that modify real files, use backup helpers. From `tests/unit/state-loader.test.js` (lines 49-75):
```javascript
function backupState() {
  const backupPath = `${COLONY_STATE_PATH}.backup`;
  const hadState = fs.existsSync(COLONY_STATE_PATH);
  if (hadState) {
    fs.copyFileSync(COLONY_STATE_PATH, backupPath);
  }
  return { backupPath, hadState };
}

function restoreState({ backupPath, hadState }) {
  if (hadState && fs.existsSync(backupPath)) {
    fs.copyFileSync(backupPath, COLONY_STATE_PATH);
    fs.unlinkSync(backupPath);
  } else if (!hadState && fs.existsSync(COLONY_STATE_PATH)) {
    fs.unlinkSync(COLONY_STATE_PATH);
  }
}

// In test:
const backup = backupState();
// ... test code that modifies state ...
restoreState(backup);
```

## Coverage

**Requirements:** No target enforced (coverage optional)

**View Coverage:** No npm script currently (can add `npm run test:coverage` with `c8`)

**Current State:**
- 47 unit tests provide broad coverage of core modules
- Critical modules: file-lock, state-sync, update-transaction (heavily tested)
- Model routing, telemetry, validation (moderate coverage)

**Test Count by Module:**
- Unit tests: 47 files covering ~15 core modules
- Bash integration tests: comprehensive coverage of aether-utils.sh
- Coverage gaps: see CONCERNS.md for areas needing more tests

## Test Types

**Unit Tests:**
- Scope: Single function/class in isolation
- Location: `tests/unit/**/*.test.js`
- Pattern: Mock all external dependencies via proxyquire + sinon
- Example: `file-lock.test.js` tests FileLock class methods individually
- Typical size: 100-300 lines per test file

**Integration Tests:**
- Scope: Multiple components working together
- Location: `tests/integration/` (directory exists, tests may be minimal)
- Pattern: May use real fs or limited mocking
- Focus: state synchronization, transaction flows

**Bash Tests:**
- Scope: Shell functions in aether-utils.sh
- Location: `tests/bash/test-aether-utils.sh`
- Pattern: Source the shell file, call functions, validate JSON output
- Tools: jq for JSON validation, bash test helpers
- Example from `tests/bash/test-helpers.sh`:
  - `assert_json_valid()` — verify valid JSON
  - `assert_json_field_equals()` — check field values
  - `assert_exit_code()` — verify exit codes

**E2E Tests:**
- Scope: Full command execution end-to-end
- Location: `tests/e2e/` (directory present, content minimal)
- Pattern: Not yet extensively populated
- Future: Would test full `aether init` → `aether build` workflows

## Common Patterns

**Async Testing:**

Unit tests are synchronous by default. For async operations, use test.serial() with async callbacks:

```javascript
test.serial('acquireAsync returns true on successful lock', async (t) => {
  const fileLock = new FileLock();
  mockFs.existsSync.returns(false);

  const result = await fileLock.acquireAsync('/test/file.json');

  t.true(result);
});
```

From `tests/unit/file-lock.test.js` pattern (lines 148+):
- Use `async (t) =>` for async test callbacks
- Use `await` for promises
- Sinon stubs work normally with async code

**Error Testing:**

Verify errors are thrown with correct code and message:

```javascript
test('throws ValidationError on invalid state', (t) => {
  const error = t.throws(() => {
    validateStateSchema(null); // Invalid input
  });

  t.is(error.code, ErrorCodes.E_INVALID_STATE);
  t.true(error.message.includes('must be a non-null object'));
});
```

From error testing pattern in `tests/unit/state-sync.test.js`:
- Use `t.throws()` to assert exception
- Check error properties: `code`, `message`, `details`, `recovery`
- Verify error can be serialized: `t.deepEqual(error.toJSON(), ...)`

**Stub Assertions:**

Verify dependencies were called correctly:

```javascript
test('writeFileSync called with correct lock data', (t) => {
  fileLock.acquire('/test/file.json');

  // First call
  const writeCall = mockFs.writeFileSync.getCall(0);
  t.is(writeCall.args[0], 1);  // fd
  t.is(writeCall.args[1], String(process.pid)); // content

  // Count total calls
  t.is(mockFs.writeFileSync.callCount, 2); // pid + lock file
});
```

**Spy Pattern:**

Track calls without changing behavior:

```javascript
const spy = sinon.spy(console, 'warn');
getModelForCaste({}, 'unknown');
t.true(spy.calledWithMatch('Unknown caste'));
spy.restore();
```

## Bash Test Helpers

Located at `tests/bash/test-helpers.sh`, provides utilities for shell integration tests:

**Execution Helper:**
```bash
execWithLoader() {
  local command="$1"
  source "${STATE_LOADER_PATH}" 2>/dev/null && $command
}
```

**Assertion Functions:**
- `assert_json_valid "$output"` — verify valid JSON
- `assert_json_field_equals "$json" ".field" "value"` — check field
- `assert_json_has_field "$json" "fieldname"` — verify field exists
- `assert_exit_code $code 0` — verify exit code
- `assert_contains "$str" "substring"` — string contains

**Test Execution:**
```bash
test_start "test name"
# ... test code ...
if assert_condition; then
  test_pass
else
  test_fail "expected" "got"
fi
```

**Counters and Reporting:**
```bash
# Test runs automatically track:
TESTS_RUN          # Total tests run
TESTS_PASSED       # Passed count
TESTS_FAILED       # Failed count

# Summary at end:
log_info "Passed: $TESTS_PASSED/$TESTS_RUN"
log_error "Failed: $TESTS_FAILED"
```

## Testing Best Practices

1. **One assertion per test (when possible)** — Clearer failure messages
2. **Use descriptive test names** — "acquire creates lock file atomically" not "test1"
3. **Setup in beforeEach** — Consistent test state, easier to read
4. **Clean up in afterEach** — Prevent test pollution
5. **Test error paths** — Not just happy path
6. **Isolate with serial** — When tests share mocked state (sinon sandbox)
7. **Mock external deps** — File system, HTTP, child_process
8. **Test return values and side effects** — Both matter
9. **Use meaningful assertions** — `t.is()` with exact values, `t.deepEqual()` for objects
10. **Document complex test setup** — Comments explain non-obvious fixtures

## Known Test Patterns in Codebase

**Schema Validation Testing (`state-sync.test.js`):**
- Custom validation function tests all field requirements
- Error collection pattern: returns array of validation errors
- Edge cases: null values, wrong types, missing required fields

**Lock Testing (`file-lock.test.js`):**
- Stale process detection using process.kill(pid, 0)
- Atomic file operations with exclusive flag
- Timeout and retry logic with clock mocking

**Transaction Testing (`update-transaction.test.js`):**
- State machine testing (PENDING → IN_PROGRESS → COMPLETE)
- Error recovery with git stash
- Checkpoint creation and restoration

**Telemetry Testing (`telemetry.test.js`):**
- Temp directory setup/cleanup for isolation
- JSON file I/O with corruption handling
- Circular dependency detection in routing

---

*Testing analysis: 2026-03-19*
