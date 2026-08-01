# Testing Patterns

**Analysis Date:** 2026-08-01

## Test Framework

### Go

**Runner:** Go's built-in `testing` package (no external test framework)
- Config: None (uses Go's native runner)
- Run commands:
  ```bash
  go test ./...                  # Run all tests
  go test ./... -race            # With race condition detection
  go test ./... -count=1         # Disable test result caching
  go test ./cmd -v               # Verbose output
  go test ./cmd -run TestName    # Run specific test by pattern
  go test ./cmd -timeout=10m     # Set timeout
  ```

**Assertion Library:** `testing.T` built-ins (no external assertion library)
- Error reporting: `t.Errorf()`, `t.Fatalf()`, `t.Fail()`
- Logging: `t.Logf()` (for development only, removed before commit)
- Cleanup: `t.Cleanup(func() { ... })` for teardown

### TypeScript/JavaScript

**Runner:** Node.js built-in `node:test` module (since Node.js 18+)
- Config: None (native Node.js)
- Run commands:
  ```bash
  node --test test/*.test.ts                      # Run all tests
  npm test                                        # Runs: node --import tsx --test test/*.test.ts
  AETHER_UPDATE_SNAPSHOTS=1 npm run test:update  # Update snapshots
  ```

**Assertion Library:** `node:assert/strict` (built-in strict assertions)
- Methods: `assert.equal()`, `assert.deepEqual()`, `assert.throws()`, `assert.match()`
- No external testing library; native Node.js assertions are sufficient

**Test Organization:**
```typescript
import { describe, it } from "node:test";
import assert from "node:assert/strict";

describe("feature name", () => {
  it("does X under condition Y", () => {
    const result = fn(input);
    assert.equal(result, expected);
  });
});
```

## Test File Organization

### Location

**Go:**
- Co-located with source: `feature.go` paired with `feature_test.go`
- In same package as the code being tested (e.g., `cmd` tests are in `cmd` package)
- Example: `cmd/flags.go` → `cmd/flags_test.go`

**TypeScript:**
- Co-located with source: `module.ts` paired with `module.test.ts`
- In same directory as source file
- Example: `src/spawn-orchestrator.ts` → `test/spawn-orchestrator.test.ts`

### Naming

**Test Functions (Go):**
- Pattern: `TestDescriptiveNameInPascalCase`
- Example: `TestAssumptionsAnalyzeCreatesAssumptions`, `TestBuildCompletionStageMakesWrapperResultResumableWithoutRedispatch`
- Descriptive names make test purpose clear without reading the body

**Test Functions (TypeScript):**
- Pattern: Descriptive string in quotes
- Example: `"passes argv arrays through without splitting arguments that contain spaces"`
- Describes the test behavior in natural language
- Grouped in `describe()` blocks for organization

**Helper Functions (Go):**
- Pattern: `setup<Context>`, `save<Item>`, `reset<Item>`
- Examples: `setupTestStore()`, `saveGlobals()`, `resetRootCmd()`, `setupBuildFlowTest()`
- Reusable across multiple tests in the same file or package

## Test Structure

### Go Test Pattern

```go
func TestFeatureName(t *testing.T) {
    // 1. Save globals (REQUIRED for tests modifying package-level state)
    saveGlobals(t)
    
    // 2. Reset shared state
    resetRootCmd(t)
    
    // 3. Setup test data
    dataDir := setupBuildFlowTest(t)
    defer os.RemoveAll(dataDir)
    
    // 4. Set environment/configuration
    os.Setenv("AETHER_OUTPUT_MODE", "json")
    
    // 5. Execute code under test
    rootCmd.SetArgs([]string{"command", "arg"})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("command failed: %v", err)
    }
    
    // 6. Parse and verify output
    env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
    if env["ok"] != true {
        t.Errorf("expected ok=true, got: %v", env["ok"])
    }
    
    // 7. Assert on results
    result := env["result"].(map[string]interface{})
    if len(result["items"].([]interface{})) != 3 {
        t.Errorf("expected 3 items, got %d", len(result["items"].([]interface{})))
    }
}
```

### TypeScript Test Pattern

```typescript
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createSpawnOrchestrator } from "../src/spawn-orchestrator.js";

describe("spawn orchestrator", () => {
  it("accepts spawns within remaining budget", () => {
    // 1. Setup test fixtures
    const orch = createSpawnOrchestrator({
      totalBudget: 5,
      consumedBudget: 3,
    });
    
    // 2. Prepare test data
    const claims = [
      { caste: "scout", task: "Research X" },
      { caste: "builder", task: "Build Y" },
    ];
    
    // 3. Execute code under test
    const result = orch.processClaims("Builder-01", 1, claims);
    
    // 4. Assert on results
    assert.equal(result.accepted.length, 2, "both claims should be accepted");
    assert.equal(result.rejected.length, 0, "no rejections expected");
    assert.equal(orch.remainingBudget, 0, "budget should be consumed");
  });

  it("rejects spawns exceeding budget", () => {
    const orch = createSpawnOrchestrator({
      totalBudget: 5,
      consumedBudget: 4,
    });
    
    const claims = [
      { caste: "scout", task: "Research A" },
      { caste: "builder", task: "Build B" },
    ];
    
    const result = orch.processClaims("Builder-01", 1, claims);
    
    assert.equal(result.accepted.length, 1, "only 1 claim fits budget");
    assert.equal(result.rejected.length, 1, "1 claim rejected");
    assert.equal(
      result.rejected[0].reason,
      "spawn budget exhausted",
      "rejection reason"
    );
  });
});
```

## Setup and Teardown

### Go Test Lifecycle

**TestMain (global setup/teardown):**
```go
func TestMain(m *testing.M) {
    // Global setup: isolate hub, set env vars
    origOutputMode, hadOutputMode := os.LookupEnv("AETHER_OUTPUT_MODE")
    os.Setenv("AETHER_OUTPUT_MODE", "json")
    
    // Create isolated test hub
    testHubDir, _ := os.MkdirTemp("", "aether-test-hub-")
    os.Setenv("AETHER_HUB_DIR", testHubDir)
    
    // Save all package globals
    origStore := store
    origStdout := stdout
    // ... save all mutables
    
    // Run all tests
    code := m.Run()
    
    // Global teardown
    os.RemoveAll(testHubDir)
    store = origStore
    stdout = origStdout
    // ... restore all globals
    
    os.Exit(code)
}
```

**Per-Test Setup (saveGlobals pattern):**
```go
func TestFeature(t *testing.T) {
    saveGlobals(t)  // Call at start; t.Cleanup() restores all globals
    
    // Now safe to modify: store, stdout, stderr, etc.
    store = testStore
    stdout = &buf
}
```

**Cleanup:**
- Use `t.Cleanup(func() { ... })` for per-test cleanup
- Use `defer` for temporary directory cleanup
- Example: `defer os.RemoveAll(tmpDir)`

### TypeScript Test Lifecycle

**No global TestMain equivalent; hooks not commonly used**
- Use function-level setup for fixtures
- Use `try/finally` if needed for cleanup

**Helpers (optional):**
```typescript
function captureStderr(fn: () => void): string {
  const chunks: string[] = [];
  const originalWrite = process.stderr.write.bind(process.stderr);
  process.stderr.write = (chunk: unknown, ...args: unknown[]) => {
    if (typeof chunk === "string") chunks.push(chunk);
    return originalWrite(chunk as string, ...(args as any[]));
  };
  try {
    fn();
  } finally {
    process.stderr.write = originalWrite;
  }
  return chunks.join("");
}
```

## Test Organization Patterns

### Table-Driven Tests (Go)

```go
func TestPhaseValidation(t *testing.T) {
    tests := []struct {
        name    string
        phase   int
        wantErr bool
        errMsg  string
    }{
        {"valid phase 1", 1, false, ""},
        {"zero phase invalid", 0, true, "phase must be >= 1"},
        {"negative invalid", -1, true, "phase must be >= 1"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePhase(tt.phase)
            if (err != nil) != tt.wantErr {
                t.Fatalf("unexpected error: %v", err)
            }
            if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
                t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
            }
        })
    }
}
```

### Error Path Testing

**Always test both success and error paths:**
```go
func TestLoadStateSuccess(t *testing.T) {
    // Happy path
    state, err := loadActiveColonyState()
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if state.Goal == nil {
        t.Fatal("expected goal to be set")
    }
}

func TestLoadStateNotInitialized(t *testing.T) {
    // Error path: no store initialized
    if err := os.RemoveAll(dataDir); err != nil {
        t.Fatalf("cleanup failed: %v", err)
    }
    
    _, err := loadActiveColonyState()
    if err == nil {
        t.Fatal("expected error when no colony initialized")
    }
    if !errors.Is(err, errNoColonyInitialized) {
        t.Errorf("expected errNoColonyInitialized, got: %v", err)
    }
}
```

## Mocking and Test Doubles

### Go Mocking Strategies

**1. Dependency Injection (preferred):**
```go
func processWithStore(store *storage.Store, data interface{}) error {
    return store.SaveJSON("file.json", data)
}

// In test:
testStore := setupTestStore(t)
err := processWithStore(testStore, testData)
```

**2. Global Variable Substitution (when necessary):**
```go
var store *storage.Store  // Package global

func TestFeatureWithMockStore(t *testing.T) {
    origStore := store
    defer func() { store = origStore }()
    
    store = mockStore  // Substitute for test
    // Test code here
}
```

**3. Interface Mocking (for complex dependencies):**
- Define small interfaces: `type Reader interface { Read() ([]byte, error) }`
- Inject implementations: test uses mock, production uses real
- Example: `type codexWorkerInvoker func(...) error`

### TypeScript Mocking Strategies

**1. Function Parameter Substitution (preferred):**
```typescript
export function runGoJSONCommand<T>(
  bridge: GoBridgeOptions,
  args: readonly string[],
  caller: GoJSONCaller = callGoJSON  // Default real caller, override in tests
): T {
  return caller<T>(bridge, assertSafeGoArgs(args));
}

// In test:
const result = runGoJSONCommand(bridge, args, <T>(_bridge, args): T => {
  captured = args;
  return { ok: true } as T;
});
```

**2. Inline Mocks (simple cases):**
```typescript
const mockCaller = <T>(_bridge: GoBridgeOptions, args: string[]): T => {
  return { ok: true, result: { count: 42 } } as T;
};
```

**3. Capturing Behavior (for verification):**
```typescript
let capturedArgs: string[] = [];
const captureArgs = <T>(_bridge: GoBridgeOptions, args: string[]): T => {
  capturedArgs = args;
  return { ok: true } as T;
};

runGoJSONCommand(bridge, ["plan", "Goal"], captureArgs);
assert.deepEqual(capturedArgs, ["plan", "Goal"]);
```

## Common Patterns and Helpers

### Go Test Helpers

**`setupTestStore(t *testing.T) *storage.Store`:**
- Creates isolated temporary storage for the test
- Returns ready-to-use store with default colony state
- Location: Any test file can call it; typically in `testing_main_test.go`

**`setupBuildFlowTest(t *testing.T) string`:**
- Sets up a complete build flow test environment
- Creates `.aether/data` directory structure
- Populates with sample COLONY_STATE.json
- Returns data directory path
- Used for integration tests covering plan → build → continue

**`saveGlobals(t *testing.T)`:**
- Must be called at start of every test that modifies: `store`, `stdout`, `stderr`, or command state
- Automatically restores all globals when test completes
- Prevents test pollution

**`resetRootCmd(t *testing.T)`:**
- Resets Cobra root command to clean state
- Removes all flags and arguments from previous test
- Called after `saveGlobals()`

**`parseEnvelope(t *testing.T, output string) map[string]interface{}`:**
- Parses JSON output envelope
- Handles JSON unmarshaling with error reporting
- Returns result for assertion

### TypeScript Test Helpers

**`captureStderr(fn: () => void): string`:**
- Redirects stderr during callback execution
- Returns captured stderr content
- Useful for testing error output or logs

**Factory Functions (within tests):**
```typescript
function makeOrchestrator(
  totalBudget: number,
  consumedBudget = 0
): SpawnOrchestrator {
  return createSpawnOrchestrator({
    goBinaryPath: "/usr/local/bin/aether",
    cwd: "/tmp/test",
    totalBudget,
    consumedBudget,
    currentDepth: 1,
  });
}
```

## Coverage and Quality Gates

### Test Coverage

**Scope:**
- Go: 379+ test files in `cmd/` and `pkg/` directories
- TypeScript: 88+ test files in `.aether/ts-host/test/`

**Requirements:**
- Critical path: 90%+ coverage (state machine transitions, build/continue/seal flow)
- Error paths: Tested for expected behaviors
- Integration tests: Full workflow coverage (init → plan → build → continue → seal)

**Coverage gaps are identified but not automatically blocked** — coverage is monitored, not enforced by CI gates.

### Quality Gates (Manual)

1. **Always parse JSON output for verification** — Don't assume success; parse envelope
2. **Reset command state between tests** — Use `saveGlobals()`, `resetRootCmd()`
3. **Clean up temporary resources** — Use `defer os.RemoveAll()` or `t.Cleanup()`
4. **Test both success and error paths** — Every command needs happy and sad path tests
5. **Race detection in local testing** — Run `go test ./... -race` before pushing

### CI Verification

**GitHub Actions runs on every PR:**
- `go test ./... -race` (Go tests with race detection)
- `npm test` (TypeScript tests)
- No automatic code coverage gates; manual review of coverage changes

## Test Types

### Unit Tests

**Scope:** Single function or small behavioral unit
**Approach:** Test pure functions with multiple inputs; mock external dependencies
**Example:** `TestPlanGranularityValid` tests the `Valid()` method with multiple inputs
**Location:** Co-located with source file

### Integration Tests

**Scope:** Multiple components working together (e.g., plan → build → continue flow)
**Approach:** Use real storage, create temporary environment, execute full workflows
**Example:** `TestBuildCompletionStageMakesWrapperResultResumableWithoutRedispatch` tests build completion with real JSON parsing
**Location:** In test files; uses `setupBuildFlowTest()` helpers

### End-to-End Tests

**Scope:** Full CLI workflow from init through seal
**Approach:** Launch binary or command, capture output, verify state changes
**Example:** Not common in this codebase; integration tests cover most E2E scenarios
**Location:** Rare; most E2E testing handled by integration tests

### Concurrency Tests

**Scope:** File locking, race conditions, parallel execution
**Approach:** Run `go test -race` for automatic detection; explicit tests for lock behavior
**Example:** `pkg/storage/lock_*.go` tests verify file locking across processes
**Location:** In `pkg/storage/` and related packages

## Running Tests

```bash
# Go tests
go test ./...                   # All tests
go test ./... -race             # With race detection
go test ./cmd -v                # Verbose
go test ./cmd -run TestName     # Specific test
go test -timeout=5m ./...       # Custom timeout

# TypeScript tests
npm test                        # All tests via node:test
npm run test:all                # Alternative syntax
AETHER_UPDATE_SNAPSHOTS=1 \
  npm run test:update           # Update snapshots
```

## Test Data and Fixtures

### Go Test Data

**Embedded data:** Use `//go:embed` for bundled fixtures (e.g., testdata files)
**Temporary data:** Use `os.TempDir()` or `t.TempDir()` for isolated test directories
**Example:**
```go
func TestLoadTestData(t *testing.T) {
    tmpDir := t.TempDir()  // Auto-cleaned after test
    testFile := filepath.Join(tmpDir, "test.json")
    os.WriteFile(testFile, []byte(`{"key":"value"}`), 0644)
    // Test with testFile
}
```

### TypeScript Test Data

**Fixture objects:** Define inline (no separate fixture files in this codebase)
**Mock data:** Create simple objects or factory functions as needed
**Example:**
```typescript
const mockBuildManifest: BuildManifest = {
  phase: 1,
  phase_name: "Phase 1",
  root: "/test/repo",
  colony_depth: "1",
  // ... required fields
};
```

---

*Testing analysis: 2026-08-01*
