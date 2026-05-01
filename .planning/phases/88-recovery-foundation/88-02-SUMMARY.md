---
phase: 88-recovery-foundation
plan: 02
subsystem: gates
tags: [go, circuit-breaker, persistence, gate-checking]

# Dependency graph
requires:
  - phase: 88-01
    provides: gate result persistence base, shouldSkipGate, gateRecoveryTemplates
provides:
  - GateCheckResult struct with Status/FixHint/RecoveryOptions/Timestamp/RetryCount
  - Per-phase gate-results-{N}.json persistence
  - alwaysRunGates map for unconditional gate re-execution
  - Circuit breaker gate retry tracking with phase-scoped keys
  - Structured failure UX (FixHint + RecoveryOptions) on every failing gate
affects: [88-03, 89-01, gate-recovery]

# Tech tracking
tech-stack:
  added: []
  patterns: [per-phase-gate-persistence, structured-failure-ux, gate-retry-circuit-breaker]

key-files:
  created: []
  modified:
    - cmd/gate.go
    - cmd/codex_continue.go
    - cmd/codex_continue_finalize.go
    - cmd/circuit_breaker.go
    - pkg/colony/colony.go
    - cmd/gate_incremental_test.go
    - cmd/gate_test.go

key-decisions:
  - "shouldSkipGate signature changed from []colony.GateResultEntry to []GateCheckResult to support Status-based skip logic"
  - "Per-phase gate results stored in gate-results-{N}.json alongside COLONY_STATE GateResults for backward compatibility"
  - "Package-level circuitBreaker variable in codex_continue.go (nil-safe) for gate retry lifecycle"
  - "alwaysRunGates map replaces hardcoded tests_pass check for cleaner extensibility"

patterns-established:
  - "Per-phase persistence: gate-results-{N}.json files complement COLONY_STATE GateResults"
  - "Structured failure UX: every failing gate carries FixHint + RecoveryOptions for recovery guidance"

requirements-completed: [GATE-01, GATE-02, GATE-04, GATE-05, LOOP-01]

# Metrics
duration: 23min
completed: 2026-05-01
---

# Phase 88 Plan 02: Gate Struct Extensions and Smart Retry Summary

**Extended gate system with FixHint/RecoveryOptions on failures, per-phase gate-results-{N}.json persistence, alwaysRunGates map, and circuit breaker integration for gate retry hard-stops.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-05-01T16:41:57Z
- **Completed:** 2026-05-01T17:05:27Z
- **Tasks:** 2 (TDD: 4 commits -- 2 RED + 2 GREEN)
- **Files modified:** 7

## Accomplishments
- gateCheck struct extended with FixHint and RecoveryOptions fields for structured failure UX
- New GateCheckResult struct with Status enum (passed/failed/skipped/not-reached) for per-phase persistence
- Per-phase gate-results-{N}.json files written and read alongside COLONY_STATE GateResults
- shouldSkipGate refactored to use alwaysRunGates map (tests_pass, flags, watcher_veto, no_critical_flags)
- Circuit breaker gateRetryKey helper generates phase-scoped keys for retry tracking
- Every failing gate in runCodexContinueGates now populates FixHint and RecoveryOptions
- Circuit breaker integration: tripped breaker returns early with clear manual-intervention message

## Task Commits

Each task was committed atomically with TDD (RED then GREEN):

1. **Task 1: Extend gate structs and implement per-phase persistence with smart retry**
   - `61b1a704` (test) -- RED: 10 failing tests for struct extensions, persistence, skip logic, circuit breaker
   - `57faa8dc` (feat) -- GREEN: gateCheck FixHint/RecoveryOptions, GateCheckResult, per-phase persistence, alwaysRunGates map, gateRetryKey, GateResultEntry extension, all callers updated

2. **Task 2: Wire structured gate failures and per-phase persistence into runCodexContinueGates**
   - `453f23a1` (test) -- RED: 4 failing tests for FixHint/RecoveryOptions wiring, flags always-run, skip passed
   - `d994f000` (feat) -- GREEN: FixHint + RecoveryOptions on all failing gates, circuit breaker check/record in continue gates, package-level circuitBreaker variable

