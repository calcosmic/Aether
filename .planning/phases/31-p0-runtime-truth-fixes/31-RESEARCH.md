# Phase 31: P0 Runtime Truth Fixes - Research

**Researched:** 2026-04-22
**Domain:** Go runtime operational safety — worker dispatch, error propagation, verification gating, state atomicity
**Confidence:** HIGH

## Summary

This phase fixes 7 P0 root causes that make Aether's runtime dishonest. The issues span the worker invocation layer (`pkg/codex/`), the build/continue orchestration layer (`cmd/codex_build.go`, `cmd/codex_continue.go`), and the state persistence layer (`pkg/storage/`). All 7 issues are confirmed by direct code inspection and the Oracle synthesis document.

The core pattern across these bugs is the same: the runtime treats absence of failure as proof of success. FakeInvoker produces synthetic completions without work. Pool.dispatch() discards errors. Continue finds reasons to advance despite failed verification. Reconcile skips verification entirely. Claims are trusted without git evidence. Test failures are dismissed as environmental. And state updates are partial, leaving the colony inconsistent on interruption.

**Primary recommendation:** Fix each issue at the layer where the lie originates — worker layer for phantom work, dispatch layer for silent errors, continue layer for bypasses, storage layer for atomicity — then add integration tests that prove the lie is no longer possible.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Worker invocation truth | `pkg/codex/` (Go runtime) | — | WorkerInvoker interface owns whether work is real or synthetic |
| Error propagation | `pkg/codex/dispatch.go` | `cmd/dispatch_runtime.go` | DispatchBatch contract decides if errors reach callers |
| Verification gating | `cmd/codex_continue.go` | `cmd/codex_build.go` | Continue orchestration decides when phases advance |
| Claim verification | `cmd/codex_continue.go` | `cmd/codex_build_worktree.go` | Continue checks claims; build_worktree collects git evidence |
| State atomicity | `pkg/storage/storage.go` | `cmd/codex_continue.go` | Storage owns atomic writes; continue owns transaction scope |
| Test failure reporting | `cmd/codex_continue.go` | — | runVerificationStep executes and interprets test results |
| Midden logging | `cmd/midden_cmds.go` | `cmd/codex_continue.go` | Midden-write command exists; continue must call it on failure |

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **FakeInvoker is test-only.** It never runs in production paths. If a worker cannot execute for real, it fails honestly.
- **Fail fast + midden logging.** When a worker fails, the phase stops immediately and the failure is recorded in `.aether/data/midden/midden.json`.
- **Always re-verify.** After any recovery action, run verification from scratch.
- **All 4 known bypass paths in continue are closed.** Continue only advances when verification proves the phase succeeded.
- **`--reconcile-task` must re-run verification** before marking tasks complete, or explicitly warn the user.
- **Claims about file modifications must be git-verified.** Worker claims alone are not sufficient evidence.
- **Test failures are surfaced honestly** with clear error messages, never dismissed as environmental without evidence.
- **Phase advancement is all-or-nothing** with rollback on failure.

### Claude's Discretion
- Exact implementation of atomic state updates (transaction wrapper, temp file + rename, or other).
- Specific error propagation mechanism for Pool.dispatch() (return error, observer callback, or result struct).
- How to detect and block FakeInvoker in production paths (runtime check, build tag, or invoker registry).

