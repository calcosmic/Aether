---
phase: 200-iterative-planning
plan: 43
subsystem: testing
tags: [go, pheromones, shelf, repository-containment, test-isolation]

# Dependency graph
requires:
  - phase: 200-41
    provides: repository-authorized command-test binding and authority-first cleanup
provides:
  - exact root and data restoration across pheromone command fixtures
  - contained entomb, init, and seal shelf fixtures
  - owned verification selectors that cannot sweep in unrelated future repairs
affects: [200-40, 200-44, final-gate, command-tests]

# Tech tracking
tech-stack:
  added: []
  patterns: [testing.T.Setenv exact restoration, repository-local data binding, scoped test selectors]

key-files:
  created: []
  modified:
    - cmd/pheromone_write_test.go
    - cmd/pheromones_test.go
    - cmd/shelf_entomb_test.go
    - cmd/shelf_init_test.go
    - cmd/shelf_seal_test.go
    - .planning/phases/200-iterative-planning/200-43-PLAN.md

key-decisions:
  - "Pheromone and shelf fixtures bind AETHER_ROOT and COLONY_DATA_DIR to the same temporary repository and let testing restore exact environment presence and values."
  - "Plan verification names only the five owned test files' cases; unrelated pheromone integration failures remain assigned to later repair plans."

patterns-established:
  - "Lifecycle fixture cleanup: pair repository-local root and data selectors with t.Setenv, while saveGlobals clears store and tracer authority."
  - "Repair-plan gates: anchor selectors to owned tests so a focused gate cannot silently run zero tests or absorb another plan's failures."

requirements-completed: [CEC-03, PLAN-06]

# Metrics
duration: 13min
completed: 2026-09-09
---

# Phase 200 Plan 43: Pheromone and Shelf Fixture Isolation Summary

**Pheromone and shelf lifecycle tests now restore repository authority exactly without changing signal policy, archive behavior, or production containment.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-09-09T09:19:19Z
- **Completed:** 2026-09-09T09:32:27Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Replaced 40 self-restoring `AETHER_ROOT` defers with testing-owned root and data bindings across the five owned fixture files.
- Added RED/GREEN regression proof that deleted pheromone and shelf repositories cannot remain in process-wide root, data, store, or tracer authority.
- Kept signal sanitization, deduplication, decay, shelf promotion/dismissal, candidate detection, and chamber-copy assertions unchanged.
- Corrected both task gates so they execute the owned tests twice instead of selecting unrelated tests or zero tests.

## Task Commits

Each task was committed atomically through its RED and GREEN TDD gates:

1. **Task 1 RED: Reproduce pheromone fixture root leakage** - `16a93070` (test)
2. **Task 1 GREEN: Isolate pheromone command fixtures** - `635993b4` (test)
3. **Task 2 RED: Reproduce shelf fixture root leakage** - `27d8b325` (test)
4. **Task 2 GREEN: Isolate shelf lifecycle fixtures** - `8bbf2548` (test)
5. **Authority probe cleanup correction** - `6ae0090e` (test)

**Plan metadata:** `0d82d390` (docs), followed by a scoped provenance-restoration commit.

## Files Created/Modified

- `cmd/pheromone_write_test.go` - Restores both repository selectors for every pheromone-write case.
- `cmd/pheromones_test.go` - Isolates reader/count fixtures and proves deleted repository authority cannot leak.
- `cmd/shelf_entomb_test.go` - Isolates chamber-copy fixtures and carries the shelf cleanup regression.
- `cmd/shelf_init_test.go` - Binds active-shelf, promotion, dismissal, and init fixtures to their owning repository.
- `cmd/shelf_seal_test.go` - Binds candidate-detection fixtures to their owning repository.
- `.planning/phases/200-iterative-planning/200-43-PLAN.md` - Narrows both automated selectors to the tests this plan owns.

## Verification

- `go test ./cmd -run '^Test(PheromoneWrite_.*|PheromoneWriteSourcePhase(NilWhenNoColony)?|PheromoneRead(Empty)?|PheromoneCount(Empty)?|PheromoneFixtureRestoresRepositoryAuthority200)$' -count=2` - PASS
- `go test ./cmd -run '^Test(CopyShelfToChamber(Missing)?|ShelfChamberSummary(Empty|AllPromoted)?|LoadActiveShelf(Empty)?|PromoteShelfEntry|DismissShelfEntry|ShelfEntryToTodo|FormatShelfForInit|InitShelfBacklogOutput|Detect(ExpiredFocus|LowConfidenceInstinct|UnresolvedFlag|RecurringRedirect|NoCandidates|Deduplicates))$' -count=2` - PASS
- `go test ./cmd -run '^Test(PheromoneFixtureRestoresRepositoryAuthority200|ShelfChamberSummaryEmpty|RepositoryTestBindingSequence200|HookPreToolUseBlocksProtectedPath)$' -count=2` - PASS
- `! rg -n 'defer os\.Setenv\("AETHER_ROOT", os\.Getenv\("AETHER_ROOT"\)\)' cmd/{pheromone_write,pheromones,shelf_entomb,shelf_init,shelf_seal}_test.go` - PASS

## Decisions Made

- Used exact `testing.T.Setenv` cleanup rather than relaxing production containment or teaching commands to tolerate a deleted repository.
- Kept each fixture's existing store and domain assertions intact; only process-wide root/data ownership changed.
- Treated the original broad Task 1 selector's failures as separately owned repair work and corrected the plan instead of editing those files.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Replaced invalid broad verification selectors**
- **Found during:** Task 1 verification
- **Issue:** The pheromone regex selected unrelated build, continue, and worktree tests with separately owned missing-artifact failures, while the shelf regex matched zero owned tests.
- **Fix:** Replaced both selectors with exact anchored expressions covering every test in the five declared files.
- **Files modified:** `.planning/phases/200-iterative-planning/200-43-PLAN.md`
- **Verification:** Both corrected commands pass at `-count=2` and the shelf selector executes real cases.
- **Committed in:** Scoped plan-completion commit.

**2. [Rule 1 - Bug] Prevented the cleanup probe from restoring repository-backed globals**
- **Found during:** Task 2 cleanup review
- **Issue:** The first version of the regression probe restored captured store and tracer values, which could itself resurrect stale authority.
- **Fix:** Routed store and tracer cleanup through `saveGlobals`, which always clears repository-backed globals.
- **Files modified:** `cmd/pheromones_test.go`
- **Verification:** The owned pheromone command and adversarial sequence pass twice.
- **Committed in:** `6ae0090e`

---

**Total deviations:** 2 auto-fixed (1 blocking selector correction, 1 test cleanup bug).
**Impact on plan:** Both changes strengthen the declared containment proof without widening production behavior or touching unowned failures.

## Issues Encountered

- The original Task 1 command reproduced unrelated missing `spawn-tree.txt` and handoff fixture failures outside this plan's five-file scope. Those failures remain visible for their later repair plans; none was hidden or modified here.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 44 can continue the bounded full-suite repair sequence without inheriting pheromone or shelf fixture roots.
- The final Plan 40 receipt remains untouched until all later repairs and all seven gates pass on one unchanged tree.

## TDD Gate Compliance

- Both tasks have a failing RED commit followed by a passing GREEN commit.

## Self-Check: PASSED

- All five modified test files, the corrected plan, and this summary exist.
- All four RED/GREEN task commits and the authority cleanup correction are present in repository history.
- The scoped plan metadata commit `0d82d390` is present and bound in `STATE.md`.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
