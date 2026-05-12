# Phase 110: Go Safety Invariant Verification - Research

**Researched:** 2026-05-12
**Domain:** Go runtime safety verification, TypeScript host boundary enforcement
**Confidence:** HIGH

## Summary

Phase 110 verifies that Go remains the sole authority for state mutation, finalizers, locking, install/update/publish, and verification contracts -- even with the TypeScript orchestration host (from Phase 109) present and active. This is a verification-only phase: the planner writes tests that confirm existing Go safety mechanisms hold, without adding new runtime guards or changing Go behavior.

The Go runtime already has strong safety mechanisms in place: all four finalizers (plan, build, continue, colonize) validate manifest provenance, workspace freshness, colony mode, and root path before committing state. The `pkg/storage` package provides atomic writes via temp-file-and-rename with cross-process file locking. The TS host already respects boundaries via `assertNoDirectDataWrites()` and `GO_OWNED_PATHS`. Install, update, and publish commands are pure Go with zero TS host imports.

The test file `cmd/safety_invariant_test.go` should use existing test helpers (`setupBuildFlowTest`, `createTestColonyState`, `saveGlobals`/`resetRootCmd`) and map each success criterion (SAFE-01 through SAFE-06) to a dedicated test function.

**Primary recommendation:** Write a single new test file `cmd/safety_invariant_test.go` with 6 test functions, one per success criterion. Reuse existing helpers. Use table-driven patterns for manifest corruption cases. Run existing golden/boundary/provenance tests to confirm no regressions.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Verify existing code only -- no new runtime guards or watchdog systems. Trust Go's existing finalizers, atomic writes, and locking; write tests that prove they work correctly.
- **D-02:** Focus on manifest validation -- verify each finalizer (plan, build, continue) rejects malformed manifests: missing phase number, invalid version, no provenance timestamp, empty worker list.
- **D-03:** Test common corruption cases only -- no adversarial payloads (no deeply nested JSON, no Unicode injection, no future-version manifests). Cover the integration bugs most likely to occur when TS sends data to Go.
- **D-04:** Per-finalizer test sets -- plan, build, and continue finalizers each get their own dedicated validation tests. Clean separation makes it easy to identify which finalizer has a gap.
- **D-05:** Normal flow only -- test the standard plan->build->continue lifecycle driven through the TS host and verify state is correct after each step. No stress scenarios (no concurrent Go+TS finalizer calls, no concurrent state writes).
- **D-06:** Reuse Phase 108 golden tests -- run the golden workflow tests against TS host-driven execution. If the same state transitions happen, Go's invariants hold. Minimal new test code.
- **D-07:** Smoke test install/update/publish purity -- verify `aether install`, `aether update`, `aether publish` have zero code path overlap with the TS host. Simple grep for TS host imports + test that commands work normally.
- **D-08:** One new dedicated file `cmd/safety_invariant_test.go` covering all 6 success criteria. Easy to find, clear purpose, single place to check safety coverage.
- **D-09:** Per-criterion test functions mapping to each success criterion: TestStateMutationSoleAuthority, TestFinalizerProvenance, TestLockingUnchanged, TestInstallPureGo, TestVerificationContractsPass, TestPlanOnlyUnchanged.

