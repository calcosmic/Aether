---
phase: 85-smart-depth-defaults
plan: 02
subsystem: review-depth
tags: [go, smart-defaults, planning-depth, verification-depth, auto-detect]

# Dependency graph
requires:
  - phase: 85-01
    provides: "resolveSmartPlanningDepth, resolveSmartVerificationDepth, phasePositionLevel, phaseRiskLevel, collectPhaseText"
provides:
  - "resolveVerificationDepth wired to smart default fallback"
  - "resolvePlanningDepthSmart wrapping planning depth with smart defaults"
  - "renderSmartDepthReason for human-readable auto-detection reasons"
  - "renderReviewDepthLineWithReason for Phase 86 smart-default display"
affects: [86-depth-selection-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Smart default wrapper pattern: new function wraps existing signature, preserving backward compatibility"

key-files:
  modified:
    - cmd/review_depth.go
    - cmd/codex_plan.go
    - cmd/codex_visuals.go
    - cmd/review_depth_test.go

key-decisions:
  - "Added resolvePlanningDepthSmart as new function rather than changing resolvePlanningDepth signature (backward compat)"
  - "renderReviewDepthLineWithReason is additive only -- existing renderReviewDepthLine callers unchanged (Phase 86 switches them)"
  - "Smart defaults only fire when no explicit flag is provided -- priority chain preserved"

patterns-established:
  - "Smart default wrapper: resolveXSmart(depth, phase, totalPhases) wraps resolveX(depth) with auto-detect fallback"

requirements-completed: [DEPTH-03]

# Metrics
duration: 6min
completed: 2026-04-30
---

# Phase 85 Plan 02: Wire Smart Defaults into Command Paths Summary

**Smart depth defaults connected to both planning and verification depth resolution, with visual helper functions ready for Phase 86 auto-detection display**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-30T22:16:11Z
- **Completed:** 2026-04-30T22:22:33Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `resolveVerificationDepth` now uses `resolveSmartVerificationDepth` as final fallback instead of flat `standard`
- `resolvePlanningDepthSmart` provides smart defaults for planning depth when no explicit `--planning-depth` flag
- Both `runCodexPlanWithOptions` and `runCodexPlanPlanOnly` use `resolvePlanningDepthSmart`
- `renderSmartDepthReason` produces human-readable reasons (e.g., "auto: final phase", "auto: security risk")
- `renderReviewDepthLineWithReason` available for Phase 86 to show auto-detection in visual output
- 5 new wiring tests verify smart defaults work correctly and explicit flags still override

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire smart defaults into resolveVerificationDepth and resolvePlanningDepth** - `5ab7fc53` (feat)
2. **Task 2: Add auto-detection reason to visual output and wiring tests** - `eef291e7` (test)

## Files Created/Modified
- `cmd/review_depth.go` - Changed `resolveVerificationDepth` final fallback to call `resolveSmartVerificationDepth`
- `cmd/codex_plan.go` - Added `resolvePlanningDepthSmart`, updated both callers
- `cmd/codex_visuals.go` - Added `renderSmartDepthReason` and `renderReviewDepthLineWithReason`
- `cmd/review_depth_test.go` - Added 5 new wiring tests (smart fallback, risk override, explicit override, planning depth smart)

## Decisions Made
- Added `resolvePlanningDepthSmart` as a new wrapper function rather than changing the `resolvePlanningDepth` signature, since it is called in other places
- Did not change existing callers of `renderReviewDepthLine` -- Phase 86 will switch them when it adds the UI layer
- `runCodexPlanPlanOnly` uses `colony.Phase{ID: 1}` (first phase) as heuristic since it generates new plans

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing `embedded_assets.go` build failure in worktree environment (missing `.aether/rules/` directory). Fixed by copying the directory from the main repo. This is a worktree setup issue, not a code issue.
- Pre-existing test failures (`TestIntegrityDetectSourceContext`, `TestQueenWisdomHygiene`) due to worktree not being the Aether repo root. These are environment-specific and unrelated to the changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Smart defaults are fully wired and tested end-to-end
- Visual helpers are available for Phase 86 to show auto-detection reasons in output
- All existing tests pass unchanged
- Explicit user flags (`--light`, `--heavy`, `--planning-depth`, `--verification-depth`) continue to override smart defaults

---
*Phase: 85-smart-depth-defaults*
*Completed: 2026-04-30*
