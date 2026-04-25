# Phase 51: Recovery Verification - Research

**Researched:** 2026-04-25
**Domain:** Go E2E testing, colony state recovery verification
**Confidence:** HIGH

## Summary

Phase 51 is a test-only phase that proves all 7 recovery paths in `aether recover` work correctly through E2E tests exercising the full `rootCmd.Execute()` path (not direct function calls). The existing codebase provides a mature E2E test pattern in `cmd/e2e_regression_test.go` and 61 unit tests in `cmd/recover_test.go` with reusable helpers. The key challenge is building seed functions that create specific broken colony states in temp directories and verifying that the full scan-repair-rescan pipeline produces clean results.

Three critical findings: (1) the existing `initRecoverTestStore` helper from `recover_test.go` can be reused directly since E2E tests in the same `cmd` package have access to all unexported helpers, (2) exit codes are surfaced via `fmt.Errorf("issues detected")` return from `runRecover` rather than `os.Exit`, so E2E tests can assert on `rootCmd.Execute()` error return, and (3) CR-02 from Phase 50 (unconditional state write in `repairDirtyWorktree`) was NOT fixed and is still live in the codebase -- the compound E2E test may expose this.

**Primary recommendation:** Write 10 E2E test functions in `cmd/e2e_recovery_test.go` using `saveGlobals` + `resetRootCmd` + `initRecoverTestStore` + `rootCmd.SetArgs` pattern. Each test seeds a specific broken state, runs the full command, and asserts on output, exit behavior, and post-recovery state.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** One test function per scenario following existing `e2e_regression_test.go` pattern -- `TestE2ERecoveryMissingBuildPacket`, `TestE2ERecoveryStaleSpawned`, etc. plus `TestE2ERecoveryCompoundState` and `TestE2ERecoveryHealthyColony`. Not table-driven -- each scenario has unique setup that would be awkward to parameterize.
- **D-02:** Tests go in a new file `cmd/e2e_recovery_test.go` to keep recovery tests isolated from the existing E2E regression suite.
- **D-03:** Helper functions in the test file that create specific broken states in temp directories. Each helper writes hand-crafted JSON/files that trigger one scanner. Examples: `seedMissingPacketState(t, dir)`, `seedStaleSpawnedState(t, dir)`, etc.
- **D-04:** Compound test uses a combined seeder that applies multiple seeds to the same temp dir.
- **D-05:** Post-recovery assertions check: (1) exit code is 0, (2) re-scan returns empty issue list (no remaining problems), (3) key state file content matches expected post-repair values.
- **D-06:** For repair tests (--apply), also verify the backup was created and the specific state mutation occurred (e.g., phase status changed from EXECUTING to READY).
- **D-07:** Two compound scenarios: (1) all 5 safe states simultaneously, (2) both destructive states simultaneously.
- **D-08:** The 7-individual-state tests cover scan-only (--apply=false) behavior. The compound tests cover --apply behavior to prove the full repair pipeline works end-to-end.
- **D-09:** `TestE2ERecoveryHealthyColony` creates a fully initialized, healthy colony state with no broken files and asserts: (1) exit code 0, (2) JSON output shows empty issues array, (3) text output contains "No stuck-state conditions detected" or equivalent.

### Claude's Discretion
- Exact helper function signatures and internal structure
- Whether to reuse existing test helpers from `recover_test.go` or write fresh E2E-specific ones
- Order of test execution
- Exact assertion messages on failure

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TEST-01 (R087) | E2E test proving recovery from each of the 7 stuck states individually | 7 scanner functions documented with exact trigger conditions; seed helpers can create each state; exit code and output patterns verified |
| TEST-02 (R088) | E2E test proving recovery from a compound stuck state (multiple issues simultaneously) | Compound scenarios feasible: safe states (5) can coexist; destructive states (2) require --force; rollback mechanism documented |
| TEST-03 (R089) | Test proving recover does not false-positive on an active, healthy colony | Healthy colony structure documented: needs valid COLONY_STATE.json with READY state, no spawn-runs.json, no survey flag, all agent files present |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Stuck-state detection | API / Backend (Go runtime) | -- | All 7 scanners run as Go functions in `cmd/recover_scanner.go` |
| State repair | API / Backend (Go runtime) | -- | All 7 repair functions in `cmd/recover_repair.go` |
| Backup/rollback | API / Backend (Go runtime) | -- | `createBackup`/`restoreFromBackup` in `cmd/medic_repair.go` |
| Output rendering | API / Backend (Go runtime) | -- | `renderRecoverDiagnosis`/`renderRecoverJSON` in `cmd/recover_visuals.go` |
| E2E test execution | Test (same package) | -- | Tests in `cmd/` package exercise `rootCmd` directly |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go testing | stdlib | Test framework | Already used by 2900+ tests in the codebase |
| cobra | existing | CLI command routing | `rootCmd.SetArgs` + `Execute` is the E2E test pattern |
| storage | pkg/storage | Colony state persistence | `storage.NewStore` + `store` package-level var |
| colony | pkg/colony | State types and transitions | `colony.ColonyState`, `colony.Transition` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | JSON marshal/unmarshal for assertions | Parsing `--json` output and state files |
| os/exec | stdlib | git commands (dirty worktree repair only) | Only destructive repair tests need this |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `rootCmd.SetArgs` + `Execute` | `exec.Command("go run ./cmd/aether", ...)` | Subprocess approach is more realistic but slower, harder to capture output, harder to set up temp dirs; same-package approach is the established pattern |

