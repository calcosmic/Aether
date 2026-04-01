# Testing Patterns

**Analysis Date:** 2026-04-01

The Aether codebase has three distinct test ecosystems: **bash integration tests**, **AVA unit tests (JavaScript)**, and **Go tests**. There is no single test runner that covers all three.

---

## Test Runners

### Bash Tests

**Runner:** Custom bash test harness (no external framework)

**Helper library:** `tests/bash/test-helpers.sh`

**Run commands:**
```bash
npm test                    # Runs AVA unit tests only
npm run test:unit           # AVA unit tests
npm run test:bash           # Main bash integration test suite (test-aether-utils.sh)
npm run test:skills         # Skills test suite
npm run test:intelligence   # Intelligence test suite
npm run test:all            # All three npm test suites (unit + bash + intelligence)
bash tests/bash/test-trust-scoring.sh   # Individual bash test file
```

### AVA Unit Tests (JavaScript)

**Runner:** AVA v6

**Config:** `package.json` `"ava"` section
```json
{
  "ava": {
    "files": ["tests/unit/**/*.test.js"],
    "timeout": "30s"
  }
}
```

**Mocking libraries:**
- `sinon` v19 -- stubs, spies, mocks
- `proxyquire` v2 -- module dependency injection for mocking `fs`, `path`, etc.

### Go Tests

**Runner:** Go standard `testing` package

**Run commands:**
```bash
go test ./...                # All Go tests
go test ./pkg/colony/        # Package-specific
go test -race ./pkg/storage/ # Race detection
go test -v ./...             # Verbose
```

---

## Test File Organization

### Bash Tests

**Location:** `tests/bash/`

**Naming convention:** `test-<module>.sh` or `test-<feature>.sh`
- `test-trust-scoring.sh`, `test-event-bus.sh`, `test-instinct-store.sh` (module tests)
- `test-colony-prime-budget.sh`, `test-hive-confidence-boost.sh` (feature tests)
- `test-learning-observe-trust.sh` (integration tests)

**E2E tests:** `tests/e2e/`
- `test-<area>.sh` (e.g., `test-lifecycle.sh`, `test-pher.sh`, `test-xml.sh`)
- Shared helpers in `tests/e2e/e2e-helpers.sh`
- Runner: `tests/e2e/run-all-e2e.sh`

**Integration tests:** `tests/integration/`
- `test-colony-depth.sh`

**Total bash test files:** ~100 files across `tests/bash/`, `tests/e2e/`, `tests/integration/`

### AVA Unit Tests

**Location:** `tests/unit/`

**Naming convention:** `<module>.test.js`
- `colony-state.test.js`, `state-guard.test.js`, `file-lock.test.js`

**Test helpers:** `tests/unit/helpers/mock-fs.js`

**Total AVA test files:** ~40 files

### Go Tests

**Location:** Co-located with source files (Go standard)

**Naming convention:** `<package>_test.go` in the same directory as source
- `pkg/colony/colony_test.go`
- `pkg/storage/storage_test.go`

**Root-level smoke test:** `golang_test.go` -- verifies all packages compile via blank imports

---

## Bash Test Structure

### Test Harness (`tests/bash/test-helpers.sh`)

Provides a lightweight test framework with:

**Setup/teardown:**
```bash
setup_test_env        # Creates temp dir with COLONY_STATE.json fixture
teardown_test_env     # Removes temp dir
run_test "func_name"  # Runs a test function with pass/fail tracking
run_test_with_env "func_name"  # Setup + run + teardown
```

**Assertions (return 0 on success, non-zero on failure):**
```bash
assert_json_valid "$json"
assert_json_field_equals "$json" ".field" "expected"
assert_ok_true "$json"           # Checks .ok == "true"
assert_ok_false "$json"          # Checks .ok == "false"
assert_exit_code $actual $expected
assert_json_has_field "$json" "field_name"
assert_json_array_length "$json" ".array" 5
assert_contains "$haystack" "needle"
assert_file_exists "$path"
assert_dir_exists "$path"
```

**Logging:**
```bash
log "message"
log_info "message"    # Blue prefix
log_warn "message"    # Yellow prefix
log_error "message"   # Red prefix
test_summary          # Prints pass/fail/total, returns 1 if any failures
```

**Utility:**
```bash
run_aether_utils "subcommand" "arg1" "arg2"  # Runs aether-utils.sh and captures output
require_jq                                   # Exits if jq not installed
```

### Typical Bash Test File Pattern