### Deferred Ideas (OUT OF SCOPE)
- Performance optimization of dispatch
- New visual indicators for failure states (wrapper polish, Phase 35)
- Auto-retry logic
- New colony states beyond what Phase 15-16 established

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| R045 | Eliminate FakeInvoker from production paths | `NewWorkerInvoker()` falls back to FakeInvoker when codex CLI unavailable; `runCodexBuild` accepts `synthetic` flag that forces FakeInvoker; colonize/plan fallback to FakeInvoker on dispatch error |
| R046 | Pool.dispatch() error propagation | `DispatchBatch` documents returned error as "always nil"; `dispatchBatchByWaveWithVisuals` propagates the nil-error contract; callers in build/continue do not check per-result errors systematically |
| R047 | Continue bypass bug fixes | 4 bypass paths identified: `verified_partial` allows advancement with failed workers (line 1129), watcher timeout treated as environmental (line 696), empty dispatch status defaults to "completed" (line 2062), `continueTasksSupportAdvancement` returns true for empty task lists when claims satisfied |
| R048 | Reconcile-task verification restoration | `assessCodexContinue` sets `taskArtifactEvidenceTrusted = true` when `reconciledTask` is true (line 1005), bypassing claims verification; reconciled tasks skip `implemented_unverified` classification |
| R049 | Git-verified in-repo claims | `dispatchCodexBuildWorkersInRepo` only calls `applyObservedClaims` for non-completed workers (line 369); completed workers in in-repo mode bypass git verification entirely |
| R050 | Honest test failure reporting | `runCodexContinueVerification` treats watcher timeout as environmental and leaves `checksPassed = true` (line 696); no `isEnvironmentalConstraintText` matcher found in current source, but the timeout bypass remains |
| R051 | Atomic phase advancement | `runCodexContinue` saves `COLONY_STATE.json` at line 451 after multiple prior state mutations (lines 378-404); if an error occurs after line 378 but before 451, the colony is left with partially advanced state |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.24 | `context`, `os`, `os/exec`, `path/filepath`, `sync` | Native runtime, no dependencies |
| `pkg/storage` | in-tree | Atomic file writes (`AtomicWrite`, `UpdateFile`) | Already implements temp-file + rename pattern |
| `pkg/codex` | in-tree | Worker dispatch, invocation, claims extraction | Core abstraction layer for all worker execution |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/colony` | in-tree | State types (`ColonyState`, `Phase`, `Task`) | State mutations throughout |
| `pkg/agent` | in-tree | Spawn tree recording | Worker lifecycle tracking |

## Architecture Patterns

### System Architecture Diagram

```
User Command (build/continue/colonize/plan)
    |
    v
[cmd/codex_*.go] -- Orchestration layer
    |-- Validates state
    |-- Plans dispatches
    |-- Calls dispatch layer
    |
    v
[cmd/dispatch_runtime.go] -- dispatchBatchByWaveWithVisuals
    |-- Groups by wave
    |-- Emits visual progress
    |-- Delegates to pkg/codex
    |
    v
[pkg/codex/dispatch.go] -- DispatchBatch / DispatchWaveWithObserver
    |-- Executes workers wave-by-wave
    |-- Returns []DispatchResult + error (currently always nil)
    |
    v
[pkg/codex/worker.go] -- WorkerInvoker interface
    |-- RealInvoker: spawns codex CLI subprocess
    |-- FakeInvoker: returns deterministic "completed" (TEST ONLY)
    |
    v
[cmd/codex_continue.go] -- Verification & gating
    |-- runCodexContinueVerification: build/test/lint/type checks
    |-- runCodexContinueGates: manifest, evidence, flags
    |-- runCodexContinueReview: gatekeeper/auditor/probe
    |-- Phase advancement (or rollback)
    |
    v
[pkg/storage/storage.go] -- Atomic state persistence
    |-- AtomicWrite: temp file + rename
    |-- UpdateFile: read-modify-write under lock