**Installation:**
No new packages needed. This phase is test-only with zero production code changes.

## Architecture Patterns

### System Architecture Diagram

```
E2E Test Function
    |
    v
saveGlobals(t)          -- snapshots and restores package-level vars
    |
    v
initRecoverTestStore(t) -- creates temp dir, storage.Store, sets AETHER_ROOT
    |
    v
Seed Helper (e.g., seedMissingPacketState) -- writes broken state to temp dir
    |
    v
rootCmd.SetArgs(["recover", "--json"])  -- or ["recover", "--apply", "--force"]
    |
    v
rootCmd.Execute()        -- runs full cobra command pipeline
    |
    v
runRecover()             -- recover.go entry point
    |
    +-- loadActiveColonyState()   -- reads COLONY_STATE.json from store
    |
    +-- performStuckStateScan()   -- runs all 7 scanners
    |       |
    |       +-- scanStaleSpawnedWorkers()
    |       +-- scanBadManifest()
    |       +-- scanMissingBuildPacket()
    |       +-- scanPartialPhase()
    |       +-- scanDirtyWorktrees()
    |       +-- scanBrokenSurvey()
    |       +-- scanMissingAgentFiles()
    |
    +-- [if --apply] performRecoverRepairs()
    |       |
    |       +-- createBackup()
    |       +-- dispatchRecoverRepair() x N
    |       +-- [if failed] restoreFromBackup()
    |
    +-- [if --apply] performStuckStateScan()  -- re-scan
    |
    +-- renderRecoverJSON() or renderRecoverDiagnosis()
    |
    v
stdout buffer            -- captured via package-level `stdout` var
    |
    v
Assertions               -- verify output, exit code, state files
```

### Recommended Project Structure
```
cmd/
└── e2e_recovery_test.go     # 10 E2E test functions + seed helpers
```

### Pattern 1: E2E Test via Cobra Command Execution

**What:** Use `saveGlobals`, `resetRootCmd`, `initRecoverTestStore`, then `rootCmd.SetArgs` + `rootCmd.Execute()` to exercise the full command path. Capture output via `bytes.Buffer` assigned to the package-level `stdout` variable. Assert on error return (for exit code), stdout content, and post-command file state.

**When to use:** All E2E tests for `aether recover`.

**Example:**
```go
// Source: [cmd/e2e_regression_test.go pattern + cmd/recover_test.go helpers]
func TestE2ERecoveryMissingBuildPacket(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    _, dataDir := initRecoverTestStore(t)

    seedMissingPacketState(t, dataDir)

    var buf bytes.Buffer
    stdout = &buf
    rootCmd.SetArgs([]string{"recover", "--json"})

    err := rootCmd.Execute()

    // Exit code 1 = issues detected (scan-only finds the problem)
    if err == nil {
        t.Fatal("expected error (exit code 1) for missing build packet")
    }
    // ... parse JSON output, verify issues array has missing_build_packet
}
```

### Pattern 2: Seed Helper Functions

**What:** Functions that write specific broken state files to a temp directory's `.aether/data/` to trigger individual scanners. Each helper knows the exact file structure and field values that make the scanner fire.

**When to use:** Setting up each of the 7 individual stuck-state tests and compound tests.

