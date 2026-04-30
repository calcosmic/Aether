---
phase: 81-plan-and-lifecycle-loop-safety
plan: 02
subsystem: ux
tags: [go, recovery, error-handling, cli, lifecycle]

# Dependency graph
requires: []
provides:
  - "RecoveryEngine with error classification, command exclusion, and menu rendering"
  - "LOOP-05 guarantee: lifecycle commands never suggest re-running themselves"
  - "JSON mode recovery_options in error envelope"
affects: [82-continue-experience, future-lifecycle-refactor]

# Tech tracking
tech-stack:
  added: []
  patterns: [recovery-engine, command-exclusion, error-classification, display-only-menu]

key-files:
  created:
    - cmd/recovery_engine.go
    - cmd/recovery_engine_test.go
  modified:
    - cmd/codex_workflow_cmds.go
    - cmd/entomb_cmd.go
    - cmd/status.go
    - cmd/session_flow_cmds.go
    - cmd/status_test.go

key-decisions:
  - "Recovery menu is display-only (no stdin reading) per research open question 2 and threat T-81-03"
  - "Error classification uses case-insensitive substring matching consistent with existing friendlyError pattern"
  - "Minimum 2 recovery options guaranteed by supplementing from genericFallback"
  - "Status command error path fixed to write to stderr (was incorrectly writing to stdout)"

patterns-established:
  - "Recovery engine pattern: classify error, look up command-specific candidates, filter failed command, guarantee minimum options"
  - "JSON mode includes recovery_options array in error envelope details"

requirements-completed: [LOOP-05]

# Metrics
duration: 16min
completed: 2026-04-30
---

# Phase 81 Plan 02: Recovery Engine for Lifecycle Commands Summary

**Recovery engine with command exclusion (LOOP-05) that prevents infinite retry loops by never suggesting the failed lifecycle command, with visual banner rendering and JSON mode support**

## Performance

- **Duration:** 16 min
- **Started:** 2026-04-30T15:56:54Z
- **Completed:** 2026-04-30T16:12:32Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Recovery engine module with error classification, command normalization, and candidate filtering
- All 4 lifecycle commands (seal, entomb, status, resume) now show actionable recovery menus instead of bare errors
- LOOP-05 guarantee enforced: the failed command is never suggested as a recovery option
- JSON mode consumers get `recovery_options` array in the error envelope details
- 11 unit tests covering exclusion, normalization, classification, rendering, and minimum option guarantees

## Task Commits

Each task was committed atomically:

1. **Task 1: TDD recovery engine (RED)** - `71d952a` (test)
2. **Task 1: TDD recovery engine (GREEN)** - `60cea57` (feat)
3. **Task 2: Wire recovery engine into lifecycle commands** - `2c1bdcf` (feat)

## Files Created/Modified
- `cmd/recovery_engine.go` - RecoveryOption struct, normalizeBaseCommand, classifyError, recoveryOptionsForCommand, renderRecoveryMenu, buildVisualRecoveryMenu
- `cmd/recovery_engine_test.go` - 11 tests: exclusion, normalization, classification, rendering, expected options, minimum options, flag variants
- `cmd/codex_workflow_cmds.go` - Seal command error paths now use renderRecoveryMenu (4 call sites)
- `cmd/entomb_cmd.go` - Entomb command error paths now use renderRecoveryMenu (4 call sites)
- `cmd/status.go` - Status fallback error path now uses renderRecoveryMenu (1 call site)
- `cmd/session_flow_cmds.go` - Resume-colony command error paths now use renderRecoveryMenu (2 call sites)
- `cmd/status_test.go` - Fixed TestStatusNoColony to capture stderr (recovery engine writes there)

## Decisions Made
- Recovery menu is display-only (no stdin reading) per research open question 2 -- user copies command manually, eliminating injection vector (threat T-81-03)
- Used case-insensitive substring matching for error classification, consistent with existing `friendlyErrorForPattern` pattern in `ux_friendly_errors.go`
- Generic fallback supplements command-specific candidates when filtering removes options below the minimum threshold of 2
- Status command error path was writing to stdout (wrong) -- fixed to stderr where all other error output goes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing .aether/rules/ directory in worktree**
- **Found during:** Task 1 (RED phase -- test compilation)
- **Issue:** `embedded_assets.go` has `//go:embed all:.aether/rules` but worktree was missing the directory
- **Fix:** Created `.aether/rules/` and copied `aether-colony.md` from main repo
- **Files modified:** `.aether/rules/aether-colony.md` (created in worktree only, gitignored)
- **Verification:** Tests compile and run
- **Committed in:** Part of `71d952a`

**2. [Rule 1 - Bug] TestRenderRecoveryMenu expected wrong banner format**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Test checked for "Recovery" but `renderBanner` uses `spacedTitle` which produces "R E C O V E R Y"
- **Fix:** Updated test assertion to match actual output format
- **Files modified:** `cmd/recovery_engine_test.go`
- **Verification:** Test passes

**3. [Rule 1 - Bug] TestRenderRecoveryMenu failed in non-terminal environments**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** `renderRecoveryMenu` delegates to JSON mode in non-terminal envs (returns empty string, writes to stderr). Test called `renderRecoveryMenu` directly.
- **Fix:** Changed test to call `buildVisualRecoveryMenu` directly for visual rendering assertions
- **Files modified:** `cmd/recovery_engine_test.go`
- **Verification:** Test passes in both terminal and non-terminal environments

**4. [Rule 1 - Bug] TestStatusNoColony captured stdout but recovery writes to stderr**
- **Found during:** Task 2 (wiring)
- **Issue:** Original status error path wrote to stdout via `fmt.Fprintln(stdout, ...)`. Recovery engine correctly writes to stderr. Existing test was checking stdout.
- **Fix:** Updated test to capture stderr instead of stdout
- **Files modified:** `cmd/status_test.go`
- **Verification:** TestStatusNoColony passes

**5. [Rule 2 - Missing Critical] Recovery candidates did not guarantee minimum 2 options**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Some error classes had only 1 candidate per command, and after filtering the failed command, the count dropped below 2
- **Fix:** Added minimum-2 guarantee in `recoveryOptionsForCommand` by supplementing from `genericFallback`
- **Files modified:** `cmd/recovery_engine.go`
- **Verification:** TestRecoveryMinimumOptions passes (24 subtests, all lifecycle commands x 6 error types)

---

**Total deviations:** 5 auto-fixed (1 blocking, 3 bugs, 1 missing critical)
**Impact on plan:** All auto-fixes necessary for correctness and test reliability. No scope creep.

## Issues Encountered
- Worktree missing `.aether/rules/` directory required by Go embed directive -- resolved by copying from main repo
- 2 pre-existing test failures (`TestIntegrityDetectSourceContext`, `TestQueenWisdomHygiene`) unrelated to this plan -- documented as out of scope

## Known Stubs

None.

## Next Phase Readiness
- Recovery engine is self-contained and ready for use by any command
- Future lifecycle commands can adopt `renderRecoveryMenu` by passing their command name and error message
- JSON consumers can parse `recovery_options` from error envelope details programmatically

---
*Phase: 81-plan-and-lifecycle-loop-safety*
*Completed: 2026-04-30*
