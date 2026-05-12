---
phase: 108-golden-workflow-tests
plan: 01
subsystem: testing
tags: [golden-tests, snapshot-tests, ceremony-output, state-mutations, regression]

# Dependency graph
requires: []
provides:
  - Golden snapshot baselines for plan, build, continue ceremony output
  - State mutation verification test covering full lifecycle transitions
  - ANSI stripping and ceremony log filtering utilities for test normalization
affects: [109-hybrid-runtime-ts-host]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Golden snapshot testing with non-deterministic output normalization
    - Worker name hash-based normalization (temp-dir-dependent names -> Worker-XX)
    - Ceremony log filtering (concurrent goroutine output removed from golden comparison)

key-files:
  created:
    - cmd/golden_workflow_test.go
    - cmd/testdata/golden_plan.txt
    - cmd/testdata/golden_build.txt
    - cmd/testdata/golden_continue.txt
  modified: []

key-decisions:
  - "Worker names normalized to Worker-XX because deterministicAntName hashes depend on temp directory paths"
  - "Ceremony [CEREMONY] and COLONY ACTIVITY log lines filtered out of golden comparison due to non-deterministic goroutine ordering"
  - "Used existing loadColonyState() from swarm_display.go instead of creating a new helper"
  - "State mutation test uses default JSON mode (not visual) for cleaner state assertions"

requirements-completed: [TEST-01, TEST-02, TEST-03, TEST-04, TEST-05]

# Metrics
duration: 16min
completed: 2026-05-12
---

# Phase 108 Plan 01: Golden Lifecycle Snapshot Tests Summary

**Four golden snapshot tests capturing plan/build/continue ceremony output and state mutation transitions across the full colony lifecycle**

## Performance

- **Duration:** 16 min
- **Started:** 2026-05-12T12:41:02Z
- **Completed:** 2026-05-12T12:57:27Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Golden snapshot tests for plan, build, and continue ceremony output with stable baselines
- State mutation test proving COLONY_STATE.json transitions: READY -> BUILT -> READY across lifecycle
- Full test suite passes with race detection (CI integration confirmed)
- Normalization utilities for handling non-deterministic output in snapshot tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Create golden lifecycle snapshot tests for plan, build, and continue** - `f7601a6` (test)
2. **Task 2: Add state mutation verification test and confirm CI integration** - `00b8a1e` (test)

## Files Created/Modified
- `cmd/golden_workflow_test.go` - Four test functions and normalization helpers
- `cmd/testdata/golden_plan.txt` - ANSI-stripped plan ceremony baseline (P L A N, Planning Wave, aether build 1)
- `cmd/testdata/golden_build.txt` - ANSI-stripped build ceremony baseline (B U I L D D I S P A T C H, S P A W N P L A N, stage markers)
- `cmd/testdata/golden_continue.txt` - ANSI-stripped continue ceremony baseline (Verification, phase completion)

## Decisions Made
- Worker name normalization to Worker-XX: deterministic worker names depend on temp directory paths used as hash seeds, making them non-deterministic across test runs. Normalizing to Worker-XX preserves structure while ensuring stability.
- Ceremony log filtering: concurrent goroutine output ([CEREMONY], COLONY ACTIVITY, wave progress, worker status lines) has non-deterministic ordering. These are filtered out since they are tested separately by unit tests on ceremony logging functions.
- Reused existing loadColonyState() from swarm_display.go instead of creating a duplicate helper, wrapping it in loadTestColonyState(t) for test convenience.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed non-deterministic golden file output due to temp-directory-dependent worker name hashes**
- **Found during:** Task 1 (TestGoldenPlanVisualOutput)
- **Issue:** Worker names like "Path-46" vs "Path-56" varied between test runs because deterministicAntName() hashes include the temp directory path as a seed. Each go test invocation creates a different temp dir, producing different name prefixes and numbers.
- **Fix:** Added normalizeWorkerNames() that replaces all CapitalWord-Number patterns with "Worker-XX". The plan specified only ANSI stripping (D-02), but this was insufficient for stable golden files.
- **Files modified:** cmd/golden_workflow_test.go
- **Verification:** Tests pass 3 consecutive times without -update-golden
- **Committed in:** `f7601a6` (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed non-deterministic golden file output due to concurrent ceremony log ordering**
- **Found during:** Task 1 (TestGoldenBuildVisualOutput)
- **Issue:** Build output includes concurrent ceremony log lines ([CEREMONY], COLONY ACTIVITY, wave progress, worker status) that have non-deterministic goroutine scheduling order. Two runs produce identical content in different order, causing golden mismatch.
- **Fix:** Added normalizeForGolden() that filters out ceremony activity lines, COLONY ACTIVITY blocks, wave progress lines, activity section headers (Context:, Completed:, Active:), indented ceremony context lines, worker status references, and wave progress tables. Only structural output (banners, stage markers, spawn plans, artifacts, task lists, next-up guidance) is kept for comparison.
- **Files modified:** cmd/golden_workflow_test.go
- **Verification:** Tests pass 3 consecutive times
- **Committed in:** `f7601a6` and `00b8a1e` (both task commits)

**3. [Rule 1 - Bug] Fixed golden file path resolution when working directory changes**
- **Found during:** Task 1 (first golden file generation)
- **Issue:** Tests call withWorkingDir(t, root) which changes the cwd to a temp directory. Relative paths like "testdata/golden_plan.txt" resolved against the temp dir instead of the source dir, causing "no such file or directory" errors.
- **Fix:** Added goldenTestdataDir() that uses runtime.Caller(0) to compute the absolute path to cmd/testdata/ before the working directory changes.
- **Files modified:** cmd/golden_workflow_test.go
- **Committed in:** `f7601a6` (Task 1 commit)

**4. [Rule 3 - Blocking] Fixed loadColonyState redeclaration**
- **Found during:** Task 2 (TestGoldenStateMutations)
- **Issue:** Plan specified creating a loadColonyState() helper, but one already exists in swarm_display.go with a different signature (returns (*colony.ColonyState, error) with no args).
- **Fix:** Renamed to loadTestColonyState(t) that wraps the existing loadColonyState() with test-friendly error handling.
- **Files modified:** cmd/golden_workflow_test.go
- **Committed in:** `00b8a1e` (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (1 bug, 3 blocking)
**Impact on plan:** All auto-fixes necessary for test stability and correctness. No scope creep. The normalization approach goes beyond the plan's D-02 (strip ANSI) but is essential for deterministic golden files in a concurrent output environment.

## Issues Encountered
- The plan assumed golden snapshot comparison would work with just ANSI stripping, but the Go runtime's concurrent ceremony logging and temp-directory-dependent worker name hashing required additional normalization layers. This was discovered incrementally and resolved with increasingly specific filters.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 5 requirements (TEST-01 through TEST-05) are covered by the four test functions
- Golden files serve as the behavioral contract for Phase 109 (TypeScript host)
- Full test suite passes with race detection, confirming CI compatibility
- No stubs remain -- all tests are fully functional

## Self-Check: PASSED

All files exist, all commits found, no unexpected deletions, no untracked files.

---
*Phase: 108-golden-workflow-tests*
*Completed: 2026-05-12*
