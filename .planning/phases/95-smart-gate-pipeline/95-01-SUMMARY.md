---
phase: 95-smart-gate-pipeline
plan: 01
subsystem: gate-engine
tags: [auto-resolve, soft-block, verification-depth, gate-classification, go-pretty]

# Dependency graph
requires:
  - phase: 93-gate-classification-infrastructure
    provides: GateClassificationTier, gateClassify(), isHardBlockGate(), QueenAnnotation, GateCheckResult, gateResultsWritePhase(), gateResultsReadPhase(), gateClassifyCmd pattern
provides:
  - gateAutoResolveThresholds map with 6 soft_block gate entries
  - autoResolveSoftBlockGates() for evaluating failed soft_block gates against thresholds
  - autoResolveDepthMultiplier() for depth-based threshold adjustment (light=1.5, standard=1.0, heavy=0.0)
  - shouldAutoResolve() helper for binary gate evaluation
  - annotateGateResult() for in-place QueenAnnotation persistence
  - gate-auto-resolve CLI command with table and JSON output
  - 11 test functions covering all auto-resolve behaviors
affects: [95-02-PLAN, continue-finalize, fixer-dispatch]

# Tech tracking
tech-stack:
  added: []
  patterns: [binary-gate-threshold-evaluation, depth-multiplier-pattern, in-place-annotation-preservation]

key-files:
  created: []
  modified:
    - cmd/gate.go
    - cmd/gate_test.go

key-decisions:
  - "Heavy depth multiplier is 0.0 (not 0.5) -- user asked for thorough checking so no auto-resolve at heavy depth"
  - "Binary gates with threshold 0.0 auto-resolve when multiplier > 0, checked via shouldAutoResolve helper"
  - "autoResolveSoftBlockGates does NOT call annotateGateResult internally -- Plan 02 handles annotation during finalize persistence"
  - "Test JSON output parsed via outputOK wrapper format ({ok:true, result:{...}})"

patterns-established:
  - "Binary gate auto-resolve: threshold 0.0 gates auto-resolve when depth multiplier > 0"
  - "Depth multiplier: light=1.5 (aggressive), standard=1.0 (normal), heavy=0.0 (disabled)"
  - "In-place annotation: annotateGateResult reads, annotates QueenAnnotation pointer, writes back without touching Detail/FixHint/RecoveryOptions"

requirements-completed: [GATE-03, GATE-04]

# Metrics
duration: 11min
completed: 2026-05-03
---

# Phase 95 Plan 01: Auto-Resolve Engine Summary

**Threshold-based auto-resolve engine for soft_block gates with depth-aware multiplier and per-phase annotation persistence**

## Performance

- **Duration:** 11 min
- **Started:** 2026-05-03T16:02:51Z
- **Completed:** 2026-05-03T16:13:51Z
- **Tasks:** 2 (TDD: RED + GREEN combined into 2 commits)
- **Files modified:** 2

## Accomplishments
- Built auto-resolve engine that evaluates failed soft_block gates against thresholds
- Implemented depth-aware multiplier: light (1.5x aggressive), standard (1.0x), heavy (0.0x disabled)
- Added in-place annotation persistence that preserves original gate finding fields
- Created gate-auto-resolve CLI command with table and JSON output
- 11 comprehensive tests covering all tiers, depth levels, mixed results, and edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests** - `08af13f8` (test)
2. **Task 1 GREEN + Task 2: Implementation + test fixes** - `87dc83a8` (feat)

_Note: TDD flow -- RED commit (failing tests) followed by GREEN commit (implementation passing all tests). Task 2's tests were included in Task 1's RED commit since both tasks share the same test file and TDD cycle._

## Files Created/Modified
- `cmd/gate.go` - Added gateAutoResolveThresholds map, autoResolveSoftBlockGates(), autoResolveDepthMultiplier(), shouldAutoResolve(), annotateGateResult(), gateAutoResolveCmd, renderGateAutoResolveTable()
- `cmd/gate_test.go` - Added 11 test functions: TestAutoResolveSoftBlock, TestAutoResolveHardBlockNever, TestAutoResolveAdvisoryIgnored, TestAutoResolveDepthMultiplier, TestAutoResolveHeavySkipsAll, TestAnnotateGateResultPreservesOriginal, TestAutoResolveMixedResults, TestAutoResolveUnclassifiedGate, TestAutoResolveEmptyReport, TestGateAutoResolveCmdJSON, TestGateAutoResolveCmdTable

## Decisions Made
- Heavy depth multiplier is 0.0 (not 0.5) -- when user asks for heavy verification, no soft_block gates auto-resolve. This is the safe default.
- Binary gates (all 6 current soft_block gates) auto-resolve when the depth multiplier is positive. The threshold value 0.0 is irrelevant for binary gates; only the multiplier matters.
- autoResolveSoftBlockGates flips Passed=true in memory but does NOT write annotations -- Plan 02's finalize integration handles persistence.
- Test for gate-auto-resolve --json parses the outputOK wrapper format ({ok:true, result:{...}}) rather than raw JSON.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed binary gate auto-resolve logic (effectiveThreshold <= 0 blocked all resolves)**
- **Found during:** Task 1 GREEN phase
- **Issue:** shouldAutoResolve checked effectiveThreshold <= 0 which blocked all binary gates (threshold 0.0 * multiplier 1.0 = 0.0)
- **Fix:** Changed shouldAutoResolve to take threshold and multiplier separately, checking multiplier > 0 for binary gates
- **Files modified:** cmd/gate.go, cmd/gate_test.go
- **Verification:** All 11 tests pass
- **Committed in:** 87dc83a8

**2. [Rule 3 - Blocking] Fixed JSON output test to handle outputOK wrapper**
- **Found during:** Task 1 GREEN phase
- **Issue:** TestGateAutoResolveCmdJSON parsed output as direct map, but outputOK wraps in {ok:true, result:{...}}
- **Fix:** Updated test to parse the wrapper struct
- **Files modified:** cmd/gate_test.go
- **Verification:** Test passes
- **Committed in:** 87dc83a8

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both auto-fixes were necessary for correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Auto-resolve engine is ready for Plan 02 to wire into the continue finalize flow
- autoResolveSoftBlockGates() returns updated report + resolved list, designed for integration at the gate failure point in codex_continue_finalize.go
- annotateGateResult() is ready for Plan 02 to call during persistence
- Fixer dispatch integration (for gates above threshold) also pending Plan 02

---
*Phase: 95-smart-gate-pipeline*
*Completed: 2026-05-03*

## Self-Check: PASSED

- FOUND: cmd/gate.go
- FOUND: cmd/gate_test.go
- FOUND: .planning/phases/95-smart-gate-pipeline/95-01-SUMMARY.md
- FOUND: 08af13f8 (RED commit)
- FOUND: 87dc83a8 (GREEN commit)
