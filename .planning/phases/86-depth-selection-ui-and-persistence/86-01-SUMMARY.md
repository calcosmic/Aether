---
phase: 86-depth-selection-ui-and-persistence
plan: 01
subsystem: cli-ux
tags: [go, cobra, depth-selection, verification-depth, colony-state, plan-command]

# Dependency graph
requires:
  - phase: 85
    provides: "resolveSmartVerificationDepth, renderSmartDepthReason, renderReviewDepthLineWithReason visual helpers"
provides:
  - "--verification-depth CLI flag on plan command"
  - "resolveVerificationDepthSmart function with smart defaults"
  - "ColonyState.VerificationDepth field for cross-command persistence"
  - "Depth Selection banner in plan visual output with stage marker style"
  - "Result map enrichment with verification_depth, smart_default flags, and planning_phase object"
affects: [86-02, build-command, continue-command]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mirror planning-depth pattern for verification-depth: flag -> options struct -> smart resolver -> ColonyState persistence -> result map"

key-files:
  created: []
  modified:
    - cmd/review_depth.go
    - cmd/review_depth_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_plan.go
    - cmd/codex_visuals.go
    - pkg/colony/colony.go

key-decisions:
  - "Mirrored resolvePlanningDepthSmart pattern exactly for resolveVerificationDepthSmart"
  - "Persist verification depth in ColonyState so downstream build/continue commands can read it without re-resolving"
  - "Used existing renderReviewDepthLineWithReason visual helper (outputs 'Review depth:' label) for verification depth in banner"

patterns-established:
  - "Depth flag pattern: flag -> options -> smart resolver -> ColonyState -> result map -> visual banner"

requirements-completed: [DEPTH-04]

# Metrics
duration: 8min
completed: 2026-05-01
---

# Phase 86 Plan 01: Verification Depth Flag and Smart Resolution Summary

**--verification-depth flag with smart defaults, ColonyState persistence, and depth selection banner using Phase 85 visual helpers**

## Performance

- **Duration:** 8 min
- **Started:** 2026-05-01T11:42:08Z
- **Completed:** 2026-05-01T11:50:58Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added `--verification-depth` flag to plan command with light/standard/heavy values
- Implemented `resolveVerificationDepthSmart` with smart defaults based on phase risk and position
- Added `VerificationDepth` field to ColonyState for cross-command persistence (DEPTH-04)
- Enriched plan result maps with verification_depth, verification_smart_default, planning_smart_default, and planning_phase keys
- Created Depth Selection banner in renderPlanVisual using stage marker style with full reason display

## Task Commits

Each task was committed atomically:

1. **Task 1: Add --verification-depth flag, resolveVerificationDepthSmart, ColonyState persistence, and wire into plan options** - `e9af37c3` (feat)
2. **Task 2: Add depth selection banner to renderPlanVisual using Phase 85 visual helpers** - `ccbd43ae` (feat)

## Files Created/Modified
- `cmd/review_depth.go` - Added resolveVerificationDepthSmart function mirroring planning depth pattern
- `cmd/review_depth_test.go` - Added 7 subtests covering explicit values, smart defaults, and invalid input
- `cmd/codex_workflow_cmds.go` - Added --verification-depth flag registration and RunE wiring
- `cmd/codex_plan.go` - Added VerificationDepth to options/manifest structs, wired resolution in both runCodexPlanWithOptions and runCodexPlanPlanOnly, added result map keys, ColonyState persistence
- `cmd/codex_visuals.go` - Replaced conditional planning_depth display with full Depth Selection banner
- `pkg/colony/colony.go` - Added VerificationDepth string field to ColonyState struct

## Decisions Made
- Followed the exact resolvePlanningDepthSmart pattern for resolveVerificationDepthSmart to maintain consistency
- Stored verification depth in ColonyState immediately during plan execution (same pattern as state-mutate) so build command can read it without re-resolving
- Used renderReviewDepthLineWithReason as-is for verification depth display (outputs "Review depth:" label rather than "Verification depth:" to reuse Phase 85 helpers)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Worktree build required creating `.aether/rules/` directory to satisfy embedded_assets.go embed pattern. This is a worktree environment issue, not a code issue.
- Two pre-existing test failures in worktree (TestIntegrityDetectSourceContext, TestQueenWisdomHygiene) caused by missing worktree environment files. Not caused by this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- ColonyState now stores verification_depth for downstream build/continue consumption (Plan 02)
- Depth Selection banner renders in plan visual with full reason display
- All verification depth tests pass (7/7 subtests)

---
*Phase: 86-depth-selection-ui-and-persistence*
*Completed: 2026-05-01*