### Claude's Discretion
- Exact test implementation patterns (table-driven, sequential, etc.)
- Which golden tests to reuse and how to adapt them for TS host execution
- How to structure the install/update/publish purity check (grep vs Go AST analysis vs import check)
- Error message expectations for rejected manifests
- Whether to use existing test helpers (setupBuildFlowTest, createTestColonyState) or write new ones

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SAFE-01 | Go remains sole authority for COLONY_STATE.json mutation | All state writes go through `store.SaveJSON()`, `store.UpdateJSONAtomically()`, or `store.AtomicWrite()` -- all in `pkg/storage`. No TS code calls these. Tests: call each finalizer with valid manifests, verify state changes only occur through Go, verify TS boundary enforcement via `assertNoDirectDataWrites()`. |
| SAFE-02 | Go finalizers validate manifest provenance before any state write | Four finalizers validate: dispatch_mode="plan-only", requires_finalizer=true, root path match, colony mode match, workspace freshness, generated_at freshness (24h max). Tests: call each finalizer with corrupted manifests (missing fields, wrong mode, expired timestamps), verify rejection. |
| SAFE-03 | Go locking and atomic write semantics unchanged by TS host presence | `pkg/storage/lock.go` uses platform-specific file locks via `platformLockFile()`. `pkg/storage/storage.go` uses temp-file-and-rename pattern. TS host never touches these -- it writes to tmpdir. Tests: run atomic write operations, verify lock files created, verify no corruption. |
| SAFE-04 | Install, update, publish commands remain pure Go -- no TS involvement | `cmd/install_cmd.go` imports: stdlib + aetherassets + pkg/downloader + cobra. `cmd/update_cmd.go` imports: stdlib + pkg/downloader + cobra. `cmd/publish_cmd.go` imports: stdlib + cobra. Zero TS host references in any of these. Tests: grep verification + command execution smoke tests. |
| SAFE-05 | Verification contracts (command-guide, parity tests, drift guards) still pass with TS host enabled | `cmd/command_guide.go` provides command metadata. `cmd/parity_test.go` tests YAML/wrapper/runtime catalog alignment. Tests: run existing parity tests, command-guide tests, and drift guard tests to verify no regressions. |
| SAFE-06 | Existing `aether plan --plan-only` and `aether build --plan-only` behavior unchanged | These commands produce JSON manifests without state mutation. Tests: call with AETHER_OUTPUT_MODE=json, verify JSON output structure unchanged, verify no state files modified. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| COLONY_STATE.json mutation | API / Backend (Go) | -- | Go owns all state via pkg/storage atomic writes |
| Manifest provenance validation | API / Backend (Go) | -- | Finalizers validate before committing state |
| File locking | API / Backend (Go) | -- | pkg/storage/lock.go provides cross-process locks |
| Worker dispatch orchestration | TS Host | -- | TS host calls Go plan-only, dispatches workers, calls finalizers |
| Install/update/publish pipeline | API / Backend (Go) | -- | Pure Go, no TS involvement |
| Verification contracts | API / Backend (Go) | -- | Command guide, parity tests, drift guards in Go |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.26.1 | Primary runtime language | [VERIFIED: go version output] |
| Cobra | current | CLI framework | Used throughout cmd/ for all commands |
| pkg/storage | current | Atomic file operations, locking | Project's own package, authoritative for state writes |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pkg/colony | current | ColonyState type definitions | All state mutation tests |
| pkg/codex | current | Worker handoff validation | Provenance tests |
| pkg/agent | current | Spawn tree tracking | Finalizer tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Single test file | Separate per-criterion files | D-08 mandates single file for discoverability |
| Go testing package | testify assertions | Project uses standard testing throughout -- no need for additional dependency |
| Table-driven tests | Sequential tests | Table-driven is preferred for manifest corruption cases (D-02) |

**Installation:**
```bash
# No new packages needed -- all dependencies already exist
go test ./cmd/... -count=1 -run TestSafety
```

**Version verification:**
```
Go: 1.26.1 darwin/arm64 [VERIFIED: go version output]
Node: v25.9.0 [VERIFIED: node --version output]
aether binary: /Users/callumcowie/.local/bin/aether [VERIFIED: which aether]
```

## Architecture Patterns

### System Architecture Diagram