```bash
#!/usr/bin/env bash
# Module Tests for <module>
# Tests <module>.sh functions via aether-utils.sh subcommands

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$SCRIPT_DIR/test-helpers.sh"
require_jq

AETHER_UTILS="$REPO_ROOT/.aether/aether-utils.sh"

# --- Helper: isolated env ---
setup_<module>_env() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/.aether/data"
    cp "$AETHER_UTILS" "$tmpdir/.aether/aether-utils.sh"
    cp -r "$(dirname "$AETHER_UTILS")/utils" "$tmpdir/.aether/"
    echo "$tmpdir"
}

run_cmd() {
    local tmpdir="$1"; shift
    AETHER_ROOT="$tmpdir" DATA_DIR="$tmpdir/.aether/data" \
        bash "$tmpdir/.aether/aether-utils.sh" "$@" 2>/dev/null || true
}

# --- Test functions ---
test_feature_description() {
    local tmpdir
    tmpdir=$(setup_<module>_env)
    local result
    result=$(run_cmd "$tmpdir" subcommand --arg value)
    rm -rf "$tmpdir"
    assert_ok_true "$result" || return 1
    local field
    field=$(echo "$result" | jq -r '.result.field')
    [[ "$field" == "expected" ]] || return 1
}

# --- Main ---
log_info "Running <module> tests..."
run_test "test_feature_description" "descriptive name for output"
run_test "test_another_feature" "another descriptive name"
test_summary
```

### E2E Test Pattern

E2E tests use `tests/e2e/e2e-helpers.sh` which extends `test-helpers.sh`:

```bash
setup_e2e_env          # Creates isolated temp dir with full aether structure
teardown_e2e_env       # Cleanup
run_in_isolated_env "$tmpdir" "subcommand" "args"
extract_json "$raw_output"  # Strips non-JSON prefix lines
init_results           # Initialize requirement tracking
record_result "REQ-ID" "PASS|FAIL" "notes"
print_area_results "Area Name"  # Markdown table output
```

### Integration Test Pattern

Integration tests in `tests/integration/` use a self-contained pattern without the shared test harness:

```bash
#!/bin/bash
set -euo pipefail
pass=0; fail=0; total=0
TMPDIR_BASE=$(mktemp -d)
trap 'rm -rf "$TMPDIR_BASE"' EXIT

setup_colony() { ... }
run_cmd() { ... }
assert_eq() {
    total=$((total + 1))
    if [[ "$1" == "$2" ]]; then
        echo "  PASS: $3"; pass=$((pass + 1))
    else
        echo "  FAIL: $3"; fail=$((fail + 1))
    fi
}
```

### Test Isolation

Bash tests create **isolated temp directories** that replicate the `.aether/` structure:
1. `mktemp -d` creates a clean temp directory
2. Copy `aether-utils.sh` and `utils/` into it
3. Set `AETHER_ROOT` and `DATA_DIR` environment variables
4. Run commands against the isolated copy
5. `rm -rf` cleanup (via trap EXIT or explicit cleanup)

This prevents tests from modifying the real colony state.

---

## AVA Unit Test Structure

### Suite Organization

```javascript
const test = require('ava');
const sinon = require('sinon');
const proxyquire = require('proxyquire');

// Module-level shared state
let mockFs;
let StateGuard;

// Lifecycle hooks
test.before(() => {
  mockFs = createMockFs();
  const module = proxyquire('../../bin/lib/state-guard.js', { fs: mockFs });
  StateGuard = module.StateGuard;
});

test.afterEach(() => {
  resetMockFs(mockFs);  // Reset stubs between tests
});

test.after(() => {
  sinon.restore();
});

// Individual test cases
test('descriptive name', (t) => {
  const result = someFunction();
  t.is(result, expected);
});
```

### Mocking Pattern

**fs mocking** via `tests/unit/helpers/mock-fs.js`:
```javascript
const { createMockFs, setupMockFiles, resetMockFs } = require('./helpers/mock-fs');

let mockFs = createMockFs();  // Returns sinon-stubbed fs object
mockFs.readFileSync.returns('{"key":"value"}');

// Inject into module under test
const module = proxyquire('../../bin/lib/state-guard.js', { fs: mockFs });
```

### Fixture Patterns

```javascript
function createValidState(overrides = {}) {
  return {
    version: '3.0',
    current_phase: 5,
    state: 'ACTIVE',
    memory: { phase_learnings: [], decisions: [], instincts: [] },
    ...overrides
  };
}
```

### Assertion Patterns

AVA uses `t.is()`, `t.true()`, `t.false()`, `t.deepEqual()`, `t.throws()`:
```javascript
t.is(result.phase, 2);
t.true(guard.isValid);
t.deepEqual(state.memory.instincts, []);
t.throws(() => guard.advance(), { instanceOf: StateGuardError });
```

---

## Go Test Structure

### Table-Driven Tests (Primary Pattern)