**Example:**
```go
// Source: [cmd/recover_test.go patterns: newRecoverTestState, recoverWriteJSON]
func seedMissingPacketState(t *testing.T, dataDir string) {
    t.Helper()
    state := newRecoverTestState(t) // State=EXECUTING, Phase=1, no manifest
    recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
    // Deliberately do NOT create build/phase-1/manifest.json
}
```

### Pattern 3: Repair Verification via Re-scan

**What:** After running `aether recover --apply --force`, verify the state is clean by running `aether recover --json` again and asserting the issues array is empty.

**When to use:** All --apply tests (compound scenarios per D-08).

**Example:**
```go
// Run repair
rootCmd.SetArgs([]string{"recover", "--apply", "--force"})
err := rootCmd.Execute()
// ... assert repair succeeded

// Re-scan to verify clean state
buf.Reset()
rootCmd.SetArgs([]string{"recover", "--json"})
err = rootCmd.Execute()
// err == nil means exit code 0, no issues
if err != nil {
    t.Fatalf("expected clean state after repair, got: %v\noutput: %s", err, buf.String())
}
// Parse JSON and verify issues array is empty
```

### Anti-Patterns to Avoid
- **Testing via direct function calls:** Calling `scanMissingBuildPacket()` directly is a unit test, not E2E. All tests MUST go through `rootCmd.Execute()`.
- **Reusing unit test fixtures without modification:** Unit tests in `recover_test.go` call functions directly. E2E tests need the full cobra pipeline, which requires `store` to be initialized and `AETHER_ROOT` to be set.
- **Ignoring the `store` initialization:** `loadActiveColonyState()` returns an error when `store` is nil. E2E tests must call `initRecoverTestStore(t)` before running the command.
- **Forgetting to save/restore globals:** Package-level vars (`store`, `stdout`, `stderr`) leak between tests. Always call `saveGlobals(t)` first.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Temp directory management | Manual `os.MkdirAll` chains | `t.TempDir()` | Auto-cleanup on test exit |
| Store initialization | Manual storage.NewStore + env var | `initRecoverTestStore(t)` | Sets store, AETHER_ROOT, creates .aether/data/ |
| Global var save/restore | Manual defer chains | `saveGlobals(t)` | Saves and restores all 12+ package-level vars |
| Root command reset | Manual flag cleanup | `resetRootCmd(t)` | Resets cobra flags to prevent leakage |
| JSON assertions | String contains checks | `json.Unmarshal` into typed struct | Catches structural issues, not just substring matches |
| Colony state creation | Manual struct literal | `newRecoverTestState(t, overrides...)` | Provides sensible defaults with functional overrides |

**Key insight:** The existing test infrastructure in `cmd/` is extensive and well-designed. E2E recovery tests should compose existing helpers rather than reinventing setup patterns. The only genuinely new code is the seed helper functions and the E2E orchestration logic.

## Common Pitfalls