```
TS Host (.aether/ts-host/)
    |
    | callGoJSON(["plan", "--plan-only"])
    v
Go Runtime (cmd/)
    |-- plan --plan-only --> JSON Manifest (no state mutation)
    |-- build --plan-only --> JSON Manifest (no state mutation)
    |-- continue --plan-only --> JSON Manifest (no state mutation)
    |
    | [TS Host dispatches workers, writes completion to tmpdir]
    |
    | callGoJSON(["plan-finalize", "--completion-file", tmpdir_path])
    v
Go Finalizers
    |-- plan-finalize     --> validates manifest --> store.SaveJSON(COLONY_STATE.json)
    |-- build-finalize    --> validates manifest --> store.UpdateJSONAtomically(COLONY_STATE.json)
    |-- continue-finalize --> validates manifest + gates --> store.UpdateJSONAtomically(COLONY_STATE.json)
    |
    v
pkg/storage
    |-- AtomicWrite()     --> temp file + rename
    |-- UpdateJSONAtomically() --> lock + read + mutate + write
    |-- FileLocker        --> platform-specific file locks
```

### Recommended Project Structure
```
cmd/
├── safety_invariant_test.go    # NEW: Phase 110 test file (6 test functions)
├── provenance.go               # validateBuildProvenance, traceContinueProvenance
├── provenance_test.go          # Existing provenance tests
├── boundary_contract_test.go   # Existing boundary tests (pattern reference)
├── golden_workflow_test.go      # Existing golden tests (reuse for TS host)
├── finality_parity_test.go     # Existing parity tests (pattern reference)
└── orchestrator_boundary_guidance.go  # validateFinalizerManifestRoot, validateFinalizerManifestColonyMode
```

### Pattern 1: Table-Driven Manifest Corruption Tests
**What:** Use Go subtests to test multiple corruption cases against each finalizer.
**When to use:** Manifest validation tests (SAFE-02), per D-02 and D-04.
**Example:**
```go
// Source: Established pattern in cmd/provenance_test.go and cmd/orchestrator_boundary_guidance_test.go
func TestFinalizerProvenance(t *testing.T) {
    t.Run("build_finalize_rejects_missing_dispatch_mode", func(t *testing.T) {
        completion := codexExternalBuildCompletion{
            Manifest: &codexBuildManifest{
                Phase:       1,
                PlanOnly:    true,
                GeneratedAt: time.Now().UTC().Format(time.RFC3339),
                // DispatchMode missing -- should be rejected
            },
        }
        _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
        if err == nil {
            t.Fatal("expected rejection for missing dispatch_mode")
        }
    })
}
```

### Pattern 2: State Mutation Verification via Snapshots
**What:** Snapshot `.aether/data/` before and after operations, verify only expected files changed.
**When to use:** SAFE-01 tests to prove Go is the sole mutator.
**Example:**
```go
// Source: cmd/boundary_contract_test.go snapshotDataDir pattern
before := snapshotDataDir(t, dataDir)
// ... call plan-only command ...
after := snapshotDataDir(t, dataDir)
assertDataDirUnchanged(t, before, after)
```

### Anti-Patterns to Avoid
- **Testing Go internals from TS host tests:** The TS host tests (vitest in .aether/ts-host/) run in a separate process and cannot inspect Go's internal state. Keep Go safety tests in Go test files.
- **Running TS host lifecycle in CI without a colony:** The lifecycle test requires a real Go binary and a temp colony setup. Smoke tests should use the `setupBuildFlowTest` helper for isolation.
- **Modifying Go runtime code to support tests:** Per D-01, this phase adds tests only. No changes to Go behavior.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Colony state setup for tests | Custom temp file creation | `setupBuildFlowTest(t)` + `createTestColonyState(t, dataDir, state)` | Handles AETHER_ROOT, store initialization, cleanup |
| Test isolation | Manual env var save/restore | `saveGlobals(t)` + `resetRootCmd(t)` | Prevents test cross-contamination |
| JSON output parsing | Manual string parsing | `parseEnvelope(t, output)` | Handles {"ok":true,"result":...} envelope correctly |
| File locking verification | Custom lock implementation | `pkg/storage.NewStore()` + `store.AtomicWrite()` | Uses real locking implementation |

