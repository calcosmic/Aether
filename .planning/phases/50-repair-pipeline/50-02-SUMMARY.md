---
phase: 50-repair-pipeline
plan: 02
subsystem: repair
tags: [cobra, cli, recover, repair, stuck-state, rendering]

# Dependency graph
requires:
  - phase: 50-01
    provides: "performRecoverRepairs, isDestructiveCategory, repair handler functions"
provides:
  - "runRecover --apply wiring: scan -> repair -> re-scan flow"
  - "renderRecoverDiagnosis with Repair Log stage"
  - "renderRepairLog with OK/FAILED per-repair status"
  - "Destructive category hint: 'Needs confirmation with --apply'"
  - "renderRecoverJSON repairs object"
  - "10 integration tests for repair wiring and visual output"
affects: [recover-command, medic-repair-pattern]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "scan-repair-rescan: repair orchestrator pattern matching medic's runMedic flow"

key-files:
  created: []
  modified:
    - cmd/recover.go
    - cmd/recover_visuals.go
    - cmd/recover_test.go

key-decisions:
  - "Destructive categories (dirty_worktree, bad_manifest) show 'Needs confirmation with --apply' instead of 'Fixable with --apply'"
  - "repairResult passed as optional pointer to render functions, nil when no --apply"

patterns-established:
  - "Repair log stage rendered after Summary when repairs were performed"
  - "JSON output includes repairs object only when repairResult is non-nil"

requirements-completed: [REPAIR-01, REPAIR-02, REPAIR-03, REPAIR-04, REPAIR-05]

# Metrics
duration: 4min
completed: 2025-04-25
---

# Phase 50 Plan 02: Wire --apply and Visual Output Summary

**Repair wiring connects scan-repair-rescan flow, destructive categories show confirmation hint, repair log renders OK/FAILED per action**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-25T16:08:13Z
- **Completed:** 2026-04-25T16:12:39Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- `runRecover` calls `performRecoverRepairs` when `--apply` is set and issues exist, then re-scans for post-repair state
- `renderRecoverDiagnosis` shows a Repair Log stage with per-repair OK/FAILED status when repairs were performed
- `writeRecoverIssueLine` distinguishes destructive categories (dirty_worktree, bad_manifest) with "Needs confirmation with --apply" vs safe categories with "Fixable with --apply"
- `renderRecoverJSON` includes a `repairs` object with attempted/succeeded/failed/skipped/details when repairResult is non-nil
- `recoverNextStep` mentions `--force` for destructive category next steps
- 10 new integration tests verify all wiring and visual output

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire --apply in runRecover and update visuals for repair output** - `56bc0ee3` (fix)
2. **Task 2: Integration tests for repair wiring, re-scan, and visual output** - `75dddad9` (test)

## Files Created/Modified
- `cmd/recover.go` - --apply wiring: calls performRecoverRepairs, re-scans after repair, passes repairResult to render functions
- `cmd/recover_visuals.go` - Repair Log stage rendering, renderRepairLog function, destructive category hints, JSON repairs object
- `cmd/recover_test.go` - Fixed 5 broken test calls (signature change), added 10 new integration tests

## Decisions Made
- Destructive categories (dirty_worktree, bad_manifest) use "Needs confirmation with --apply" to warn users that these repairs may lose data
- repairResult passed as optional `*RepairResult` to render functions -- nil when no --apply, populated when repairs ran
- Followed the medic pattern (scan -> repair -> re-scan) for consistency across the repair pipeline

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing tests had stale function signatures**
- **Found during:** Task 1 (Wire --apply in runRecover)
- **Issue:** `renderRecoverDiagnosis` and `renderRecoverJSON` signatures were updated to accept `*RepairResult` parameter in prior work, but 5 existing test calls were not updated, causing compilation failure
- **Fix:** Updated 5 test calls to pass `nil` as the repairResult argument (3 for renderRecoverDiagnosis, 2 for renderRecoverJSON)
- **Files modified:** cmd/recover_test.go
- **Verification:** `go test ./cmd/ -run "TestRenderRecoverDiagnosis|TestRenderRecoverJSON"` passes
- **Committed in:** `56bc0ee3` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** The signature fix was required for the tests to compile. The implementation changes (wiring, visuals) were already in place from prior work, so this plan primarily delivered the test coverage and the signature fix.

## Issues Encountered
- The implementation changes described in the plan (wiring --apply, repair log rendering, destructive hints, JSON repairs) were already present in the codebase. This suggests partial prior execution or the changes were committed alongside Plan 01. The remaining work was fixing broken tests and adding comprehensive integration tests.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Repair pipeline is fully wired: scan -> repair -> re-scan -> render
- All visual output (diagnosis, repair log, JSON) handles repair results correctly
- Destructive vs safe category distinction is properly communicated to users
- Ready for any follow-up work on the recover command or repair pipeline

---
*Phase: 50-repair-pipeline*
*Completed: 2025-04-25*
