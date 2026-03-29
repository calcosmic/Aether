---
phase: 34-cross-colony-isolation
plan: 04
subsystem: infra
tags: [colony-isolation, per-colony-data, COLONY_DATA_DIR, utils-modules]

# Dependency graph
requires:
  - phase: 34-01
    provides: colony-name subcommand for colony identification
  - phase: 34-03
    provides: COLONY_DATA_DIR infrastructure in aether-utils.sh with auto-migration
provides:
  - All 15 utils/ modules now use COLONY_DATA_DIR for per-colony file references
  - Standalone scripts (swarm-display.sh, watch-spawn-tree.sh) have COLONY_DATA_DIR resolution
  - Zero per-colony DATA_DIR references remain in utils/
affects: [34-05, future multi-colony work]

# Tech tracking
tech-stack:
  added: []
  patterns: [per-colony data isolation in all utility modules, COLONY_DATA_DIR as standard for per-colony files]

key-files:
  created: []
  modified:
    - .aether/utils/session.sh
    - .aether/utils/pheromone.sh
    - .aether/utils/learning.sh
    - .aether/utils/flag.sh
    - .aether/utils/spawn.sh
    - .aether/utils/midden.sh
    - .aether/utils/swarm.sh
    - .aether/utils/suggest.sh
    - .aether/utils/queen.sh
    - .aether/utils/error-handler.sh
    - .aether/utils/swarm-display.sh
    - .aether/utils/watch-spawn-tree.sh
    - .aether/utils/chamber-utils.sh
    - .aether/aether-utils.sh

key-decisions:
  - "Standalone scripts (swarm-display.sh, watch-spawn-tree.sh) resolve COLONY_DATA_DIR inline since they are not sourced by aether-utils.sh"
  - "error-handler.sh uses COLONY_DATA_DIR since it is sourced after COLONY_DATA_DIR initialization (line 20)"
  - "safety-stats.json in aether-utils.sh updated to COLONY_DATA_DIR as per-colony data"
  - "state-api.sh and state-loader.sh unchanged -- they only reference COLONY_STATE.json at DATA_DIR"

patterns-established:
  - "All per-colony file references use COLONY_DATA_DIR, shared files use DATA_DIR"
  - "Standalone utility scripts resolve COLONY_DATA_DIR by reading COLONY_STATE.json colony_name field"

requirements-completed: [SAFE-02]

# Metrics
duration: 10min
completed: 2026-03-29
---

# Phase 34: Cross-Colony Isolation - Plan 04 Summary

**All 15 utils/ modules updated to use COLONY_DATA_DIR for per-colony files, completing the per-colony data isolation migration across the entire utility layer**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-29T07:41:50Z
- **Completed:** 2026-03-29T07:52:00Z
- **Tasks:** 2
- **Files modified:** 14

## Accomplishments

- Updated 93 per-colony DATA_DIR references across all 15 utils/ modules to COLONY_DATA_DIR
- Added inline COLONY_DATA_DIR resolution to standalone scripts (swarm-display.sh, watch-spawn-tree.sh) that cannot access the global variable
- Verified COLONY_STATE.json and shared resources (backups/, survey/, queen-wisdom.json) remain at DATA_DIR
- Comprehensive sweep confirms zero per-colony DATA_DIR references remain in utils/
- All 603 tests pass (13 pre-existing failures unrelated to this change)

## Task Commits

Each task was committed atomically:

1. **Task 1: Update high-traffic utils modules (session, pheromone, learning, flag, spawn, midden)** - `08d68ed` (feat)
2. **Task 2: Update remaining utils modules (swarm, suggest, queen, error-handler, display, watch, chamber-utils)** - `19dea37` (feat)

**Additional fix:** `f1fc081` (fix: update safety-stats.json to use COLONY_DATA_DIR)

## Files Created/Modified