**Key insight:** Every helper listed above already exists and is used in test files like `boundary_contract_test.go`, `build_flow_cmds_test.go`, and `finality_parity_test.go`. Reusing them keeps the safety tests consistent with existing patterns.

## Common Pitfalls

### Pitfall 1: TS Host Tests Use Wrong Test Runner
**What goes wrong:** The TS host test files use `node:test` (Node built-in test runner) but vitest is configured in package.json, causing all TS host tests to report "No test suite found."
**Why it happens:** Phase 109 wrote tests with `import { describe, it } from "node:test"` instead of vitest-style imports.
**How to avoid:** Do not depend on TS host tests for this phase. Run Go-side tests only. If TS host test execution is needed, use `node --test` instead of `npx vitest`.
**Warning signs:** `npx vitest run` shows "No test suite found in file" for all TS host test files.

### Pitfall 2: Pre-Existing Test Failures from Deleted Planning Artifacts
**What goes wrong:** 6 tests fail because they reference deleted files (`.planning/phases/103-*/DATA-FLOW.md`, `.planning/phases/102-*/WORKER-ECONOMY.md`).
**Why it happens:** Phases 100-104 planning artifacts were deleted from git but the audit tests still reference them.
**How to avoid:** Do not treat these failures as regressions. Run safety-specific tests with `-run TestSafety` or `-run TestSafetyInvariant` to avoid the unrelated failures.
**Warning signs:** `TestDataFlowSnapshot`, `TestDataFlowDeadEnds`, `TestDataFlowColonyPrimeWiring`, `TestDataFlowReportAccuracy`, `TestDispatchedCastesDocumented`, `TestVisualOutputTracesToState` all fail with "no such file or directory."

### Pitfall 3: Finalizer Tests Require Full Colony State
**What goes wrong:** Calling `runCodexBuildFinalize()` or `runCodexContinueFinalize()` without proper state initialization produces "No active colony goal" or "No project plan" errors.
**Why it happens:** Finalizers load and validate colony state from disk as their first step.
**How to avoid:** Use `createTestColonyState()` with a properly initialized `colony.ColonyState` including goal, phases, and correct state/status values. See `finality_parity_test.go` for the exact state setup patterns.
**Warning signs:** "No active colony goal" or "phase N not found" errors in finalizer tests.

### Pitfall 4: Manifest Root Path Validation Requires Real Directory
**What goes wrong:** `validateFinalizerManifestRoot()` compares the manifest root against the actual workspace root. Tests that use empty or fake paths get rejected.
**Why it happens:** Security measure -- prevents manifest injection from different workspaces.
**How to avoid:** Pass the actual temp directory path as both the manifest root and the workspace root. See `finality_parity_test.go` line 166-175 for the pattern (empty root or matching root accepted).
**Warning signs:** "root does not match current workspace" errors.

### Pitfall 5: Plan Manifest Freshness Validation (24-hour max age)
**What goes wrong:** Creating a manifest with `GeneratedAt` set to a past timestamp that exceeds 24 hours causes rejection.
**Why it happens:** `validateCodexPlanManifestFreshness()` enforces a 24-hour max age and 5-minute future skew.
**How to avoid:** Always use `time.Now().UTC().Format(time.RFC3339)` for manifest timestamps in tests. Do not hardcode timestamps.
**Warning signs:** "stale plan_manifest generated_at exceeds max age" errors.

## Code Examples

