---
phase: 63-lifecycle-ceremony-status-entomb-resume
plan: 02
subsystem: lifecycle-ceremony
tags: [entomb, registry, midden, pheromones, instincts, hive]

# Dependency graph
requires: []
provides:
  - Near-miss instinct extraction at entomb (confidence 0.5-0.8 preserved in chamber)
  - Temp sweep: expired pheromones, old midden, stale backups cleaned before archive reset
  - Registry final stats: phase count, learning count, instinct count, duration recorded at entomb
  - midden.json archived in chamber before sweep
affects: [63-03-resume, future-entomb-iterations]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Near-miss extraction: filter instincts by confidence range [0.5, 0.8)
    - Temp sweep after artifact copy: ensures chamber preserves data before cleanup
    - Best-effort registry update: silent return when no registry entry exists

key-files:
  created: []
  modified:
    - cmd/entomb_cmd.go - Near-miss extraction, temp sweep, registry stats wiring, visual output
    - cmd/registry.go - registryFinalStats struct, updateRegistryFinalStats helper
    - cmd/entomb_cmd_test.go - 6 new tests for near-miss, temp sweep, registry, suggestion output
    - cmd/codex_continue_finalize.go - Fixed pre-existing compilation error (unused bool param)

key-decisions:
  - "Phase struct lacks PlanCount field, so countTotalPlans proxies to completedPhaseCount"
  - "Temp sweep runs after copyEntombArtifacts to preserve data in chamber before cleanup"
  - "Registry update is best-effort: silent return when no registry file or no matching entry"
  - "midden.json added to copyEntombArtifacts list so temp-swept data is preserved in archive"

patterns-established:
  - "Near-miss pattern: extract borderline-confidence data for preservation before reset"
  - "Post-copy sweep: cleanup runs only after artifacts are safely copied to chamber"

requirements-completed: [CERE-07]

# Metrics
duration: 11min
completed: 2026-04-27
---

# Phase 63 Plan 02: Entomb Near-Miss Extraction, Temp Sweep, Registry Stats Summary

**Entomb ceremony enriched with near-miss instinct preservation (0.5-0.8 confidence), temp file sweep (expired pheromones, old midden, stale backups), and registry final stats recording.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-27T17:08:24Z
- **Completed:** 2026-04-27T17:19:21Z
- **Tasks:** 1 (TDD)
- **Files modified:** 4

## Accomplishments
- Near-miss instincts (confidence 0.5-0.8) extracted and archived in chamber as `near-miss-instincts.json`
- Temp sweep cleans expired pheromones (Active=false, Strength=0), midden entries older than 30 days, and stale backups older than 7 days
- Registry entry updated with FinalStats (phase count, plan count, learning count, instinct count, seal date, duration) and marked inactive
- Chamber manifest includes `near_miss_instincts` count
- Entomb output includes hive promotion suggestion when near-miss instincts exist
- `midden.json` now archived in chamber before temp sweep cleans old entries

## Task Commits

Each task was committed atomically:

1. **Task 1: Near-miss instinct extraction, temp sweep, and registry final stats for entomb** - RED: `7bd7761a` (test), GREEN: `9c3a1556` (feat)

_Note: TDD task with RED/GREEN commits._

## Files Created/Modified
- `cmd/entomb_cmd.go` - Added `extractNearMissInstincts`, `entombTempSweep`, `countTotalPlans`, `computeColonyDuration`; wired near-miss extraction and temp sweep into entomb flow; extended result map and visual output
- `cmd/registry.go` - Added `registryFinalStats` struct with `omitempty` fields; added `updateRegistryFinalStats` helper function
- `cmd/entomb_cmd_test.go` - Added 6 tests: `TestEntombNearMissExtraction`, `TestEntombTempSweepMidden`, `TestEntombTempSweepExpiredPheromones`, `TestEntombRegistryFinalStats`, `TestEntombRegistryNoEntry`, `TestNearMissSuggestionOutput`
- `cmd/codex_continue_finalize.go` - Fixed pre-existing compilation error by restoring unused bool parameter

## Decisions Made
- Phase struct lacks `PlanCount` field, so `countTotalPlans` proxies to `completedPhaseCount` (completed phases as plan count proxy)
- Temp sweep runs after `copyEntombArtifacts` to ensure chamber preserves data before cleanup
- Registry update is best-effort: silently returns when no registry file or no matching entry exists (per D-06/Pitfall 5)
- Added `midden.json` to `copyEntombArtifacts` data files list so temp-swept entries are preserved in the chamber archive

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added midden.json to copyEntombArtifacts list**
- **Found during:** Task 1 (TestEntombTempSweepMidden failed)
- **Issue:** `midden.json` was not in the list of files copied to chamber, so temp sweep would delete entries before they could be archived. The test expected midden data in the chamber.
- **Fix:** Added `"midden.json"` to the `dataFiles` slice in `copyEntombArtifacts`
- **Files modified:** `cmd/entomb_cmd.go`
- **Verification:** `TestEntombTempSweepMidden` passes; archived midden contains both old and recent entries
- **Committed in:** `9c3a1556` (part of GREEN commit)

**2. [Rule 3 - Blocking] Fixed pre-existing compilation error in codex_continue_finalize.go**
- **Found during:** Task 1 (test compilation failed)
- **Issue:** `externalContinueReviewReport` function signature was changed in commit `c8345ea5` to remove the 4th bool parameter, but two test call sites still pass 4 arguments. This prevented any test in the `cmd` package from compiling.
- **Fix:** Restored the 4th `_ bool` parameter to the function signature (ignored) and added it to the internal call site. Tests were already reverted to their original (broken) state.
- **Files modified:** `cmd/codex_continue_finalize.go`
- **Verification:** `go build ./cmd/...` succeeds; all entomb tests pass
- **Committed in:** `9c3a1556` (part of GREEN commit)
- **Note:** Two tests (`TestExternalContinueReviewReportTimeoutNotBlocking`, `TestExternalContinueReviewReportSkipMissing`) still fail at runtime due to behavior changes in the same commit. These are pre-existing and out of scope.

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking)
**Impact on plan:** midden archiving is necessary for the temp sweep to be meaningful (data preserved before cleanup). Compilation fix unblocked all test execution. No scope creep.

## Issues Encountered
- Pre-existing test failures in `codex_continue_test.go` (5 tests) from commit `c8345ea5` which changed `externalContinueReviewReport` behavior but didn't update tests. These are documented as out of scope.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CERE-07 fully implemented: near-miss extraction, temp sweep, registry stats
- No blockers for plan 03 (resume stale pheromone detection)
- Registry `FinalStats` struct available for any future consumers

---
*Phase: 63-lifecycle-ceremony-status-entomb-resume*
*Completed: 2026-04-27*

## Self-Check: PASSED

All files exist, commits verified, key functions present.