```

### Recommended Project Structure

No new directories needed. Changes are confined to:
- `pkg/codex/worker.go` — FakeInvoker detection / removal from production
- `pkg/codex/dispatch.go` — Error propagation contract
- `cmd/codex_continue.go` — Bypass fixes, reconcile verification, atomic advancement
- `cmd/codex_build.go` — Synthetic flag handling
- `cmd/codex_build_worktree.go` — Git verification for in-repo mode
- `cmd/dispatch_runtime.go` — Error propagation from dispatch
- `pkg/storage/storage.go` — Transaction helper (if needed)

### Pattern 1: Fail-Fast with Midden Logging
**What:** When a worker fails, stop the phase immediately and record the failure.
**When to use:** Any production dispatch path where worker failure indicates the phase cannot complete.
**Example:**
```go
// Source: cmd/codex_continue.go (proposed pattern)
if !gates.Passed {
    // ... build blocked result ...
    // Log to midden for colony learning
    _ = store.SaveJSON("midden.json", appendMiddenEntry(mf, colony.MiddenEntry{
        Category: "continue_blocked",
        Message:  summary,
        Source:   "continue",
    }))
    return result, blockedState, phase, nil, nil, false, nil
}
```

### Pattern 2: Atomic State Update
**What:** All state mutations for a single logical operation are written together.
**When to use:** Phase advancement, state transitions that must be all-or-nothing.
**Example:**
```go
// Source: pkg/storage/storage.go (existing AtomicWrite)
func (s *Store) AtomicWrite(path string, data []byte) error {
    // ... lock, write temp, validate JSON, rename ...
}
```

### Anti-Patterns to Avoid
- **Silent error swallowing:** `_ = g.Wait()` or ignoring `DispatchResult.Error` — always surface or log.
- **Synthetic success as default:** FakeInvoker in production paths makes all success metrics meaningless.
- **Partial state updates:** Mutating `state.Plan.Phases[i].Status` before all validation gates have passed.
- **Environmental excuse:** Treating timeouts or CLI unavailability as "not real failures" — they are real failures.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic file writes | Custom temp+rename in each command | `pkg/storage.Store.AtomicWrite` | Already implemented, tested, with JSON validation |
| Worker error aggregation | Custom error collection in each orchestrator | Change `DispatchBatch` to return error when any worker fails | Single contract, consistent behavior |
| Fake worker detection | Inline type assertions everywhere | Central `IsProductionInvoker()` helper or invoker registry | One place to enforce the test/production boundary |
| State rollback | Manual clone + restore in each command | `Store.UpdateFile` with mutator function | Already atomic; mutator can return error to abort |

## Runtime State Inventory

This phase is a bug-fix/refactor phase. No stored data keys or runtime registrations change names. However, the following runtime behaviors change:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `COLONY_STATE.json` phase/task statuses | Code edit — advancement logic changes, state format unchanged |
| Stored data | `midden.json` failure records | Code edit — continue must write failures when blocked |
| Stored data | `last-build-claims.json` | Code edit — in-repo builds must populate with git-verified claims |
| Live service config | None | — |
| OS-registered state | None | — |
| Secrets/env vars | `AETHER_CODEX_REAL_DISPATCH` controls FakeInvoker vs RealInvoker | Code edit — may need to change default or add validation |
| Build artifacts | None | — |

## Common Pitfalls

### Pitfall 1: FakeInvoker Still Used in Tests After Production Ban
**What goes wrong:** Tests that previously relied on FakeInvoker for production-path coverage break when FakeInvoker is rejected.
**Why it happens:** Many tests set `newCodexWorkerInvoker = func() { return &codex.FakeInvoker{} }` to avoid needing the codex CLI.
**How to avoid:** Update tests to use a test-double that implements `WorkerInvoker` but is explicitly marked as test-only, or mock `newCodexWorkerInvoker` to return a `RealInvoker` with a fake binary path.
**Warning signs:** Test failures with "synthetic worker detected in production path" after the fix.

### Pitfall 2: Dispatch Error Propagation Breaks Existing Callers
**What goes wrong:** Changing `DispatchBatch` to return a non-nil error when workers fail will break callers that currently assume error is always nil.
**Why it happens:** `dispatchBatchByWaveWithVisuals`, `runCodexContinueReview`, `runCodexContinueWatcherVerification` all check `if err != nil` but the error is currently always nil.
**How to avoid:** Audit all callers. Decide whether to propagate the error or require callers to iterate results. Update tests to expect the new behavior.
**Warning signs:** Compilation errors in tests, or unexpected error returns in production.

### Pitfall 3: Continue Bypass Fixes Break Partial-Success Semantics
**What goes wrong:** The `verified_partial` outcome is intentionally used for phases where some workers timed out but verification passed. Removing it entirely may block legitimate partial-success cases.
**Why it happens:** `TestContinueAdvancesOnVerifiedPartialSuccess` (line 854) explicitly tests this case.
**How to avoid:** Distinguish between "worker failed but verification passed" (legitimate partial success) and "worker failed and no verification evidence exists" (bypass). The fix should target the latter.
**Warning signs:** `TestContinueAdvancesOnVerifiedPartialSuccess` fails after the fix.

### Pitfall 4: Reconcile Verification Becomes Too Strict
**What goes wrong:** Requiring full verification for reconciled tasks may make recovery impossible when the build environment has changed.
**Why it happens:** Reconcile is the escape hatch for stuck colonies. If the escape hatch requires the same verification that is failing, the colony stays stuck.
**How to avoid:** The requirement is "re-run verification before marking tasks complete, or explicitly warn the user that verification was skipped." A warning path is acceptable.
**Warning signs:** Users cannot recover from stuck states after the fix.

### Pitfall 5: Git Verification Adds Significant Overhead
**What goes wrong:** Running `git status` for every completed worker in in-repo mode may slow down builds.
**Why it happens:** `snapshotGitStatus` shells out to `git status --porcelain`.
**How to avoid:** Snapshot once per wave, not once per worker. The baseline can be captured before the wave starts and compared after.
**Warning signs:** Build tests timeout after the fix.

### Pitfall 6: Atomic Advancement Creates Deadlocks
**What goes wrong:** If all post-advance operations (housekeeping, context update, spawn tree) must succeed before state is saved, a failure in any one prevents advancement permanently.
**Why it happens:** `continueContextUpdater` writes to `.aether/CONTEXT.md` which could fail due to permissions.
**How to avoid:** Distinguish between "must succeed" (state mutation) and "best effort" (context update, spawn tree). Roll back state only if must-succeed operations fail.
**Warning signs:** Continue never advances because CONTEXT.md is read-only.

## Code Examples

### Verified Pattern: FakeInvoker Detection
```go
// Source: cmd/codex_continue.go:723 (existing)
if _, ok := invoker.(*codex.FakeInvoker); ok {
    summary := "continue watcher verification blocked because codex CLI is not available (FakeInvoker in use)"
    return codexWatcherVerification{...}, &codexContinueWorkerFlowStep{...}
}
```
This pattern exists for the watcher. It must be extended to ALL production dispatch paths (build, plan, colonize, continue review).

### Verified Pattern: Git Claim Verification
```go
// Source: cmd/codex_build_worktree.go:249-255 (worktree mode — already correct)
touched, touchErr := collectWorktreeTouchedPaths(session.AbsPath, baseline, result)
if touchErr != nil {
    dr.Status = "failed"
    dr.Error = touchErr
    finalStatus = colony.WorktreeOrphaned
} else {
    applyObservedClaims(session.AbsPath, baseline, touched, dr.WorkerResult)
}
```
The same pattern must be applied to in-repo mode in `dispatchCodexBuildWorkersInRepo`.

### Verified Pattern: Midden Write
```go
// Source: cmd/midden_cmds.go:401-474 (existing command)
var middenWriteCmd = &cobra.Command{...}
```
This command exists but is never called from the continue/build flow. The fix should add a helper `logFailureToMidden(category, message, source)` that can be called from any orchestrator.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `DispatchBatch` error always nil | Same (bug) | Never fixed | Silent failure propagation |
| FakeInvoker as default fallback | Same (bug) | Never fixed | Phantom workers in production |
| `verified_partial` allows advancement | Same (bug) | v1.2 Phase 15 | Partial success semantics too permissive |
| Reconcile skips verification | Same (bug) | v1.2 Phase 16 | Recovery tool becomes bypass tool |
| State saved mid-advancement | Same (bug) | v1.2 Phase 15 | Inconsistent colony state possible |

**Deprecated/outdated:**
- `planningDispatchTimeout` variable: Referenced in stale worktree copies but NOT in current source. Tests use `planningScoutTimeout` and `planningRouteSetterTimeout` (defined in `cmd/codex_dispatch_contract.go`). No action needed for Phase 31.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 4 continue bypass paths identified in Oracle are the only significant bypass paths | Continue Bypass Fixes | If additional bypasses exist, they will be discovered during testing |
| A2 | `pkg/storage.Store.AtomicWrite` is sufficient for atomic phase advancement | Atomic Phase Advancement | If multiple files need atomic update together, a broader transaction mechanism may be needed |
| A3 | Tests can be updated to use test-doubles instead of FakeInvoker for production-path coverage | FakeInvoker Elimination | If codex CLI is required for tests, CI must provide it or tests must be restructured |
| A4 | The `isEnvironmentalConstraintText` matcher mentioned in Oracle P0-6 does not exist in current source | Honest Test Reporting | The actual bypass is the watcher timeout environmental excuse at line 696; if additional matchers exist in other files, they need separate fixes |

## Open Questions

1. **How should FakeInvoker be blocked in production?**
   - What we know: `NewWorkerInvoker()` falls back to FakeInvoker when `AETHER_CODEX_REAL_DISPATCH` is unset and codex CLI is unavailable.
   - What's unclear: Whether to change the default, add a runtime check in each command, or both.
   - Recommendation: Add an explicit check in `runCodexBuild`, `runCodexPlan`, `runCodexColonize`, and `runCodexContinue` that rejects FakeInvoker before dispatch begins. Keep FakeInvoker available for tests via the env var.

2. **Should DispatchBatch return an error when any worker fails, or should callers check results?**
   - What we know: Currently the error is always nil. Callers like `dispatchBatchByWaveWithVisuals` return the error to their callers.
   - What's unclear: Whether changing the contract breaks more callers than it fixes.
   - Recommendation: Add a new function `DispatchBatchStrict` that returns an error when any worker fails, and migrate production callers to it. Keep `DispatchBatch` for backward compatibility during the transition.

3. **What is the exact scope of "atomic phase advancement"?**
   - What we know: State is saved once at the end of `runCodexContinue`, but multiple mutations happen before that.
   - What's unclear: Whether atomicity means "all state mutations in one write" or "rollback on failure."
   - Recommendation: Use `cloneColonyState` to create a working copy, mutate it, and only save if all operations succeed. If any operation fails, return the original unmutated state.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build, test, vet | Yes | 1.24 | — |
| git | Worktree mode, claim verification | Yes | — | In-repo mode |
| codex CLI | Real worker dispatch | Unknown | — | FakeInvoker (to be removed) |

**Missing dependencies with no fallback:**
- codex CLI: If unavailable, production paths will fail honestly after FakeInvoker is removed. This is the intended behavior.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (standard) |
| Config file | None — standard `go test` |
| Quick run command | `go test ./pkg/codex/... -v` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| R045 | FakeInvoker rejected in production build path | unit | `go test ./cmd -run TestBuild.*FakeInvoker -v` | Yes (needs update) |
| R045 | FakeInvoker rejected in continue review | unit | `go test ./cmd -run TestContinueBlocksWhenWatcherUsesFakeInvoker -v` | Yes |
| R046 | Dispatch errors propagate to caller | unit | `go test ./pkg/codex -run TestDispatchBatch -v` | Yes (needs update) |
| R047 | `verified_partial` does not bypass gates | unit | `go test ./cmd -run TestContinueAdvancesOnVerifiedPartialSuccess -v` | Yes (needs update) |
| R047 | Empty status defaults to failed, not completed | unit | New test needed | No |
| R048 | Reconcile requires verification or warns | unit | `go test ./cmd -run TestContinueReconcileTaskDoesNotTrustOtherTasks -v` | Yes (needs update) |
| R049 | In-repo claims are git-verified | unit | `go test ./cmd -run TestBuild.*Claims -v` | Yes (needs update) |
| R050 | Watcher timeout blocks advancement | unit | `go test ./cmd -run TestContinue.*Watcher.*Timeout -v` | New test needed |
| R051 | Partial state update rolled back on failure | unit | New test needed | No |

### Wave 0 Gaps
- [ ] `TestDispatchBatch_ReturnsErrorOnWorkerFailure` — covers R046
- [ ] `TestContinueEmptyStatusDefaultsToFailed` — covers R047 bypass #3
- [ ] `TestContinueWatcherTimeoutBlocksAdvancement` — covers R050
- [ ] `TestContinueAtomicRollbackOnFailure` — covers R051
- [ ] `TestBuildInRepoClaimsAreGitVerified` — covers R049

## Security Domain

This phase does not introduce new security surface. It hardens existing operational safety:

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | Yes | Claims verification ensures worker-reported file changes match actual filesystem state |
| V6 Cryptography | No | Not in scope |

## Sources

### Primary (HIGH confidence)
- `pkg/codex/worker.go` — FakeInvoker definition, NewWorkerInvoker fallback logic
- `pkg/codex/dispatch.go` — DispatchBatch contract, error handling
- `cmd/codex_continue.go` — Continue orchestration, bypass paths, reconcile logic, atomicity gap
- `cmd/codex_build.go` — Build dispatch, synthetic flag
- `cmd/codex_build_worktree.go` — Worktree claim verification, in-repo gap
- `cmd/dispatch_runtime.go` — dispatchBatchByWaveWithVisuals wrapper
- `pkg/storage/storage.go` — AtomicWrite implementation
- `cmd/midden_cmds.go` — midden-write command

### Secondary (MEDIUM confidence)
- `.aether/oracle/progress.md` — Oracle remediation plan with 17 executable steps
- `.aether/oracle/synthesis.md` — Oracle synthesis of findings
- `cmd/codex_continue_test.go` — Existing tests showing expected behavior and bypass coverage
- `cmd/codex_build_test.go` — Build tests, claim verification tests

### Tertiary (LOW confidence)
- Oracle P0-6 mentions `isEnvironmentalConstraintText()` matcher at lines 777-800 — this function was not found in current source via grep, suggesting it may have been removed or the line numbers refer to a different version. The actual environmental bypass is the watcher timeout handling at line 696.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all in-tree Go code, inspected directly
- Architecture: HIGH — code paths traced manually
- Pitfalls: MEDIUM — some test interactions may be subtle

**Research date:** 2026-04-22
**Valid until:** 2026-05-22 (stable Go codebase)