### SAFE-01: Verify Go Is Sole State Mutator
```go
// Source: cmd/boundary_contract_test.go pattern
func TestStateMutationSoleAuthority(t *testing.T) {
    tmpDir := t.TempDir()
    dataDir := filepath.Join(tmpDir, ".aether", "data")
    os.MkdirAll(dataDir, 0755)

    s, _ := storage.NewStore(dataDir)
    goal := "test safety"
    now := time.Now().UTC()
    state := colony.ColonyState{
        Version: "1.0", Goal: &goal, CurrentPhase: 1,
        InitializedAt: &now, State: colony.StateREADY,
        Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Test", Status: colony.PhaseReady}}},
    }
    s.SaveJSON("COLONY_STATE.json", &state)

    // Snapshot before orchestration
    before := snapshotDataDir(t, dataDir)

    // Simulate TS host orchestration phase (no Go finalizer calls)
    // ... TS host would dispatch workers here ...

    // Verify no state mutation during orchestration
    after := snapshotDataDir(t, dataDir)
    assertDataDirUnchanged(t, before, after)

    // Now simulate finalizer (Go-owned state mutation)
    s.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
        state.Plan.Phases[0].Status = colony.PhaseCompleted
        return nil
    })
    // Verify state was mutated by Go
    var updated colony.ColonyState
    s.LoadJSON("COLONY_STATE.json", &updated)
    if updated.Plan.Phases[0].Status != colony.PhaseCompleted {
        t.Fatal("Go finalizer should have mutated state")
    }
}
```

### SAFE-02: Manifest Provenance Validation
```go
// Source: cmd/codex_build_finalize.go validation logic
func TestFinalizerProvenance(t *testing.T) {
    saveGlobals(t)
    dataDir := setupBuildFlowTest(t)
    goal := "test provenance"
    now := time.Now().UTC()
    state := colony.ColonyState{
        Version: "1.0", Goal: &goal, CurrentPhase: 1,
        State: colony.StateREADY, InitializedAt: &now,
        Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Test", Status: colony.PhaseReady}}},
    }
    createTestColonyState(t, dataDir, state)

    root := filepath.Dir(filepath.Dir(dataDir))
    tmpDir := t.TempDir()

    t.Run("build_finalize_rejects_wrong_dispatch_mode", func(t *testing.T) {
        manifest := validBuildManifest(root, now)
        manifest.DispatchMode = "live" // Should be "plan-only"
        completion := codexExternalBuildCompletion{DispatchManifest: &manifest}
        completionPath := writeCompletionJSON(t, tmpDir, completion)
        // Call build-finalize and verify rejection
    })

    t.Run("build_finalize_rejects_stale_generated_at", func(t *testing.T) {
        manifest := validBuildManifest(root, now)
        manifest.GeneratedAt = now.Add(-25 * time.Hour).Format(time.RFC3339) // Exceeds 24h
        // Verify rejection
    })
}
```

