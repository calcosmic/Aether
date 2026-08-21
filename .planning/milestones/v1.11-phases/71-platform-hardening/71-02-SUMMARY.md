---
phase: 71-platform-hardening
plan: 02
subsystem: cli
tags: [cobra, cli-audit, smoke-test, subcommand-registration, flag-coverage]

# Dependency graph
requires:
  - phase: 71-01
    provides: "worker cleanup, process group management base"
provides:
  - "Systematic CLI flag audit test (TestCLIFlagAudit) proving markdown-to-Go flag coverage"
  - "5 missing subcommands registered: suggest-approve, versions, chamber-compare, council parent, flag-create alias"
  - "8 flag gaps fixed across state-mutate, flag-list, midden-cross-pr-analysis, pheromone-merge-back, pheromone-expire, learning-approve-proposals, continue"
  - "Smoke test covering all 316 registered subcommands with --help validation (PLAT-05)"
affects: [72+, all plans that add CLI commands or markdown wrappers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CLI audit pattern: regex-extract markdown calls, compare against Go Cobra registrations"
    - "Smoke test pattern: iterate rootCmd.Commands() with --help, check exit code and non-empty output"

key-files:
  created:
    - cmd/cli_flag_audit_test.go
    - cmd/smoke_test.go
  modified:
    - cmd/compatibility_cmds.go
    - cmd/council.go
    - cmd/chamber.go
    - cmd/flag_cmds.go
    - cmd/flags.go
    - cmd/learning_cmds.go
    - cmd/midden_cmds.go
    - cmd/pheromone_mgmt.go
    - cmd/pheromone_write.go
    - cmd/state_cmds.go
    - cmd/codex_workflow_cmds.go

key-decisions:
  - "Council parent command added without removing direct subcommand registration (backward compatible: both 'aether council-deliberate' and 'aether council deliberate' work)"
  - "Flag-create implemented as alias of flag-add (not separate command) since they have identical semantics"
  - "Audit regex requires at least one --flag after subcommand name to avoid false positives from prose like 'older aether versions'"
  - "Smoke test sets rootCmd.SetOut/SetErr to capture Cobra help output, not just global stdout/stderr"

patterns-established:
  - "CLI audit test: systematic markdown-to-Go flag coverage verification"
  - "Smoke test: all subcommands respond to --help without error"

requirements-completed: [PLAT-04, PLAT-05]

# Metrics
duration: 12min
completed: 2026-04-28
---

# Phase 71 Plan 02: CLI Flag Audit and Smoke Test Summary

**Systematic audit proves 108 markdown CLI calls match Go registrations; 5 missing subcommands registered, 8 flag gaps fixed, smoke test covers all 316 subcommands**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-28T16:43:50Z
- **Completed:** 2026-04-28T16:56:25Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Created systematic CLI flag audit (TestCLIFlagAudit) that scans all markdown CLI calls and verifies each subcommand+flag combination exists in Go runtime -- resolves RESEARCH.md open questions with evidence
- Registered 5 missing subcommands: suggest-approve, versions, chamber-compare, council (parent), flag-create (alias)
- Fixed 8 flag gaps discovered by audit: state-mutate --verify-only/--revert, flag-list --phase, midden-cross-pr-analysis --window, pheromone-merge-back --export-file, pheromone-expire --phase-end-only, learning-approve-proposals --deferred/--verbose, continue --synthetic
- Delivered PLAT-05 smoke test validating all 316 registered subcommands respond to --help

## Task Commits

Each task was committed atomically:

1. **Task 1: Systematic CLI flag audit and missing subcommand registration** - `7b3ea4fa` (feat)
2. **Task 2: Create CLI smoke test (PLAT-05)** - `2923322b` (test)

## Files Created/Modified
- `cmd/cli_flag_audit_test.go` - Systematic audit: extracts CLI calls from markdown, compares against Go registrations, reports mismatches
- `cmd/smoke_test.go` - Smoke test: validates all 316 subcommands respond to --help; validates new subcommands with flags
- `cmd/compatibility_cmds.go` - Added suggest-approve and versions subcommands
- `cmd/council.go` - Added council parent command with backward-compatible direct subcommand registration
- `cmd/chamber.go` - Added chamber-compare subcommand
- `cmd/flag_cmds.go` - Added flag-create alias to flag-add
- `cmd/flags.go` - Added --phase filter to flag-list
- `cmd/learning_cmds.go` - Added --deferred and --verbose flags to learning-approve-proposals
- `cmd/midden_cmds.go` - Added --window flag to midden-cross-pr-analysis
- `cmd/pheromone_mgmt.go` - Added --export-file flag to pheromone-merge-back
- `cmd/pheromone_write.go` - Added --phase-end-only flag to pheromone-expire
- `cmd/state_cmds.go` - Added --verify-only and --revert flags to state-mutate
- `cmd/codex_workflow_cmds.go` - Added --synthetic flag to continue

## Decisions Made
- Council parent command added without removing direct subcommand registration -- both `aether council-deliberate` and `aether council deliberate` work, maintaining backward compatibility
- Flag-create implemented as alias of flag-add since they have identical semantics
- Audit regex requires at least one --flag after subcommand name to avoid false positives from prose mentioning "aether versions" in comments
- Smoke test uses `rootCmd.SetOut(&buf)` in addition to global stdout/stderr to capture Cobra help output correctly

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed store.ReadColonyState undefined error in suggest-approve**
- **Found during:** Task 1 (suggest-approve subcommand registration)
- **Issue:** Plan specified `store.ReadColonyState()` but `storage.Store` has no such method
- **Fix:** Changed to `loadActiveColonyState()` which is the existing pattern for reading colony state
- **Files modified:** cmd/compatibility_cmds.go
- **Verification:** Build compiles, test passes
- **Committed in:** 7b3ea4fa (Task 1 commit)

**2. [Rule 2 - Missing Critical] Fixed smoke test output capture for Cobra --help**
- **Found during:** Task 2 (smoke test creation)
- **Issue:** Plan's smoke test only set global `stdout`/`stderr` vars, but Cobra routes --help output through `rootCmd.SetOut()`. All 316 subcommands showed "produced no output" despite help text being printed
- **Fix:** Added `rootCmd.SetOut(&buf)` and `rootCmd.SetErr(&buf)` to smoke test alongside global variable assignment
- **Files modified:** cmd/smoke_test.go
- **Verification:** All 316 subcommands pass smoke test
- **Committed in:** 2923322b (Task 2 commit)

**3. [Rule 3 - Blocking] Created missing .aether/rules/ directory for build**
- **Found during:** Task 1 (initial build attempt)
- **Issue:** Worktree was missing `.aether/rules/` directory, causing `go build` to fail with "pattern all:.aether/rules: no matching files found"
- **Fix:** Created `.aether/rules/.gitkeep` to satisfy Go embed directive
- **Files modified:** .aether/rules/.gitkeep (generated, not committed)
- **Verification:** Build succeeds

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical), 1 blocking fix
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- Worktree missing `.aether/rules/` directory (build embed failure) -- fixed by creating directory with placeholder
- Pre-existing test failures (TestQueenWisdomHygiene, TestContinueEmitsLifecycleCeremonyEvents, TestContinueBlocksWhenWatcherUsesFakeInvoker) confirmed on base commit -- out of scope, not caused by this plan

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- PLAT-04 and PLAT-05 fully delivered
- Audit test provides ongoing regression guard for CLI flag coverage
- Smoke test provides ongoing regression guard for subcommand registration
- RESEARCH.md open questions resolved with audit evidence

---
*Phase: 71-platform-hardening*
*Completed: 2026-04-28*

## Self-Check: PASSED

- FOUND: cmd/cli_flag_audit_test.go
- FOUND: cmd/smoke_test.go
- FOUND: .planning/phases/71-platform-hardening/71-02-SUMMARY.md
- FOUND: 7b3ea4fa (Task 1 commit)
- FOUND: 2923322b (Task 2 commit)
