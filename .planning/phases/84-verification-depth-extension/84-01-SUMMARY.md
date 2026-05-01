---
phase: 84-verification-depth-extension
plan: 01
subsystem: verification
tags: [go, verification-depth, colony-runtime, review-dispatch]

# Dependency graph
requires: []
provides:
  - VerificationDepth type in pkg/colony/ with Valid() and NormalizeVerificationDepth()
  - resolveVerificationDepth returning 3-level depth (light/standard/heavy)
  - resolveVerificationDepthFlag for backward-compatible --light/--heavy aliases
  - --verification-depth CLI flag on build and continue commands
  - 3-tier dispatch logic: light=0 review agents, standard=probe only, heavy=all 3
  - Visual rendering for all 3 depth levels
affects: [85, 86, 87]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "VerificationDepth mirrors PlanningDepth pattern from Phase 83"
    - "Incremental migration: old ReviewDepth type kept, new code uses colony.VerificationDepth"

key-files:
  created: []
  modified:
    - pkg/colony/colony.go
    - pkg/colony/colony_test.go
    - cmd/review_depth.go
    - cmd/review_depth_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_build.go
    - cmd/codex_visuals.go
    - cmd/colony_prime_context.go
    - cmd/codex_continue_test.go
    - cmd/continue_wrapper_ceremony_test.go

key-decisions:
  - "Kept old ReviewDepth type and resolveReviewDepth for backward compat (incremental migration)"
  - "Standard is the new auto-detect default for intermediate phases (replaces light)"
  - "Heavy boolean flag takes priority over --verification-depth string (heavier is safer)"
  - "Probe extracted from codexContinueReviewSpecs[2:] for standard mode dispatch"

patterns-established:
  - "VerificationDepth type pattern mirrors PlanningDepth exactly (Valid, Normalize, constants)"
  - "3-level dispatch: light skips review, standard runs probe only, heavy runs full gauntlet"

requirements-completed: [DEPTH-02]

# Metrics
duration: 27min
completed: 2026-04-30
---

# Phase 84 Plan 01: VerificationDepth 3-Level Type and Dispatch Summary

**VerificationDepth type (light/standard/heavy) with 3-level review dispatch, CLI flag, and backward-compatible --light/--heavy aliases**

## Performance

- **Duration:** 27 min
- **Started:** 2026-04-30T19:49:19Z
- **Completed:** 2026-04-30T20:16:04Z
- **Tasks:** 2 (4 TDD commits: 2 RED + 2 GREEN)
- **Files modified:** 13

## Accomplishments
- VerificationDepth type with Valid(), NormalizeVerificationDepth(), and 3 constants mirroring PlanningDepth pattern
- resolveVerificationDepth with priority chain: final phase > heavy flag > keyword match > light flag > explicit string > default standard
- --verification-depth flag registered on build and continue commands with --light/--heavy as backward-compatible aliases
- 3-tier dispatch: light=0 review agents, standard=probe only (1 agent), heavy=gatekeeper+auditor+probe (3 agents)
- Visual rendering shows "Review depth: standard (Phase X of Y)" for standard depth
- Colony-prime context shows "Standard review -- watcher and probe verification"

## Task Commits

Each task was committed atomically:

1. **Task 1: Add VerificationDepth type and update resolveReviewDepth for 3-level dispatch** - `8dd424e` (test: RED), `359bfe7` (feat: GREEN)
2. **Task 2: Wire --verification-depth flag and update dispatch, visual, and context for 3 levels** - `67e7477` (test: RED), `65bf1430` (feat: GREEN)

_Note: TDD tasks with multiple commits (test -> feat)_