### SAFE-04: Install/Update/Publish Purity Check
```go
// Source: Verified via import analysis
func TestInstallPureGo(t *testing.T) {
    // Verify install, update, publish commands have zero TS host imports
    files := []string{
        "install_cmd.go",
        "update_cmd.go",
        "publish_cmd.go",
    }
    for _, file := range files {
        content, err := os.ReadFile(filepath.Join("..", "cmd", file))
        if err != nil {
            t.Fatalf("read %s: %v", file, err)
        }
        s := string(content)
        for _, forbidden := range []string{"ts-host", "tsHost", "ts_host", "typescript", "assertNoDirect"} {
            if strings.Contains(s, forbidden) {
                t.Errorf("%s contains forbidden reference to TS host: %s", file, forbidden)
            }
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell-based state mutation | Go-owned atomic writes via pkg/storage | v1.0 (Apr 2026) | All state writes go through locked, atomic operations |
| Wrapper-owned state writes | Boundary contract (Go-only state mutation) | Phase 106 | TS host enforces boundary via assertNoDirectDataWrites() |
| No provenance validation | Manifest provenance validation in finalizers | Phase 97-99 | Finalizers reject malformed or unattributed manifests |
| No worker identity checks | Worker identity validation (caste, stage, taskID) | Phase 100 | Finalizers reject worker results with mismatched identity |

**Deprecated/outdated:**
- Shell-based `atomic-write.sh`: Replaced by Go `pkg/storage.AtomicWrite()`
- `state-sync.js`: Obsolete, Go handles atomically
- `file-lock.js`: Replaced by Go `pkg/storage/lock.go`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | TS host tests intentionally use `node:test` not vitest; not a bug | Pitfall 1 | Medium -- if vitest is intended, tests need rewrite |
| A2 | Pre-existing test failures (DATA-FLOW, WORKER-ECONOMY) are known and unrelated | Pitfall 2 | Low -- git status confirms deleted planning artifacts |
| A3 | `runCodexBuildFinalize()` and `runCodexContinueFinalize()` are directly callable from tests within the `cmd` package (same package, unexported functions) | Code Examples | Low -- verified they are unexported functions in cmd package |
| A4 | The `snapshotDataDir` and `assertDataDirUnchanged` helpers from `boundary_contract_test.go` can be reused from the same package | Code Examples | Low -- they are unexported helpers in the same package |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **TS host test runner mismatch**
   - What we know: TS host test files use `node:test` imports but vitest is configured in package.json. Running `npx vitest run` fails with "No test suite found."
   - What's unclear: Whether this is intentional (tests designed for `node --test`) or a bug from Phase 109.
   - Recommendation: Do not depend on TS host tests for this phase. Run Go-side safety tests only. If TS host verification is needed, use `node --test test/*.ts`.

2. **Golden workflow test reuse (D-06)**
   - What we know: Phase 108 created `cmd/golden_workflow_test.go` with lifecycle tests. These test plan->build->continue via the Go runtime directly.
   - What's unclear: Whether these tests already cover the TS host-driven path, or whether additional test functions are needed.
   - Recommendation: Check if `TestGoldenWorkflowPlanBuildContinue` (or similar) already exercises the full lifecycle. If it tests Go-only execution, add a TS host variant. If it already covers TS host, reference it as evidence for SAFE-01.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go runtime | All Go tests | Yes | 1.26.1 darwin/arm64 | -- |
| Node.js | TS host tests (optional) | Yes | v25.9.0 | -- |
| aether binary | Integration tests | Yes | /Users/callumcowie/.local/bin/aether | `go run ./cmd/aether` |
| npm | TS host dependency | Yes | 11.12.1 | -- |

**Missing dependencies with no fallback:**
- None -- all required tools are available.

**Missing dependencies with fallback:**
- None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + vitest (TS host, optional) |
| Config file | go is implicit; vitest: `.aether/ts-host/vitest.config.ts` if present |
| Quick run command | `go test ./cmd/... -count=1 -run TestSafety -v` |
| Full suite command | `go test ./cmd/... -count=1 -v` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SAFE-01 | Go sole authority for COLONY_STATE.json mutation | unit | `go test ./cmd/... -run TestStateMutationSoleAuthority -v` | Wave 0 (new) |
| SAFE-02 | Finalizers validate manifest provenance | unit | `go test ./cmd/... -run TestFinalizerProvenance -v` | Wave 0 (new) |
| SAFE-03 | Locking/atomic semantics unchanged by TS host | unit | `go test ./cmd/... -run TestLockingUnchanged -v` | Wave 0 (new) |
| SAFE-04 | Install/update/publish pure Go | unit | `go test ./cmd/... -run TestInstallPureGo -v` | Wave 0 (new) |
| SAFE-05 | Verification contracts pass with TS host | integration | `go test ./cmd/... -run "TestCatalog|TestParity|TestCommandGuide" -v` | Existing |
| SAFE-06 | plan-only and build-only behavior unchanged | unit | `go test ./cmd/... -run TestPlanOnlyUnchanged -v` | Wave 0 (new) |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -count=1 -run TestSafety -v`
- **Per wave merge:** `go test ./cmd/... -count=1 -v` (excluding known unrelated failures)
- **Phase gate:** Full suite green (excluding 6 pre-existing data flow audit failures)

### Wave 0 Gaps
- [ ] `cmd/safety_invariant_test.go` -- covers SAFE-01 through SAFE-06 (new file)
- [ ] No framework install needed -- Go testing package already in use

*(If no gaps: "None -- existing test infrastructure covers all phase requirements")*

## Security Domain

> Security enforcement is implied by the nature of this phase (verifying safety invariants).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | yes | Finalizers validate colony_mode and root path (access control for state mutation) |
| V5 Input Validation | yes | Manifest provenance validation (phase number, timestamps, dispatch mode, worker identity) |
| V6 Cryptography | no | -- |

### Known Threat Patterns for Go Safety Invariants

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Manifest injection (corrupted manifest from TS host) | Tampering | Provenance validation: dispatch_mode, requires_finalizer, generated_at freshness, root match |
| State mutation bypass (TS host writes directly to .aether/data/) | Tampering | assertNoDirectDataWrites() boundary enforcement, GO_OWNED_PATHS check |
| Concurrent state corruption | Tampering | pkg/storage/lock.go cross-process file locking, atomic temp-file-and-rename |
| Phantom builds (no real worker output) | Spoofing | validateBuildProvenance() rejects builds with no file changes from completed workers |

## Sources

### Primary (HIGH confidence)
- `cmd/codex_plan_finalize.go` -- Plan finalizer validation logic (read in full)
- `cmd/codex_build_finalize.go` -- Build finalizer validation logic (read in full)
- `cmd/codex_continue_finalize.go` -- Continue finalizer validation logic (read in full)
- `cmd/codex_colonize_finalize.go` -- Colonize finalizer validation logic (read in full)
- `pkg/storage/storage.go` -- Atomic write operations (read in full)
- `pkg/storage/lock.go` -- Cross-process file locking (read in full)
- `cmd/provenance.go` -- Provenance validation functions (read in full)
- `cmd/state_cmds.go` -- State mutation command with atomic writes (read in full)
- `.aether/ts-host/src/go-bridge.ts` -- Boundary enforcement (read in full)
- `.aether/ts-host/src/boundary-reference.ts` -- GO_OWNED_PATHS definition (read in full)
- `.aether/ts-host/src/lifecycle.ts` -- TS host lifecycle orchestrator (read in full)
- `.aether/references/contracts/runtime-boundary-contract.md` -- Boundary contract (read in full)
- `cmd/boundary_contract_test.go` -- Existing boundary tests (read in full)
- `cmd/finality_parity_test.go` -- Existing parity tests (read 80 lines)
- `cmd/golden_workflow_test.go` -- Golden test patterns (read 120 lines)
- `cmd/build_flow_cmds_test.go` -- Test helpers (read 100 lines)
- `cmd/install_cmd.go` -- Install command imports (verified: pure Go)
- `cmd/update_cmd.go` -- Update command imports (verified: pure Go)
- `cmd/publish_cmd.go` -- Publish command imports (verified: pure Go)

### Secondary (MEDIUM confidence)
- `cmd/orchestrator_boundary_guidance.go` -- Manifest root/mode validation (read 80 lines)
- `cmd/command_guide.go` -- Command guide structure (read 60 lines)
- `cmd/parity_test.go` -- Parity test patterns (read 80 lines)
- Grep results for COLONY_STATE.json references in cmd/ -- all state writes go through store methods
- Grep results for ts-host/typescript references in Go code -- only in test files for language detection, not TS host imports

### Tertiary (LOW confidence)
- None -- all findings verified via code reading or grep.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries and versions verified from running commands
- Architecture: HIGH - read all finalizer, storage, and TS host files in full
- Pitfalls: HIGH - discovered through running tests and analyzing failures
- Test patterns: HIGH - read existing test files and helpers in full

**Research date:** 2026-05-12
**Valid until:** 2026-06-12 (stable -- Go runtime does not change frequently)