### Pitfall 1: Missing Agent Files Scanner Checks Against Real Constants
**What goes wrong:** `scanMissingAgentFiles` checks against `expectedClaudeAgents = 25`, `expectedOpenCodeAgents = 25`, `expectedCodexAgents = 25`. If the healthy colony test doesn't create 25 agent files in each directory, it will false-positive.
**Why it happens:** The scanner uses `filepath.Glob` against the AETHER_ROOT directory, which in tests is the temp dir. If no agent dirs exist, glob returns empty and triggers the issue.
**How to avoid:** The healthy colony test must either (a) create 25 agent files in each of `.claude/agents/ant/`, `.opencode/agents/`, `.codex/agents/` under the temp root, or (b) test via `--json` and check the issues array, accepting that a healthy colony in a temp dir will report missing agents (since temp dir won't have 25 files). The better approach: create the minimum expected agent files in the temp dir.
**Warning signs:** Healthy colony test fails with "missing_agents" in the issues array.

### Pitfall 2: Stale Spawn Threshold Requires Old Timestamps
**What goes wrong:** `scanStaleSpawnedWorkers` uses `time.Since(started) > time.Hour` to detect stale workers. If the test writes a timestamp that's only minutes old, the scanner won't trigger.
**Why it happens:** The threshold is hardcoded to 1 hour. Test timestamps must be at least 1 hour in the past.
**How to avoid:** Use `time.Now().Add(-2 * time.Hour).Format(time.RFC3339)` for spawn run timestamps in seed data.
**Warning signs:** Stale spawn test finds 0 issues when expecting 1.

### Pitfall 3: `loadCodexContinueManifest` Reads From `store` Not Filesystem
**What goes wrong:** `scanMissingBuildPacket` calls `loadCodexContinueManifest`, which uses `store.LoadJSON(rel, &manifest)`. The store reads from its base path (`dataDir`), not from `dataDir` directly. But the scan functions also read directly from `dataDir` via `os.ReadFile`.
**Why it happens:** The manifest is stored at `dataDir/build/phase-N/manifest.json` via the store. The seed helper must write the manifest to the correct path relative to `dataDir`.
**How to avoid:** For tests where the manifest should exist, write it to both `dataDir/build/phase-N/manifest.json` (for direct file reads) and use `s.SaveJSON()` (for store.LoadJSON calls). The existing unit tests in `recover_test.go` show this dual-write pattern.
**Warning signs:** Scanner reports "missing build packet" when a manifest file clearly exists.

### Pitfall 4: Destructive Repairs in Non-Interactive Mode
**What goes wrong:** When running `--apply` without `--force`, destructive repairs (`dirty_worktree`, `bad_manifest`) prompt for confirmation on stdin. In test environment, stdin is closed, so confirmation fails and the repair is skipped.
**Why it happens:** `confirmRepair` reads from `os.Stdin`. Tests don't provide input unless using `withMockStdin`.
**How to avoid:** E2E repair tests should always use `--force` to bypass confirmation, or use `withMockStdin` to provide "y\n" input. The CONTEXT.md decision D-07 specifies compound tests use destructive states simultaneously -- these require `--force`.
**Warning signs:** Repair test shows "skipped" for destructive categories.

### Pitfall 5: Exit Code Is Error Return, Not os.Exit
**What goes wrong:** Tests try to check `os.Exit` via `exec.Command` subprocess, which is unnecessary and complex.
**Why it happens:** `runRecover` returns `fmt.Errorf("issues detected")` when issues are found (lines 88-91 of recover.go). It does NOT call `os.Exit`. The cobra framework returns this error from `Execute()`.
**How to avoid:** Check the return value of `rootCmd.Execute()`. `err == nil` means exit code 0 (healthy). `err != nil` means exit code 1 (issues found).
**Warning signs:** None -- this is actually simpler than expected.

### Pitfall 6: `repairDirtyWorktree` Still Has Unconditional State Write (CR-02 NOT Fixed)
**What goes wrong:** The Phase 50 code review identified that `repairDirtyWorktree` unconditionally writes COLONY_STATE.json even for sub-types (git stash, orphan branch deletion) that don't modify state. This was NOT fixed -- only CR-01 (rollback tracking) was fixed in commit `366ce4c1`.
**Why it happens:** The fix commit only addressed CR-01. CR-02 remains live.
**How to avoid:** The compound E2E test should include a dirty_worktree state-disk-mismatch issue alongside other safe issues. After repair, verify the state file is correct. If the unconditional write causes issues, the E2E test will catch it. Note: this is a pre-existing bug, not something the E2E tests need to work around -- the tests should expose it if it matters.
**Warning signs:** Compound test with dirty_worktree + another state-modifying repair produces unexpected state.

## Code Examples

### E2E Test Skeleton (Scan-Only)
```go
// Pattern from: cmd/e2e_regression_test.go + cmd/recover_test.go
func TestE2ERecoveryStaleSpawned(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    _, dataDir := initRecoverTestStore(t)

    // Seed: colony state + stale spawn-runs.json
    state := newRecoverTestState(t)
    recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
    oldTime := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
    recoverWriteJSON(t, dataDir, "spawn-runs.json", map[string]interface{}{
        "current_run_id": "run-1",
        "runs": []map[string]interface{}{
            {"id": "run-1", "started_at": oldTime, "status": "active"},
        },
    })

    var buf bytes.Buffer
    stdout = &buf
    rootCmd.SetArgs([]string{"recover", "--json"})

    err := rootCmd.Execute()

    // Exit code 1 (issues detected)
    if err == nil {
        t.Fatal("expected error for stale spawned workers")
    }

    var result map[string]interface{}
    if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }
    if exitCode, _ := result["exit_code"].(float64); exitCode != 1 {
        t.Errorf("expected exit_code=1, got %v", exitCode)
    }
    issues, _ := result["issues"].([]interface{})
    if len(issues) == 0 {
        t.Fatal("expected stale_spawned issue")
    }
}
```

### E2E Test Skeleton (Repair + Re-scan)
```go
func TestE2ERecoveryCompoundSafeStates(t *testing.T) {
    saveGlobals(t)
    resetRootCmd(t)
    _, dataDir := initRecoverTestStore(t)

    // Seed all 5 safe stuck states simultaneously
    seedMissingPacketState(t, dataDir)
    seedStaleSpawnedState(t, dataDir)
    seedPartialPhaseState(t, dataDir)
    seedBrokenSurveyState(t, dataDir)
    // (missing_agents requires creating agent dirs with < 25 files)

    var buf bytes.Buffer
    stdout = &buf
    rootCmd.SetArgs([]string{"recover", "--apply", "--force"})

    err := rootCmd.Execute()
    if err != nil {
        // After repair, re-scan should find 0 issues -> exit code 0
        t.Fatalf("expected successful repair: %v\noutput: %s", err, buf.String())
    }

    // Verify backup was created
    backupsDir := filepath.Join(filepath.Dir(dataDir), "backups")
    entries, err := os.ReadDir(backupsDir)
    if err != nil || len(entries) == 0 {
        t.Fatal("expected backup directory to be created")
    }

    // Re-scan to verify clean state
    buf.Reset()
    rootCmd.SetArgs([]string{"recover", "--json"})
    err = rootCmd.Execute()
    if err != nil {
        t.Fatalf("expected clean state after repair: %v\noutput: %s", err, buf.String())
    }
}
```

### Healthy Colony Seed
```go
func seedHealthyColonyState(t *testing.T, tmpDir string) {
    t.Helper()
    dataDir := filepath.Join(tmpDir, ".aether", "data")
    os.MkdirAll(dataDir, 0755)

    goal := "Healthy colony"
    state := colony.ColonyState{
        Goal:          &goal,
        State:         colony.StateREADY,
        CurrentPhase:  1,
        Plan:          colony.Plan{Phases: []colony.Phase{{ID: 1, Status: "completed"}}},
        PlanGranularity: colony.PlanGranularityStandard,
    }
    recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)

    // Create agent dirs with expected count (25 each) to avoid false positives
    for i := 0; i < 25; i++ {
        recoverWriteFile(t, tmpDir, fmt.Sprintf(".claude/agents/ant/agent%d.md", i), "# Agent")
        recoverWriteFile(t, tmpDir, fmt.Sprintf(".opencode/agents/agent%d.md", i), "# Agent")
        recoverWriteFile(t, tmpDir, fmt.Sprintf(".codex/agents/agent%d.toml", i), "[agent]")
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os.Exit(code)` in commands | Return `fmt.Errorf` from `RunE` | This codebase (established pattern) | E2E tests check `err != nil` instead of subprocess exit codes |
| Subprocess E2E tests | Same-package `rootCmd.Execute()` | This codebase (established pattern) | Faster, cleaner output capture, direct access to internals |

**Deprecated/outdated:**
- None relevant to this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `initRecoverTestStore(t)` is accessible from `e2e_recovery_test.go` (same package) | Standard Stack | Low risk -- verified both files are in `package cmd` |
| A2 | `newRecoverTestState(t)` creates state with `State=EXECUTING` by default | Code Examples | Low risk -- verified in source code line 33 of recover_test.go |
| A3 | The `store` package-level var is the only way `loadActiveColonyState` reads state | Pitfall 3 | Low risk -- verified in state_load.go line 18 |
| A4 | `surveyFiles` constant has 5 entries: blueprint, chambers, disciplines, provisions, pathogens | Pitfall (seed helpers) | Low risk -- verified in survey.go lines 12-18 |
| A5 | CR-02 (dirty worktree unconditional state write) was not fixed and is still live | Pitfall 6 | Verified -- only CR-01 was addressed in commit 366ce4c1 |
| A6 | The `--json` flag produces valid JSON parseable as `map[string]interface{}` | Code Examples | Low risk -- verified in recover_visuals.go and existing tests |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **Should the healthy colony test create 25 agent files per surface?**
   - What we know: `scanMissingAgentFiles` checks against `expectedClaudeAgents=25`, `expectedOpenCodeAgents=25`, `expectedCodexAgents=25`. A temp dir without these files will trigger false positives.
   - What's unclear: Whether the healthy colony test should account for this (create files) or accept it as a known limitation of testing in a temp dir.
   - Recommendation: Create the minimum 25 agent files per surface in the healthy colony seed. This makes the test more realistic and avoids false positive noise.

2. **Should E2E tests for destructive repairs include confirmation prompt tests?**
   - What we know: D-08 says compound tests use `--apply` (with `--force` implied for destructive). D-05 says individual tests are scan-only.
   - What's unclear: Whether confirmation-prompt E2E tests (using `withMockStdin`) are in scope.
   - Recommendation: Out of scope. The unit tests in `recover_test.go` already cover confirmation prompts (`TestRepairDirtyWorktree_DestructiveNeedsConfirmation`, `TestRepairBadManifest_DestructiveNeedsConfirmation`). E2E tests should use `--force` for destructive repairs.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- this is a test-only phase using only Go stdlib and existing codebase packages).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none -- uses `go test ./cmd/` |
