---
phase: 88-recovery-foundation
plan: 01
subsystem: trust-validation
tags: [provenance, build-validation, continue-validation, phantom-build, safe-colony]

# Dependency graph
requires:
  - phase: 80-82
    provides: "loop detection and circuit breaker infrastructure"
provides:
  - "validateBuildProvenance: rejects phantom builds at build-complete (SAFE-01, SAFE-02)"
  - "traceContinueProvenance: rejects claims with missing/stale provenance at continue (SAFE-03, SAFE-04)"
  - "provenance check insertion in build-finalize flow"
  - "provenance check insertion in continue-finalize flow"
affects: [89-gate-ux, 90-learning, 91-privacy, 92-unblock]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "metadata-only provenance validation (no filesystem checks)"
    - "provenance check between merge and checkpoint write"
    - "reject-and-halt pattern (no warn-and-allow)"

key-files:
  created:
    - cmd/provenance.go
    - cmd/provenance_test.go
  modified:
    - cmd/codex_build_finalize.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_build_test.go
    - cmd/codex_continue_test.go

key-decisions:
  - "Only FilesModified counts for build provenance per D-01 (not FilesCreated or TestsWritten)"
  - "Continue provenance validates manifest dispatches' Outputs field (already populated by mergeExternalBuildResults)"
  - "Pre-existing worktree test failures (TestIntegrityDetectSourceContext, TestQueenWisdomHygiene) documented as out-of-scope"

patterns-established:
  - "Provenance check after merge, before checkpoint write (build)"
  - "Provenance check after gate results read, before gate run (continue)"

requirements-completed: [SAFE-01, SAFE-02, SAFE-03, SAFE-04]

# Metrics
duration: 15min
completed: 2026-05-01
---

# Phase 88 Plan 01: Build and Continue Provenance Validation Summary

**Metadata-only provenance validation that rejects phantom builds at build-complete and traces continue claims back to valid worker outputs**

## Performance

- **Duration:** 15 min
- **Started:** 2026-05-01T16:42:00Z
- **Completed:** 2026-05-01T16:57:17Z
- **Tasks:** 2 (TDD: 4 commits)
- **Files modified:** 6

## Accomplishments
- `validateBuildProvenance` rejects builds where no worker completed with FilesModified > 0 (SAFE-01, SAFE-02)
- `traceContinueProvenance` rejects continue claims where completed dispatches have empty Outputs (SAFE-03, SAFE-04)
- Both checks use reject-and-halt pattern -- no warn-and-allow path (D-03)
- 14 unit tests covering all edge cases including nil slices, FilesCreated-only rejection, TestsWritten-only rejection

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Add failing tests for provenance validation** - `0d3ee8a2` (test)
2. **Task 1 GREEN: Implement provenance validation functions** - `edb373ea` (feat)
3. **Task 2: Wire provenance into build/continue finalize** - `f2ffcb36` (feat)

## Files Created/Modified
- `cmd/provenance.go` - Two exported functions: validateBuildProvenance, traceContinueProvenance
- `cmd/provenance_test.go` - 14 tests: 8 for build provenance, 6 for continue provenance
- `cmd/codex_build_finalize.go` - Inserted validateBuildProvenance call after mergeExternalBuildResults
- `cmd/codex_continue_finalize.go` - Inserted traceContinueProvenance call before runCodexContinueGates
- `cmd/codex_build_test.go` - Added FilesModified to builder worker result for realistic provenance
- `cmd/codex_continue_test.go` - Added Outputs to completed dispatches in seedContinueBuildPacket

## Decisions Made
- Used canonical `"completed"` status (not `"success"`) matching codebase convention -- D-01 says "status=success" but the codebase never uses that value
- Only `FilesModified > 0` counts per D-01 -- FilesCreated-only and TestsWritten-only workers are correctly rejected
- Continue provenance checks `manifest.Data.Dispatches` (the build manifest's dispatches which already have Outputs populated by mergeExternalBuildResults)
- Test data updated to be realistic rather than adding skip logic to provenance checks

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Duplicate `contains` helper function**
- **Found during:** Task 1 (RED phase)
- **Issue:** `contains` function already declared in `cmd/medic_scanner_test.go`, causing redeclaration compilation error
- **Fix:** Removed duplicate helper from provenance_test.go, added comment referencing existing definition
- **Files modified:** cmd/provenance_test.go
- **Committed in:** `0d3ee8a2`

**2. [Rule 3 - Blocking] Missing .aether/rules directory in worktree**
- **Found during:** Task 1 (RED phase verification)
- **Issue:** `embedded_assets.go` embeds `.aether/rules` which doesn't exist in worktree, causing build failure
- **Fix:** Created `.aether/rules/.gitkeep` to satisfy embed directive
- **Files modified:** .aether/rules/.gitkeep
- **Committed in:** `0d3ee8a2`

**3. [Rule 2 - Missing Critical] Test data unrealistic for provenance validation**
- **Found during:** Task 2 (GREEN phase verification)
- **Issue:** 4 existing tests failed because test data had completed workers/dispatches without FilesModified/Outputs, which our provenance checks correctly reject
- **Fix:** Updated `seedContinueBuildPacket` to add Outputs for completed dispatches; added `FilesModified` to builder result in build finalize test
- **Files modified:** cmd/codex_build_test.go, cmd/codex_continue_test.go
- **Committed in:** `f2ffcb36`

---

**Total deviations:** 3 auto-fixed (1 bug, 1 blocking, 1 missing critical)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- 2 pre-existing test failures in worktree environment (`TestIntegrityDetectSourceContext`, `TestQueenWisdomHygiene`) -- documented as out-of-scope per deviation rules

## Known Stubs
None.

## Threat Flags
None -- no new network endpoints, auth paths, or file access patterns introduced beyond the planned trust boundary.

## Next Phase Readiness
- Provenance foundation complete for SAFE-01 through SAFE-04
- Ready for gate failure UX (plan 88-02), privacy gate (plan 88-03), and unblock command (plan 88-04)

## Self-Check: PASSED

---
*Phase: 88-recovery-foundation*
*Completed: 2026-05-01*
