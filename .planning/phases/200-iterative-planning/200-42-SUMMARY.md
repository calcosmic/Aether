---
phase: 200-iterative-planning
plan: 42
subsystem: testing
tags: [go, cobra, repository-containment, test-isolation]

# Dependency graph
requires:
  - phase: 200-41
    provides: repository-authorized command-test binding and authority-first cleanup
provides:
  - seeded command fixtures bound to one physical repository authority
  - exact AETHER_ROOT and COLONY_DATA_DIR restoration across curation, flags, history, memory, and phase tests
  - bidirectional memory/phase order proof with cleared store and tracer globals
affects: [200-40, 200-43, final-gate, command-tests]

# Tech tracking
tech-stack:
  added: []
  patterns: [seeded repository-authorized fixtures, exact testing cleanup, order-independent command tests]

key-files:
  created: []
  modified:
    - cmd/curation_cmds_test.go
    - cmd/flags_test.go
    - cmd/history_test.go
    - cmd/memory_test.go
    - cmd/phase_test.go
    - cmd/memory_details_render_test.go

key-decisions:
  - "Seed legacy command fixture files into the Plan 41 repository binder instead of pairing an unbound store with AETHER_ROOT."
  - "Exercise memory and phase cleanup in both orders so a deleted temporary authority cannot survive indirectly."
  - "Treat the same defect in memory_details_render_test.go as a bounded Rule 3 verification blocker without changing product assertions."

patterns-established:
  - "Seeded command tests: create authority first, then copy fixture files into its repository-local data directory."
  - "Cobra fixture cleanup: clear store and tracer before exact environment restoration and temporary-directory removal."

requirements-completed: [CEC-03, PLAN-06]

# Metrics
duration: 9min
completed: 2026-09-09
---

# Phase 200 Plan 42: Stale-Root Test Cleanup Summary

**Curation, flag, history, memory, and phase command fixtures now use one contained repository and restore process authority exactly in repeated and reversed execution.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-09-09T09:07:35Z
- **Completed:** 2026-09-09T09:16:20Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Replaced every owned self-restoring `AETHER_ROOT` defer with the Plan 41 repository binder.
- Added a seeded binder that preserves the existing status fixture data while aligning root, data directory, store, and tracer authority.
- Removed redundant `newTestStore` cleanup from the owned Cobra paths and retained all existing output and compatibility assertions.
- Proved curation/flags/history cleanup and both memory-to-phase execution orders repeatedly, including shuffled runs.
- Closed the same root/data mismatch in three memory-details/status Cobra fixtures selected by the declared Task 2 verification.

## Task Commits

Each task was committed atomically through its RED and GREEN test-only TDD gates:

1. **Task 1 RED: Reproduce stale command root cleanup** - `43c33130` (test)
2. **Task 1 GREEN: Isolate curation, flag, and history fixtures** - `7e795816` (test)
3. **Task 2 RED: Reproduce memory and phase root leakage** - `5d585032` (test)
4. **Task 2 GREEN: Isolate memory and phase fixtures** - `3c755dbe` (test)
5. **Rule 3 verification repair: Contain memory-details command fixtures** - `c773b7e8` (test)

**Plan metadata:** `b954fa75` (docs), followed by a scoped provenance-restoration commit.

## Files Created/Modified

- `cmd/curation_cmds_test.go` - Seeds legacy fixture files inside the shared repository-authorized test binding.
- `cmd/flags_test.go` - Uses bound seeded or empty repositories for flag read/write commands.
- `cmd/history_test.go` - Uses contained fixtures and proves curation, flags, and history restore prior authority.
- `cmd/memory_test.go` - Uses contained fixtures and proves memory/phase isolation in both orders.
- `cmd/phase_test.go` - Runs every phase command against repository-authorized seeded data.
- `cmd/memory_details_render_test.go` - Binds the Cobra-backed memory-details and status cases selected by the task gate.

## Verification

- `go test ./cmd -run '^Test(Curation|Flag|History)' -count=2` - PASS
- `go test ./cmd -run '^Test(Memory|Phase)' -count=2` - PASS
- `go test ./cmd -run '^(TestHistoryCurationFlagsRestoreRepositoryAuthority200|TestMemoryPhaseRestoreRepositoryAuthority200)$' -count=5 -shuffle=on` - PASS
- `! rg -n 'defer os\.Setenv\("AETHER_ROOT", os\.Getenv\("AETHER_ROOT"\)\)' cmd/{curation_cmds,flags,history,memory,phase}_test.go` - PASS

## Decisions Made

- Reused production-equivalent repository authorization from Plan 41 and copied only top-level fixture files, matching the old `setupTestStore` fixture contract.
- Kept all changes in tests; production containment, planning authority, and command behavior remain unchanged.
- Expanded by one test file only because the plan's exact Task 2 regex selected it and it failed for the identical root/data binding defect.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Bound memory-details Cobra fixtures selected by the declared verification**
- **Found during:** Task 2 verification
- **Issue:** `^Test(Memory|Phase)` also selects `TestMemoryDetailsJSONFlagStillReturnsTheEnvelope` in `cmd/memory_details_render_test.go`; it failed alone because `newTestStore` set a temporary data path while `AETHER_ROOT` still selected the source checkout.
- **Fix:** Routed the three Cobra-backed cases in that file through `bindCommandTestRepository` and kept their rendering, read-only fingerprint, and status assertions unchanged.
- **Files modified:** `cmd/memory_details_render_test.go`
- **Verification:** The formerly failing test passes alone and the exact Task 2 command passes twice.
- **Committed in:** `c773b7e8`

---

**Total deviations:** 1 auto-fixed (1 blocking issue).
**Impact on plan:** The bounded test-only expansion was required for the declared gate and applied the same containment repair without changing runtime behavior.

## Issues Encountered

- The first Task 2 gate run exposed the extra memory-details fixture. Its isolated failure confirmed the same root/data mismatch; the scoped Rule 3 commit resolved it.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 43 can migrate the pheromone and shelf lifecycle fixture family using the same repository-authorized pattern.
- Plan 40 remains the final receipt-only proof after all repair plans complete.

## TDD Gate Compliance

- Both tasks have a failing RED test commit followed by a passing GREEN test-only commit; test-only commit types preserve the repository's commit taxonomy.

## Self-Check: PASSED

- All six modified test files and this summary exist.
- All four task RED/GREEN commits, the scoped Rule 3 repair commit, and the plan metadata commit are present in repository history.
- All declared plan verification commands pass on the current tree.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