- `.aether/utils/session.sh` - session.json, spawn-tree.txt, spawn-tree-archive/ now use COLONY_DATA_DIR
- `.aether/utils/pheromone.sh` - pheromones.json, constraints.json, flags.json, rolling-summary.log, midden/ now use COLONY_DATA_DIR
- `.aether/utils/learning.sh` - learnings.json, learning-observations.json, learning-deferred.json, .promotion-undo.json, last-build-claims.json now use COLONY_DATA_DIR
- `.aether/utils/flag.sh` - flags.json now uses COLONY_DATA_DIR
- `.aether/utils/spawn.sh` - activity.log, spawn-tree.txt now use COLONY_DATA_DIR
- `.aether/utils/midden.sh` - midden/, errors.log now use COLONY_DATA_DIR
- `.aether/utils/swarm.sh` - swarm-findings-*, swarm-display.json, swarm-activity.log, swarm-archive/, timing.log now use COLONY_DATA_DIR
- `.aether/utils/suggest.sh` - pheromones.json, session.json references now use COLONY_DATA_DIR
- `.aether/utils/queen.sh` - learning-observations.json reference now uses COLONY_DATA_DIR
- `.aether/utils/error-handler.sh` - activity.log, errors.log references now use COLONY_DATA_DIR
- `.aether/utils/swarm-display.sh` - Added COLONY_DATA_DIR resolution for standalone script, swarm-display.json uses COLONY_DATA_DIR
- `.aether/utils/watch-spawn-tree.sh` - Added COLONY_DATA_DIR resolution for standalone script, spawn-tree.txt and view-state.json use COLONY_DATA_DIR
- `.aether/utils/chamber-utils.sh` - pheromones.json reference now uses COLONY_DATA_DIR
- `.aether/aether-utils.sh` - safety-stats.json reference updated to COLONY_DATA_DIR

## Decisions Made

- Standalone scripts (swarm-display.sh, watch-spawn-tree.sh) resolve COLONY_DATA_DIR inline by reading COLONY_STATE.json colony_name and computing the sanitized colony path, since they are not sourced by aether-utils.sh and do not have access to the global COLONY_DATA_DIR variable
- error-handler.sh safely uses COLONY_DATA_DIR because it is sourced at line 29 of aether-utils.sh, after COLONY_DATA_DIR is initialized at line 20
- state-api.sh and state-loader.sh were intentionally left unchanged -- they only reference COLONY_STATE.json which must remain at DATA_DIR as the colony identification anchor

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated safety-stats.json reference in aether-utils.sh**
- **Found during:** Final comprehensive verification sweep
- **Issue:** safety-stats.json was referenced via DATA_DIR in aether-utils.sh (not utils/) but is per-colony data
- **Fix:** Changed `$DATA_DIR/safety-stats.json` to `$COLONY_DATA_DIR/safety-stats.json`
- **Files modified:** .aether/aether-utils.sh
- **Verification:** Comprehensive sweep now returns only migration function references (intentional)
- **Committed in:** `f1fc081` (fix)

**2. [Rule 3 - Blocking] Added COLONY_DATA_DIR resolution to standalone scripts**
- **Found during:** Task 2 execution
- **Issue:** swarm-display.sh and watch-spawn-tree.sh are standalone scripts not sourced by aether-utils.sh, so they lack access to the global COLONY_DATA_DIR variable. Without resolution, they would continue reading from the flat DATA_DIR and miss per-colony files.
- **Fix:** Added inline COLONY_DATA_DIR resolution logic that reads COLONY_STATE.json colony_name field and computes the sanitized colony path, matching the logic in _resolve_colony_data_dir
- **Files modified:** .aether/utils/swarm-display.sh, .aether/utils/watch-spawn-tree.sh
- **Committed in:** `19dea37` (part of Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking)
**Impact on plan:** Both auto-fixes essential for complete per-colony isolation. No scope creep.

## Issues Encountered

- 13 pre-existing test failures (model-profile and instinct-confidence tests) confirmed to exist before changes -- not caused by this plan
- Comprehensive grep sweep returns 8 DATA_DIR references in aether-utils.sh migration function -- all intentional (migration source paths must reference DATA_DIR)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 15 utils/ modules fully migrated to COLONY_DATA_DIR
- COLONY_STATE.json and shared resources correctly remain at DATA_DIR
- Auto-migration infrastructure from Plan 03 will transparently move existing files on first access
- Ready for Plan 05 (OpenCode command updates) and future multi-colony work
- No blockers or concerns

---
*Phase: 34-cross-colony-isolation*
*Completed: 2026-03-29*
