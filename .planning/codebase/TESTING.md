---
last_mapped_commit: 92252d01
---

# Testing Patterns

**Analysis Date:** 2026-08-22

## Test Framework

**Runner:**
- Go standard `testing` package (`import "testing"`)
- No external assertion libraries (no testify, no gomega)
- Test binaries: `go test ./...`

**Run Commands:**
```bash
go test ./...              # Run all tests (cached)
go test ./... -count=1     # Run all tests, no cache (CI standard)
go test ./... -race        # Run with race detector (CI mandatory)
go test ./... -race -count=1 -timeout 2400s  # Full CI mode
go test ./... -v           # Verbose output with test names
```

**Test Timeout:**
- Standard tests: `-timeout 900s` (15 minutes)
- Race detector tests: `-timeout 2400s` (40 minutes)
- Custom timeouts for specific slow tests

## Test File Organization

**Location:**
- `cmd/*_test.go` — Co-located with cmd modules (402 test files, ~3600 tests)
- `pkg/**/*_test.go` — Co-located with pkg modules (102 test files, ~980 tests)
- Total: ~4600 tests across the codebase

**Naming:**
- Test functions: `TestDescriptionOfWhatIsBeingTested`
- Test helpers: lowercase, e.g., `saveGlobals()`, `resetRootCmd()`, `createTestColonyState()`
- Setup functions: `setup{Feature}Test()` (e.g., `setupBuildFlowTest()`)

**Structure:**
```
cmd/
├── feature.go
├── feature_test.go        # Test functions, helpers, fixtures
├── feature_helper.go      # Optional: large helper functions extracted
└── testdata/              # Fixture files, golden files
    ├── fixtures/
    └── golden/
```

## Test Structure

**Global Setup/Teardown (TestMain):**

Located in `cmd/testing_main_test.go`. Every test suite execution:
1. Saves all mutable package-level globals (store, stdout, stderr, flags, etc.)
2. Sets `AETHER_OUTPUT_MODE=json` to isolate tests from human output
3. Creates isolated test hub directory (`AETHER_HUB_DIR`) so tests don't touch the real hub
4. Unsets model environment variables to prevent developer shell settings leaking into golden fixtures
5. Cleans up test git worktrees after suite finishes
6. Restores all globals

**Per-Test Setup:**

Every test that modifies globals calls `saveGlobals(t)` as its first action:

```go
func TestSomething(t *testing.T) {
	saveGlobals(t)           // Captures current globals, restores on t.Cleanup()
	resetRootCmd(t)          // For Cobra tests; resets flags and command state
	t.Setenv("VAR", "value") // Override specific env vars (cleared on test exit)
	
	// ... test logic ...
}
```

**Helper Functions Pattern:**

```go
func helperFunction(t *testing.T, args string) string {
	t.Helper()  // Marks this as a helper so test failures report the caller's line, not helper's
	// ... setup or utility logic ...
	return result
}
```

**Test Cleanup:**

```go
func TestSomething(t *testing.T) {
	saveGlobals(t)  // Uses t.Cleanup() internally
	
	tmpDir := t.TempDir()  // Auto-cleaned after test
	
	t.Cleanup(func() {
		// Manual cleanup code here
	})
	
	// ... test logic ...
}
```

## Test Patterns

### Basic Assertion Pattern

```go
func TestSomething(t *testing.T) {
	result, err := someFunction()
	
	// Assertion for errors:
	if err != nil {
		t.Fatalf("someFunction failed: %v", err)  // Fatal if setup fails
	}
	
	// Logical assertion:
	if result != expected {
		t.Errorf("result = %v, want %v", result, expected)  // Error for test failure
	}
}
```

**Rule:** Use `t.Fatalf()` for setup/precondition failures. Use `t.Errorf()` for test assertions. Never use `t.Skipf()` in production code being tested.

### Setup with Temp Files

```go
func TestFileOperations(t *testing.T) {
	saveGlobals(t)
	
	tmpDir := t.TempDir()  // Auto-cleaned after test
	testFile := filepath.Join(tmpDir, "test.json")
	
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	
	// Use testFile in test...
}
```