| Quick run command | `go test ./cmd/ -run TestE2ERecovery -v -count=1` |
| Full suite command | `go test ./cmd/ -v -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | Recovery from each of 7 stuck states individually | E2E | `go test ./cmd/ -run TestE2ERecovery(MissingBuildPacket\|StaleSpawned\|PartialPhase\|BadManifest\|DirtyWorktree\|BrokenSurvey\|MissingAgents) -v` | No -- Wave 0 |
| TEST-02 | Recovery from compound stuck state | E2E | `go test ./cmd/ -run TestE2ERecoveryCompound -v` | No -- Wave 0 |
| TEST-03 | No false positives on healthy colony | E2E | `go test ./cmd/ -run TestE2ERecoveryHealthyColony -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run TestE2ERecovery -v -count=1`
- **Per wave merge:** `go test ./cmd/ -v -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/e2e_recovery_test.go` -- the file to be created (all tests)
- [ ] No framework install needed (Go stdlib)

## Security Domain

This phase adds test files only (no production code). Security assessment: not applicable.

## Sources

### Primary (HIGH confidence)
- [cmd/recover.go] -- Command entry point, --apply wiring, scan->repair->re-scan flow (read in full)
- [cmd/recover_scanner.go] -- 7 stuck-state detection functions with exact trigger conditions (read in full)
- [cmd/recover_repair.go] -- 7 repair functions, backup, rollback, confirmation orchestrator (read in full)
- [cmd/recover_visuals.go] -- JSON output structure `recoverJSONOutput`, text rendering, exit code logic (read in full)
- [cmd/recover_test.go] -- 61 existing unit tests with helper functions (read in full)
- [cmd/e2e_regression_test.go] -- Established E2E test pattern (read in full)
- [cmd/medic_repair.go] -- `HealthIssue`, `RepairRecord`, `RepairResult` types, `createBackup`, `findLastValidJSON` (read relevant sections)
- [cmd/medic_wrapper.go] -- `expectedClaudeAgents=25`, `expectedOpenCodeAgents=25`, `expectedCodexAgents=25` constants (verified)
- [cmd/testing_main_test.go] -- `saveGlobals`, `resetRootCmd` helpers (read relevant sections)
- [cmd/state_load.go] -- `loadActiveColonyState` reads from `store` package-level var (verified)
- [cmd/survey.go] -- `surveyFiles` constant: blueprint, chambers, disciplines, provisions, pathogens (verified)
- [cmd/codex_continue.go] -- `loadCodexContinueManifest` reads from store, `codexContinueManifest` type (verified)
- [cmd/codex_build.go] -- `codexBuildManifest`, `codexBuildDispatch` struct definitions (verified)
- [pkg/colony/colony.go] -- `ColonyState` struct, state constants (READY, EXECUTING, BUILT) (verified)
- [pkg/colony/state_machine.go] -- `Transition` function (verified)
- [.planning/phases/50-repair-pipeline/50-REVIEW.md] -- Code review findings CR-01 (fixed) and CR-02 (NOT fixed) (verified)
- [.planning/phases/50-repair-pipeline/50-VERIFICATION.md] -- Phase 50 verification passed 5/5 (verified)
- [commit 366ce4c1] -- Only CR-01 was fixed, CR-02 remains (verified via diff)

### Secondary (MEDIUM confidence)
- None needed -- all findings verified from primary sources.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all components are existing codebase packages, verified in source
- Architecture: HIGH -- full data flow traced through recover.go -> recover_scanner.go -> recover_repair.go -> recover_visuals.go
- Pitfalls: HIGH -- all pitfalls derived from reading the actual source code and test patterns

**Research date:** 2026-04-25
**Valid until:** 30 days (stable domain -- recovery implementation is complete, tests build on top)
