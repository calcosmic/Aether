---
phase: 91-hive-intelligence
plan: 03
subsystem: database
tags: [sqlite, curator, lifecycle, skills, usage-tracking, cli]

# Dependency graph
requires:
  - phase: 91-02
    provides: "SkillService, SkillMetadata, stage constants, skillDirForStage helper"
provides:
  - "Keeper Curator with lifecycle transitions (active->stale->archived at 14-day intervals)"
  - "Usage tracking (RecordSkillView, RecordSkillUse) with timestamp updates"
  - "Skill recovery (RecoverSkill) restoring archived skills to active"
  - "CLI commands: skill-curator-run, skill-recover"
affects: [91-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [collect-then-process-queries, curator-lifecycle-transition]

key-files:
  created:
    - pkg/learn/curator.go
    - pkg/learn/curator_test.go
    - pkg/learn/skills.go
    - cmd/skill_curator.go
  modified: []

key-decisions:
  - "Collect query results before processing to avoid deadlock with MaxOpenConns(1) on modernc.org/sqlite"
  - "Curator uses same *sql.DB as SkillService and SQLiteColonyStore (shared connection)"
  - "Created skills.go dependency from 91-02 plan since 91-02 has not yet merged"

patterns-established:
  - "Collect-then-process pattern for SQLite queries with single-connection constraint"
  - "Curator lifecycle: query eligible -> close rows -> process file moves + DB updates"

requirements-completed: [SKIL-04, SKIL-05, SKIL-06]

# Metrics
duration: 6min
completed: 2026-05-02
---

# Phase 91 Plan 03: Keeper Curator Summary

**Keeper Curator transitions skills through active -> stale -> archived lifecycle with 14-day thresholds, usage tracking (view/use counts with timestamps), pinned skill immunity, and archived skill recovery**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-02T11:36:08Z
- **Completed:** 2026-05-02T11:42:33Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Curator transitions unused skills through lifecycle stages: active (14 days unused) -> stale (28 days unused) -> archived
- Pinned skills are immune to all automatic transitions (D-09, SKIL-05)
- Usage tracking increments view_count/use_count and updates timestamps, resetting the transition clock
- Archived skills are recoverable via RecoverSkill, restoring to active with reset timestamps (SKIL-06)
- CLI commands `skill-curator-run` and `skill-recover` registered
- 11 tests covering all transitions, pin immunity, file moves, recovery, usage tracking

## Task Commits

Each task was committed atomically:

1. **Task 1: Keeper Curator with lifecycle transitions, usage tracking, and pin immunity** - `38601772` (feat)
2. **Task 2: CLI command for curator and cache invalidation** - `aaf816f9` (feat)

## Files Created/Modified
- `pkg/learn/skills.go` - SkillService CRUD, SkillMetadata, stage constants, skillDirForStage, progressive disclosure (dependency from 91-02)
- `pkg/learn/curator.go` - Curator with RunTransitions, RecoverSkill, RecordSkillView/Use, pin immunity via SQL WHERE clause
- `pkg/learn/curator_test.go` - 11 tests: active->stale, stale->archived, skip recent, pin immunity, file moves, recovery, empty table, view/use tracking, usage resets transition, transition count
- `cmd/skill_curator.go` - CLI commands: skill-curator-run runs transitions, skill-recover restores archived skills; resolveColonyDBPath/resolveSkillBaseDir helpers

## Decisions Made
- Collect query results before processing file moves and DB updates to avoid deadlock with MaxOpenConns(1) on modernc.org/sqlite -- the pure Go SQLite driver cannot interleave query and exec on a single connection
- Curator uses the same `*sql.DB` as SkillService and SQLiteColonyStore, sharing the single-connection constraint
- Created `skills.go` from 91-02 plan's provided code since 91-02 has not yet merged into the worktree

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed deadlock in transitionStage with MaxOpenConns(1)**
- **Found during:** Task 1 (GREEN phase -- first test run timed out at 60s)
- **Issue:** Plan's `transitionStage` held an open `rows` cursor while calling `c.db.Exec()` inside the loop. With `MaxOpenConns(1)`, the single connection was occupied by the query, causing the Exec to block indefinitely. modernc.org/sqlite cannot interleave query and exec on the same connection.
- **Fix:** Refactored `transitionStage` to collect all eligible skills into a `[]transitionSkill` slice, close `rows` to release the connection, then iterate the slice for file moves and DB updates. Added a `transitionSkill` struct for collected data.
- **Files modified:** pkg/learn/curator.go
- **Verification:** All 11 tests pass in 0.5s (previously timed out at 60s)
- **Committed in:** 38601772 (Task 1 commit)

**2. [Rule 3 - Blocking] Created skills.go dependency from 91-02 plan**
- **Found during:** Task 1 (prerequisite -- Curator depends on SkillMetadata, SkillService, stage constants, skillDirForStage)
- **Issue:** Plan 91-02 (Skill Service) has not yet merged, so `pkg/learn/skills.go` does not exist in the worktree. The Curator cannot compile without these types.
- **Fix:** Created `pkg/learn/skills.go` using the full code from 91-02 plan's `<action>` section. This includes SkillMetadata, SkillService, SkillIndexEntry, SkillEvidenceFrontmatter, skillDirForStage, validateSkillName, and all CRUD operations.
- **Files created:** pkg/learn/skills.go
- **Verification:** `go test ./pkg/learn/...` passes; `go vet ./pkg/learn/...` clean
- **Committed in:** 38601772 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness and compilation. No scope creep. skills.go may need reconciliation when 91-02 merges.

## Issues Encountered
- `go build ./cmd/aether` fails with pre-existing `embedded_assets.go` pattern error (`.aether/rules: no matching files found`). This is unrelated to our changes and was present before this plan's base commit. Verified by checking build on clean base.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Keeper Curator is complete and tested with 11 passing tests
- Curator.RunTransitions() ready for integration into continue/seal lifecycle hooks
- RecordSkillView/RecordSkillUse ready for wiring into skill-match and skill-inject flows
- skills.go was created from 91-02 plan code -- may need reconciliation when 91-02 merges

## Self-Check: PASSED

- All 4 key files exist (curator.go, curator_test.go, skills.go, skill_curator.go)
- All 3 commits verified (38601772, aaf816f9, 653aacb3)
- All 11 curator tests pass
- go vet ./pkg/learn/... clean

---
*Phase: 91-hive-intelligence*
*Completed: 2026-05-02*