### Environment Variable Isolation

```go
func TestWithEnv(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")  // Set for this test only
	// t.Setenv automatically reverts after test
	
	// ... test logic using AETHER_OUTPUT_MODE ...
}
```

### Testing Cobra Commands

```go
func TestCommand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)  // Must reset Cobra state
	var buf bytes.Buffer
	stdout = &buf  // Redirect command output
	
	rootCmd.SetArgs([]string{"command-name", "arg1", "arg2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "expected text") {
		t.Errorf("output missing expected text: %s", output)
	}
}
```

## Mocking

**Framework:** No external mocking library. Use two strategies:

### Strategy 1: Global Variable Replacement

For `store`, `stdout`, `stderr`, and other package globals:

```go
func TestWithMockStore(t *testing.T) {
	saveGlobals(t)  // Saves original store
	
	mockStore := setupTestStore(t)  // Create test store
	store = mockStore  // Replace global
	
	// Test runs with mock store
	// saveGlobals() automatically restores original on test exit
}
```

### Strategy 2: Dependency Injection

For complex objects, pass mocks as parameters:

```go
func TestWithMockWorker(t *testing.T) {
	mockInvoker := &MockCodexWorkerInvoker{
		// Set up behavior
	}
	
	result := functionTakingInvoker(context.Background(), mockInvoker)
	
	if !mockInvoker.WasCalled {
		t.Error("expected invoker to be called")
	}
}
```

**What to Mock:**
- Global mutable state (`store`, `stdout`, `stderr`)
- External service clients (when testing integration points)
- Time-dependent code (use fixed times in tests)
- File I/O (use `t.TempDir()` instead when possible)

**What NOT to Mock:**
- Standard library functions (test them directly)
- Business logic in the same package (test the real code)
- Core data structures (use real structs with test fixtures)

## Fixtures and Factories

**Test Data Factories:**

Simple helper functions that create test objects:

```go
func judgementPhase(name, description string, mode colony.PhaseMode) colony.Phase {
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
	}
}

// Usage in tests:
phase := judgementPhase("Add endpoint", "Implement /hello", colony.PhaseModePrototype)
```

**Fixture Files:**

Located in `cmd/testdata/`:
- Fixture data files (JSON, YAML, markdown)
- Golden files for output comparison
- Real example inputs for integration tests

**Fixture Loading:**

```go
func TestWithFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	
	// Use state in test...
}
```

## Special Test Types

### Dry-Run Purity Tests

Tests that verify `--dry-run` / `--inspect` flags don't mutate state:

```go
func TestConsolidationPhaseEndDryRunDoesNotMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	
	dataDir := seedConsolidationFixture(t)
	before, _ := os.ReadFile(filepath.Join(dataDir, "QUEEN.md"))
	
	rootCmd.SetArgs([]string{"consolidation-phase-end", "--dry-run"})
	rootCmd.Execute()  // Dry-run should not modify state
	
	after, _ := os.ReadFile(filepath.Join(dataDir, "QUEEN.md"))
	
	if !bytes.Equal(before, after) {
		t.Error("dry-run mutated state when it should not have")
	}
}
```

**Rule:** Every inspection/report command must have a corresponding `TestXDryRunDoesNotMutate` test.

### Invariant Tests

Tests that assert proportions or relationships rather than checking for named sections:

```go
// Good: asserts an invariant
func TestEveryBuildSelectableCasteCanDispatch(t *testing.T) {
	dispatchable := map[string]bool{...}
	
	for _, profile := range casteRelevanceRegistry {
		if !casteAllowedForFlow(profile.Caste, "build") {
			continue
		}
		if !dispatchable[profile.Caste] {
			t.Errorf("caste %q selectable but not dispatchable", profile.Caste)
		}
	}
}

// Better than: just checking for caste name in a list
// Bad would be: for _, name := range casteName; if name == "builder" {...}
```

**Philosophy:** From CLAUDE.md Definition of Done: "Prefer a test that asserts a proportion or an invariant over one that asserts a section exists."

### Wiring Tests

Tests that verify a feature has actual callers (not orphaned):