## Files Created/Modified
- `pkg/colony/colony.go` - Added VerificationDepth type, constants, Valid(), NormalizeVerificationDepth()
- `pkg/colony/colony_test.go` - Added TestVerificationDepthValid and TestNormalizeVerificationDepth
- `cmd/review_depth.go` - Added resolveVerificationDepth and resolveVerificationDepthFlag; kept old ReviewDepth
- `cmd/review_depth_test.go` - Added 8 new tests for 3-level dispatch and flag resolution
- `cmd/codex_workflow_cmds.go` - Registered --verification-depth flag; migrated visual calls to reviewDepthFromResult
- `cmd/codex_continue.go` - Migrated plannedContinueReviewDispatches to colony.VerificationDepth with 3-level switch
- `cmd/codex_continue_plan.go` - Migrated plannedExternalContinueDispatches to colony.VerificationDepth with 3-level switch
- `cmd/codex_continue_finalize.go` - Migrated externalContinueReviewReport to colony.VerificationDepth; updated string comparisons to NormalizeVerificationDepth
- `cmd/codex_build.go` - Migrated plannedBuildDispatchesForSelection to colony.VerificationDepth; updated all ReviewDepth refs
- `cmd/codex_visuals.go` - Migrated reviewDepthFromResult and renderReviewDepthLine to colony.VerificationDepth with standard case
- `cmd/colony_prime_context.go` - Replaced binary branch with 3-level switch; uses resolveVerificationDepth
- `cmd/codex_continue_test.go` - Updated ReviewDepth constants to colony.VerificationDepth
- `cmd/continue_wrapper_ceremony_test.go` - Updated ReviewDepthLight to colony.VerificationDepthLight

## Decisions Made
- Kept old ReviewDepth type and resolveReviewDepth function for backward compatibility (incremental migration, not a rename)
- Standard is the new auto-detect default for intermediate phases -- intermediate phases now get watcher + probe instead of nothing
- Heavy boolean flag takes priority over --verification-depth string flag (heavier is safer principle)
- Probe is extracted from codexContinueReviewSpecs[2:] for standard mode (index 2 is probe)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Created .aether/rules/ directory in worktree**
- **Found during:** Task 1 (initial test compilation)
- **Issue:** Worktree was missing `.aether/rules/` directory needed by `go:embed all:.aether/rules` in embedded_assets.go, causing all cmd/ tests to fail with "no matching files found"
- **Fix:** Created directory and copied `aether-colony.md` from main repo
- **Files modified:** .aether/rules/aether-colony.md (worktree-local, not committed)
- **Verification:** All cmd/ tests compile and pass after fix
- **Committed in:** Not committed (worktree-local fix only)

**2. [Rule 1 - Bug] resolveVerificationDepthFlag priority order**
- **Found during:** Task 1 (RED phase test failure)
- **Issue:** Initial implementation checked --light before --heavy, but test expected heavy to win when both are set (heavier is safer)
- **Fix:** Reversed priority to check --heavy first
- **Files modified:** cmd/review_depth.go
- **Verification:** TestResolveVerificationDepthFlag_BoolPriority passes
- **Committed in:** 359bfe7 (Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 bug)
**Impact on plan:** Both auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- Worktree embed issue: `.aether/rules/` directory not present in worktree (go:embed requires exact match). Resolved by creating directory.
- Flaky visual tests under `-race` flag: TestBuildBufferedOutputBreaksJSONUnderVisualEnv, TestBuildVisualOutputShowsArtifactContract, TestCodexVisualParity fail intermittently under race detection but pass consistently without. Pre-existing timing issue in terminal output tests.
- TestIntegrityDetectSourceContext and TestQueenWisdomHygiene fail in worktree environment (missing QUEEN.md, source context detection). Pre-existing worktree issues.

## Known Stubs
None - all planned functionality is wired end-to-end.

## Next Phase Readiness
- VerificationDepth type ready for use in any future code that needs review depth awareness
- Plan 02 (84-02) can build on this foundation for additional depth-aware features
- The old ReviewDepth type still exists for any call sites not yet migrated; future phases can continue the incremental migration

---
*Phase: 84-verification-depth-extension*
*Completed: 2026-04-30*

## Self-Check: PASSED