## Files Created/Modified
- `cmd/gate.go` - Extended gateCheck struct, added GateCheckResult struct, per-phase persistence functions, alwaysRunGates map, refactored shouldSkipGate signature
- `pkg/colony/colony.go` - Extended GateResultEntry with FixHint and RecoveryOptions fields
- `cmd/circuit_breaker.go` - Added gateRetryKey helper function
- `cmd/codex_continue.go` - Updated runCodexContinueGates with FixHint/RecoveryOptions on failures, circuit breaker integration, per-phase persistence write
- `cmd/codex_continue_finalize.go` - Updated to read prior results from per-phase file, write per-phase gate results
- `cmd/gate_incremental_test.go` - 14 new tests covering struct extensions, persistence, skip logic, circuit breaker, and wiring
- `cmd/gate_test.go` - Updated existing tests for shouldSkipGate signature change

## Decisions Made
- **shouldSkipGate signature change**: Changed from `[]colony.GateResultEntry` to `[]GateCheckResult` to support Status-based skip logic (skipping "passed" and "skipped" gates, not just checking `Passed` bool). All callers updated across 4 files.
- **Per-phase file alongside COLONY_STATE**: Per-phase gate-results-{N}.json provides richer data (Status, RetryCount) while COLONY_STATE GateResults remains the backward-compatible format used by other subsystems.
- **Package-level circuitBreaker**: Created a nil-safe package-level variable in codex_continue.go rather than passing it through function signatures, since the continue flow is the only consumer of gate retry circuit breaking.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Missing .aether/rules/ directory in worktree**
- **Found during:** Task 1 (RED phase -- test compilation)
- **Issue:** Worktree lacked `.aether/rules/` directory, causing embed pattern `all:.aether/rules` to fail during test compilation
- **Fix:** Created `.aether/rules/.gitkeep` placeholder
- **Files modified:** `.aether/rules/.gitkeep` (created)
- **Verification:** Tests compiled successfully after fix
- **Committed in:** `61b1a704` (part of Task 1 RED commit)

**2. [Rule 1 - Bug] Extra closing brace from partial function replacement**
- **Found during:** Task 2 (GREEN phase -- syntax error at line 2501)
- **Issue:** Partial replacement of runCodexContinueGates left an orphaned closing brace
- **Fix:** Removed the extra brace via Python line deletion
- **Files modified:** `cmd/codex_continue.go`
- **Verification:** Compilation succeeded after fix
- **Committed in:** `d994f000` (part of Task 2 GREEN commit)

**3. [Rule 2 - Missing Critical] Second caller of runCodexContinueGates missed in plan**
- **Found during:** Task 1 (GREEN phase -- compilation error at codex_continue.go:515)
- **Issue:** Plan listed the caller in codex_continue_finalize.go but missed the direct caller in codex_continue.go line 515 (the non-finalize continue path)
- **Fix:** Updated both callers to read from per-phase file and pass []GateCheckResult; added per-phase persistence write to both paths
- **Files modified:** `cmd/codex_continue.go`
- **Verification:** Full test suite passes (only pre-existing failures remain)
- **Committed in:** `57faa8dc` (part of Task 1 GREEN commit)

**4. [Rule 1 - Bug] TestShouldSkipGateCmd test broke after CLI update**
- **Found during:** Task 1 (GREEN phase -- test failure)
- **Issue:** shouldSkipGateCmd now reads from per-phase file when --phase is provided, but the test wrote to COLONY_STATE.json without --phase flag
- **Fix:** Updated test to write per-phase file and pass --phase flag
- **Files modified:** `cmd/gate_test.go`
- **Verification:** TestShouldSkipGateCmd_PassedGate passes
- **Committed in:** `57faa8dc` (part of Task 1 GREEN commit)

---

**Total deviations:** 4 auto-fixed (1 missing critical, 3 bugs)
**Impact on plan:** All auto-fixes were necessary for correctness. The second caller (Rule 2) was a genuine oversight in the plan -- without it, the non-finalize continue path would have been broken.

## Issues Encountered
- Tab vs space indentation mismatch caused Edit tool failures when updating gate.go and gate_test.go -- resolved using Python for direct string replacement
- Pre-existing test failures (TestIntegrityDetectSourceContext, TestQueenWisdomHygiene) confirmed unrelated to changes

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Per-phase gate results infrastructure ready for 88-03 (gate recovery banners)
- GateCheckResult Status field supports "not-reached" for plans that need unexecuted gate tracking
- Circuit breaker integration ready for /ant-unblock command (88-03)
- FixHint and RecoveryOptions populated on all failing gates for recovery banner rendering

---
*Phase: 88-recovery-foundation*
*Completed: 2026-05-01*
