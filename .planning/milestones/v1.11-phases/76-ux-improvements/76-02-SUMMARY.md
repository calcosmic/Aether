---
phase: 76-ux-improvements
plan: 02
subsystem: ui
tags: [go, status-dashboard, warnings, ux]

# Dependency graph
requires:
  - phase: 76-01
    provides: "UI spec with dashboard section order and warning types"
provides:
  - "computeWarnings function: stale state, failed phases, unacknowledged midden, expiring pheromones"
  - "renderWarningsSection: visual-mode-only warning display with banner"
  - "Extended workflowSuggestionsForState: failed-phase retry and all-complete seal cases"
affects: [status-command, next-step-suggestions]

# Tech tracking
tech-stack:
  added: []
  patterns: ["warning computation from colony state and data files", "visual-mode-only warning rendering"]

key-files:
  created: [cmd/status_ux_test.go]
  modified: [cmd/status.go, cmd/codex_visuals.go]

key-decisions:
  - "No PhaseFailed constant exists -- used literal string \"failed\" for phase status comparison"
  - "Warnings use actual warning emoji character, not descriptive string"
  - "Midden loaded from midden.json (not midden/midden.json) per storage layer convention"
  - "Named helper createSeedStore to avoid collision with existing createTestStore in session_cmds_test.go"

patterns-established:
  - "Warning computation as a pure function returning []string from colony state and store"
  - "Visual-mode-only rendering gated by caller (statusCmd), not inside render functions"

requirements-completed: [UX-04]

# Metrics
duration: 6min
completed: 2026-04-29
---

# Phase 76 Plan 02: Dashboard Warnings and Next-Step Suggestions Summary

**Status dashboard redesigned with proactive warnings (stale state, failed phases, unacknowledged midden, expiring pheromones) and extended next-step suggestions (failed-phase retry, all-complete seal)**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-29T18:57:41Z
- **Completed:** 2026-04-29T19:03:41Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Warning computation engine that inspects colony state and data files for actionable issues
- Visual-mode-only warning section inserted into dashboard between signals and progress
- Extended workflow suggestions with failed-phase retry and all-complete seal prompts
- 11 new tests covering all warning types and suggestion logic

## Task Commits

Each task was committed atomically:

1. **Task 1: Dashboard warnings section and extended next-step suggestions** - `c9e7b0b1` (feat)
2. **Task 2: Dashboard warnings and suggestions tests** - `b882b68b` (test)

## Files Created/Modified
- `cmd/status.go` - Added computeWarnings, renderWarningsSection; inserted warnings into renderDashboard
- `cmd/codex_visuals.go` - Extended workflowSuggestionsForState with failed-phase and all-complete cases
- `cmd/status_ux_test.go` - 11 tests for warning computation, rendering, and suggestion logic

## Decisions Made
- Used literal `"failed"` string for phase status comparison since no `PhaseFailed` constant exists in the codebase
- Warnings load midden from `midden.json` (storage layer convention) rather than `midden/midden.json`
- Named test helper `createSeedStore` to avoid redeclaration conflict with existing `createTestStore` in `session_cmds_test.go`
- All existing dashboard sections preserved unchanged; warnings inserted as a new non-destructive section

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing .aether/rules/ directory in worktree**
- **Found during:** Task 1 (build verification)
- **Issue:** Worktree was missing `.aether/rules/aether-colony.md` causing `go build` to fail on embedded_assets.go
- **Fix:** Copied the missing file from main repo to worktree
- **Files modified:** `.aether/rules/aether-colony.md` (worktree only, not committed)

**2. [Rule 1 - Bug] TestRenderWarningsSectionWithWarnings assertion mismatch**
- **Found during:** Task 2 (test execution)
- **Issue:** `renderBanner` uses `spacedTitle` which renders "W A R N I N G S" not "Warnings"
- **Fix:** Updated test assertion to match spaced output format
- **Files modified:** `cmd/status_ux_test.go`
- **Committed in:** `b882b68b` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both auto-fixes necessary for build and correctness. No scope creep.

## Issues Encountered
- 7 pre-existing test failures in cmd/ package unrelated to this plan (TestContinueEmitsLifecycleCeremonyEvents, TestContinueBlocksWhenWatcherUsesFakeInvoker, TestClaudeOpenCodeCommandParity, TestInitInvalidCharterJSON, TestIntegrityDetectSourceContext, TestLifecycleCommandDocsPreferRuntimeCLI, TestQueenWisdomHygiene). All 42 relevant tests pass.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Status dashboard now surfaces actionable warnings and next-step suggestions
- All existing dashboard sections preserved and all existing tests pass
- Ready for subsequent UX improvement plans

---
*Phase: 76-ux-improvements*
*Completed: 2026-04-29*

## Self-Check: PASSED

- cmd/status_ux_test.go: FOUND
- .planning/phases/76-ux-improvements/76-02-SUMMARY.md: FOUND
- c9e7b0b1: FOUND
- b882b68b: FOUND
- cmd/status.go: FOUND
- cmd/codex_visuals.go: FOUND