```go
// TestSealProbeRequiresTestableCode pins that a Probe is only required
// when seal produces code that can be tested.
func TestSealProbeRequiresTestableCode(t *testing.T) {
	// Verify that probeRequiredForSeal() is called during seal
	// and that it correctly filters for phases with testable code
	
	phase := colony.Phase{Mode: colony.PhaseModeDocumentation}
	if probeRequiredForSeal(phase) {
		t.Error("docs-only phase incorrectly requires Probe")
	}
}
```

**Rule:** Every feature must have a wiring test proving it has callers. Orphaned code (no callers) is detected in CI.

### Parity Tests

Tests run separately to verify consistency across implementations:

```bash
go test ./cmd/... -run "TestParity|TestGoOnly" -count=1 -v
```

These verify that:
- Go implementation matches TS host behavior
- Command line flags match between platforms
- Output formats are consistent

## Coverage

**Requirements:** No explicit coverage percentage enforced by config.

**Philosophy:** From CLAUDE.md Definition of Done: "A requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet." Coverage is measured by the existence of executable verification, not by a coverage percentage.

**Coverage Calculation:** (No CI step, but available via):
```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Retired Tests

Documented in `.aether/docs/retired-tests-ledger.md` with format:

| Field | Example |
|-------|---------|
| Original path | `cmd/autopilot_test.go` |
| What it covered | "the `autopilot-check-replan` subcommand's interval arithmetic" |
| Disposition | `recovered-by:cmd/compatibility_cmds_test.go (TestRunAutopilotReplanDue)` OR `dead-with-no-replacement` |
| Removed in | `v5.4.0-richness restoration, Stage 2 (autopilot)` |

**Rule (RETIRE-04):** No test leaves the repository without a recorded disposition. Use `recovered-by:` if its coverage survived in another test, or `dead-with-no-replacement` if deliberately removed.

## CI Testing

**CI Pipeline** (`.github/workflows/ci.yml`):

1. **Build:** `go build ./cmd/aether`
2. **Vet:** `go vet ./...`
3. **Standard tests:** `go test ./... -count=1 -timeout 900s`
4. **Race tests:** `go test ./... -race -count=1 -timeout 2400s`
5. **Parity tests:** `go test ./cmd/... -run "TestParity|TestGoOnly" -count=1 -v`
6. **Wiring tests:** 50+ specific wiring/verification tests (extensive CLI audits)
7. **Binary smoke test:** Verify built binary runs and reports correct version
8. **Package verification:** TS packages, npm bootstrap tests

**Key Test Classes in CI:**

- **Wiring tests** (50+ tests): `TestNoRegisteredSubcommandIsUnreferenced`, `TestOrphanAllowlistOnlyShrinks`, `TestRatchetDetectsASyntheticOrphan`, etc.
- **Command wiring:** `TestCommandCallsMatchCobraContracts`, `TestAuditDetectsPositionalDrift`, `TestCLIFlagAudit`
- **Spawn/dispatch:** `TestSpawnCanSpawnAcceptsDocumentedInvocation`, `TestSpawnTreeDepthReportsTwoForAThreeLevelTree`
- **Release gates:** `TestReleaseGateCommandFailsATreeWithAFailingTest`, `TestReleaseGateWorkflowActuallyRuns`

## Testing Strategy

**Definition of Done** (from CLAUDE.md):
A requirement is satisfied only when **a command exists that someone can run, and that command fails when the requirement is unmet.**

This means:
- ✅ Tests verify executable behavior (commands that fail/pass)
- ✅ Tests assert proportions and invariants (not just "section exists")
- ✅ Wiring tests prove features have real callers
- ✅ Dry-run tests verify inspection commands don't mutate
- ❌ Tests don't just check for named sections or hard-coded strings
- ❌ Tests don't rely on documentation claims alone

**Practical Application:**

When testing a feature:
1. Write a test that executes the command/function
2. Verify the actual observable behavior (output, state change, error)
3. If testing a requirement, verify the command that enforces it fails when violated
4. Document retired tests in the ledger with their disposition

---

*Testing analysis: 2026-08-22*