```go
func TestValidTransitions(t *testing.T) {
    tests := []struct {
        from State
        to   State
    }{
        {StateREADY, StateEXECUTING},
        {StateEXECUTING, StateBUILT},
        {StateBUILT, StateREADY},
    }
    for _, tt := range tests {
        t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
            if err := Transition(tt.from, tt.to); err != nil {
                t.Fatalf("expected no error for %s->%s, got: %v", tt.from, tt.to, err)
            }
        })
    }
}
```

### Test Naming

- `Test<Noun>_<Condition>`: `TestAdvancePhase_FirstPending`, `TestAdvancePhase_NoMorePending`
- `Test<Noun>_<Action>`: `TestColonyDepthField`, `TestRoundTripRealColonyState`
- Subtests via `t.Run("description", ...)`

### Helper Functions

Test helpers defined at bottom of test file, NOT in separate helper packages (except `internal/testing`):
```go
func strPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }
func parseTime(s string) time.Time {
    t, err := time.Parse(time.RFC3339, s)
    if err != nil { panic(err) }
    return t
}
```

Custom assertion helpers use `t.Helper()`:
```go
func assertColonyStateEqual(t *testing.T, a, b ColonyState) {
    t.Helper()
    if a.Version != b.Version {
        t.Errorf("Version mismatch: %q vs %q", a.Version, b.Version)
    }
}
```

### Test Isolation

- `t.TempDir()` for filesystem-based tests (auto-cleaned)
- `newTestStore(t)` pattern for creating test fixtures:
```go
func newTestStore(t *testing.T) (*Store, string) {
    t.Helper()
    dir := t.TempDir()
    s, err := NewStore(dir)
    if err != nil { t.Fatalf("NewStore(%q): %v", dir, err) }
    return s, dir
}
```

### Skipping Tests

Tests that depend on real colony data use `t.Skip()`:
```go
data, err := os.ReadFile("../../.aether/data/COLONY_STATE.json")
if err != nil {
    t.Skip("COLONY_STATE.json not found, skipping round-trip test")
}
```

### Error Testing

```go
// Negative test: expect error
func TestInvalidTransitions(t *testing.T) {
    err := Transition(StateCOMPLETED, StateREADY)
    if err == nil { t.Fatal("expected error") }
}

// Error chain check
func TestTransitionErrorIs(t *testing.T) {
    err := Transition(StateCOMPLETED, StateREADY)
    if !errors.Is(err, ErrInvalidTransition) {
        t.Errorf("expected ErrInvalidTransition, got: %v", err)
    }
}
```

### JSON Round-Trip Testing

Go tests validate that types serialize/deserialize correctly against real JSON files:
```go
func TestRoundTripRealColonyState(t *testing.T) {
    data, _ := os.ReadFile("../../.aether/data/COLONY_STATE.json")
    var original ColonyState
    json.Unmarshal(data, &original)
    remarshaled, _ := json.Marshal(original)
    var rematched ColonyState
    json.Unmarshal(remarshaled, &rematched)
    assertColonyStateEqual(t, original, rematched)
}
```

### Concurrency Testing

```go
func TestConcurrentWrites_NoRace(t *testing.T) {
    s, dir := newTestStore(t)
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            _ = s.SaveJSON(path, data)
        }(i)
    }
    wg.Wait()
}
```

---

## Coverage

**Bash:** No automated coverage measurement. Coverage gaps tracked manually.

**AVA:** No coverage threshold configured. No `nyc` or `c8` in devDependencies.

**Go:** No coverage threshold configured. Available via:
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

---

## Test Count Summary

| Test Type | Framework | Approximate Count |
|-----------|-----------|-------------------|
| Bash unit/integration | Custom harness | ~90 files, 500+ test cases |
| E2E | Custom harness | ~20 files |
| AVA unit tests | AVA v6 | ~40 files |
| Go tests | testing package | 3 test files, ~60 test cases |
| **Total** | | **~524+ passing** |

---

## Common Patterns Across Test Types

### JSON Response Validation

All three test ecosystems validate the `{"ok": true/false}` envelope:
- Bash: `assert_ok_true "$result"`
- AVA: `t.is(response.ok, true)`
- Go: Not applicable (Go tests validate struct fields directly)

### Isolated Environment

All test types create temporary, isolated environments:
- Bash: `mktemp -d` with `.aether/` structure copy
- AVA: `proxyquire` with mock `fs`
- Go: `t.TempDir()`

### Floating-Point Comparisons

Bash tests use `awk` or `bc -l` for floating-point comparisons:
```bash
[[ $(echo "$score >= 0.95" | bc -l 2>/dev/null || awk "BEGIN{print ($score >= 0.95)}") == "1" ]]
```

---

*Testing analysis: 2026-04-01*
