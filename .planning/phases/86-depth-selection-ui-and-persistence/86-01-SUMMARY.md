---
phase: 86-depth-selection-ui-and-persistence
plan: 01
subsystem: ui
tags: [go, depth-selection, cli-flags, smart-defaults, colony-state]

# Dependency graph
requires:
  - phase: 85-depth-selection-smart-defaults
    provides: "resolveSmartVerificationDepth, renderSmartDepthReason, renderReviewDepthLineWithReason, Phase 85 visual helpers"
provides:
  - "--verification-depth CLI flag on plan command with smart default resolution"
  - "resolveVerificationDepthSmart function with validation and alias support"
  - "ColonyState.VerificationDepth field for cross-command persistence"
  - "Depth selection banner in renderPlanVisual showing both depths with reasons"
  - "verification_depth, verification_smart_default, planning_smart_default, planning_phase keys in all plan result maps"
affects: [codex_plan, codex_visuals, depth-selection-banner, build-manifest]

# Tech tracking
tech-stack:
  added: []
  patterns: [smart-default-with-explicit-override pattern, depth-selection-banner-with-reason-display]

key-files:
  created: []
  modified:
    - cmd/codex_workflow_cmds.go
    - cmd/codex_plan.go
    - cmd/review_depth.go
    - cmd/review_depth_test.go
    - cmd/codex_visuals.go
    - pkg/colony/colony.go

key-decisions:
  - "Mirrored --planning-depth pattern exactly for --verification-depth to maintain UI consistency"
  - "Used ColonyState.VerificationDepth for persistence so build/continue commands read from state rather than re-resolving"

patterns-established:
  - "Smart default with explicit override: resolve function checks for empty input, delegates to smart resolver, validates aliases, returns canonical value"

requirements-completed: [DEPTH-04]

# Metrics
duration: 0min
completed: 2026-05-01
---

# Phase 86 Plan 01: Verification Depth Flag, Smart Resolution, ColonyState Persistence, and Depth Selection Banner Summary

**--verification-depth flag on plan command with smart default resolution via resolveVerificationDepthSmart, depth selection banner with reason display, and ColonyState persistence for downstream build consumption.**

## Performance

- **Duration:** 0 min (previously completed on feature branch)
- **Started:** 2026-05-01T12:52:03Z
- **Completed:** 2026-05-01T12:52:03Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- --verification-depth flag registered on planCmd, read in RunE, passed through codexPlanOptions
- resolveVerificationDepthSmart function validates input against known aliases (light/minimal/coarse, standard/default, heavy/full/thorough) and delegates to resolveSmartVerificationDepth for empty input
- ColonyState.VerificationDepth field stores resolved depth, persisted via store.SaveJSON during plan execution
- Depth selection banner in renderPlanVisual uses renderStageMarker("Depth Selection"), displays both planning and verification depth with full reason strings from renderSmartDepthReason and renderReviewDepthLineWithReason
- Both result maps (normal and plan-only) include verification_depth, verification_smart_default, planning_smart_default, and planning_phase keys
- codexPlanManifest includes VerificationDepth field for build manifest consumption
- Tests cover explicit values, smart defaults, and invalid input rejection

## Task Commits

Each task was committed atomically:

1. **Task 1: Add --verification-depth flag, resolveVerificationDepthSmart, ColonyState persistence, and wire into plan options** - `e9af37c3` (feat)
2. **Task 2: Add depth selection banner to renderPlanVisual using Phase 85 visual helpers** - `ccbd43ae` (feat)

## Files Created/Modified
- `cmd/codex_workflow_cmds.go` - Added --verification-depth flag registration and RunE wiring
- `cmd/codex_plan.go` - Added VerificationDepth to codexPlanOptions and codexPlanManifest, resolution logic, ColonyState persistence, result map keys
- `cmd/review_depth.go` - Added resolveVerificationDepthSmart function
- `cmd/review_depth_test.go` - Added TestResolveVerificationDepthSmart_ExplicitValue, TestResolveVerificationDepthSmart_EmptyUsesSmartDefault, TestResolveVerificationDepthSmart_InvalidValue
- `cmd/codex_visuals.go` - Added depth selection banner with stage marker and reason display
- `pkg/colony/colony.go` - Added VerificationDepth string field to ColonyState

## Decisions Made
- Mirrored the --planning-depth pattern exactly for --verification-depth to maintain consistent UX
- ColonyState persistence chosen over re-resolution so build/continue commands get the exact same depth the user selected or was auto-selected during planning

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Worktree missing `.aether/rules/` directory causing embedded_assets.go build failure (Rule 3 blocking issue). Fixed by copying rules from main repo. This is a pre-existing worktree setup issue, not related to plan changes.
- Two pre-existing test failures in full cmd test suite (TestIntegrityDetectSourceContext and TestQueenWisdomHygiene) are worktree environment issues, not caused by plan changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All plan 86-01 deliverables verified present and functional
- ColonyState.VerificationDepth is populated during plan execution, ready for build/continue consumption
- No blockers for downstream plans

## Self-Check: PASSED

- 86-01-SUMMARY.md: FOUND
- Commit e9af37c3: FOUND
- Commit ccbd43ae: FOUND
- cmd/codex_workflow_cmds.go: FOUND
- cmd/codex_plan.go: FOUND
- cmd/review_depth.go: FOUND
- cmd/review_depth_test.go: FOUND
- cmd/codex_visuals.go: FOUND
- pkg/colony/colony.go: FOUND

---
*Phase: 86-depth-selection-ui-and-persistence*
*Completed: 2026-05-01*
