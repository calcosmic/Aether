---
phase: 83-planning-depth-system
plan: 01
subsystem: cli
tags: [go, planning-depth, cobra, tdd]

# Dependency graph
requires: []
provides:
  - PlanningDepth type with light/standard/deep constants, Valid(), NormalizePlanningDepth()
  - --planning-depth CLI flag on aether plan command
  - planning_depth field in codexPlanManifest and all result maps
  - planning_depth in wrapper_contract with updated source_command
  - Visual output showing planning depth (hidden for standard default)
  - 9 test functions covering type validation, normalization, CLI wiring, manifest, and wrapper contract
affects: [83-02-planning-depth-wrapper-md]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PlanningDepth type mirrors PlanGranularity pattern (type + constants + Valid() + Normalize function)"
    - "resolvePlanningDepth follows resolvePlanGranularityDepth pattern (normalize + validate + error)"
    - "Visual output hidden for standard default to reduce noise"

key-files:
  created: []
  modified:
    - pkg/colony/colony.go
    - pkg/colony/colony_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/codex_plan.go
    - cmd/codex_plan_test.go
    - cmd/codex_visuals.go

key-decisions:
  - "Follow existing PlanGranularity pattern exactly for PlanningDepth type"
  - "Hide standard depth from visual output (default, no need to clutter)"
  - "Resolve planningDepth inside runCodexPlanPlanOnly rather than adding parameter"
  - "Validate explicit user input but default empty to standard (per D-08)"
  - "Do NOT add PlanningDepth to ColonyState -- persistence is Phase 86 (per D-07)"

patterns-established:
  - "PlanningDepth type pattern: type PlanningDepth string + Valid() + NormalizePlanningDepth()"
  - "resolvePlanningDepth pattern: normalize via colony package, validate explicit input, error on unknown"

requirements-completed: [DEPTH-01]

# Metrics
duration: 11min
completed: 2026-04-30
---

# Phase 83 Plan 01: PlanningDepth Type and CLI Flag Summary

**PlanningDepth type with light/standard/deep values, alias normalization, CLI --planning-depth flag, manifest integration, and visual output**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-30T18:41:30Z
- **Completed:** 2026-04-30T18:52:12Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- PlanningDepth type in pkg/colony/colony.go with Valid() method, NormalizePlanningDepth() function, and sentinel error
- --planning-depth CLI flag on `aether plan` with light/standard/deep values and alias support (minimal/coarse -> light, granular/thorough -> deep)
- planning_depth field added to codexPlanManifest JSON, all four result maps, and wrapper_contract
- Visual output shows "Planning depth: X" only when non-standard (light or deep)
- 9 test functions covering type validation, normalization, defaults, aliases, invalid input, granularity independence, manifest inclusion, and wrapper contract

## Task Commits

Each task was committed atomically (TDD: test -> feat):

1. **Task 1: Add PlanningDepth type and unit tests** - `62e02ad6` (test), `12740b5f` (feat)
2. **Task 2: Wire --planning-depth flag through CLI, manifest, result maps, visual output, and tests** - `123589ba` (test), `d82d4b73` (feat)

## Files Created/Modified
- `pkg/colony/colony.go` - Added PlanningDepth type, constants, Valid(), NormalizePlanningDepth(), ErrInvalidPlanningDepth
- `pkg/colony/colony_test.go` - Added TestPlanningDepthValid and TestNormalizePlanningDepth
- `cmd/codex_workflow_cmds.go` - Registered --planning-depth flag, parsed value in RunE, passed through codexPlanOptions
- `cmd/codex_plan.go` - Added PlanningDepth to codexPlanOptions and codexPlanManifest, added resolvePlanningDepth function, wired planning_depth into all result maps and wrapper_contract
- `cmd/codex_plan_test.go` - Added 7 test functions: TestPlanningDepthDefault, TestPlanningDepthValues, TestPlanningDepthAliases, TestPlanningDepthInvalid, TestPlanningDepthIndependentOfGranularity, TestPlanningDepthInManifest, TestPlanningDepthInWrapperContract
- `cmd/codex_visuals.go` - Added planning depth line in renderPlanVisual (hidden for standard)

## Decisions Made
- Followed the existing PlanGranularity type pattern exactly for consistency
- Hidden standard depth from visual output since it's the default and would add noise
- Resolved planningDepth inside runCodexPlanPlanOnly rather than adding a parameter to keep the function signature simpler
- Validated explicit user input against known aliases but defaulted empty strings to standard per D-08
- Did NOT add PlanningDepth to ColonyState -- persistence is deferred to Phase 86 per D-07

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created missing .aether/rules directory for embed directive**
- **Found during:** Task 2 (RED phase test compilation)
- **Issue:** `go test ./cmd/` failed with "pattern all:.aether/rules: no matching files found" -- the worktree was missing the `.aether/rules/` directory required by the embed directive in `embedded_assets.go`
- **Fix:** Created `.aether/rules/` directory with `.gitkeep` placeholder
- **Files modified:** .aether/rules/.gitkeep (created, not committed -- gitignored)
- **Verification:** `go test ./cmd/` compilation succeeded after fix
- **Committed in:** Not committed (generated/runtime file)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The embed fix was necessary to run tests in the worktree environment. No code changes resulted.

## Issues Encountered
- Worktree environment missing `.aether/rules/` directory (embed directive requirement) -- fixed by creating the directory
- Two pre-existing test failures in worktree environment (TestIntegrityDetectSourceContext, TestQueenWisdomHygiene) -- unrelated to this plan, caused by worktree isolation

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (wrapper markdown updates) can now reference the planning_depth field in wrapper_contract
- All runtime plumbing complete -- Plan 02 only needs markdown wrapper updates
- No blockers

---
*Phase: 83-planning-depth-system*
*Completed: 2026-04-30*
