---
phase: 200-iterative-planning
plan: 32
subsystem: planning-authority
tags: [go, tdd, ast, transactions, concurrency, numeric-validation]

requires:
  - phase: 200-28
    provides: Repository-scoped planning mutation sessions and lifecycle transactions
  - phase: 200-31
    provides: Independent candidate acceptance and canonical impact authority
provides:
  - Session-first authority reads for every named plan, specification, revision, timeline, stage, candidate, and insert writer
  - All-target artifact, state, cleanup, and owner-projection publication with exact captured baselines
  - AST call inventory, separate-process stale-writer races, and strict score/position/hash boundary proofs
affects: [planning, specification, plan-revision, phase-insertion, build-authority]

tech-stack:
  added: []
  patterns: [session-dominated writers, prepared artifact publication, exact-baseline target sets, AST call inventory]

key-files:
  created:
    - cmd/planning_writer_session_200_test.go
  modified:
    - cmd/codex_plan.go
    - cmd/codex_plan_finalize.go
    - cmd/specification.go
    - cmd/plan_revision.go
    - cmd/state_extra.go

key-decisions:
  - "Every mutating public planning entry acquires the Plan 28 repository session before its first authoritative read; internal InSession helpers carry that authority through branch selection and commit."
  - "Locally generated planning artifacts are prepared off-tree and published with cleanup, colony state, session.json, CONTEXT.md, and HANDOFF.md in one declared target set."
  - "Validated worker-authored artifacts are represented as exact no-op write targets, so refresh cleanup cannot delete them and their baseline remains part of the atomic publication."
  - "Whole-integer JSON validation applies at the staged evidence boundary while legacy weighted confidence calculations retain their established fractional compatibility."

patterns-established:
  - "Prepared publication: render fallback projections outside the visible tree, then commit their bytes only with the authoritative state transition."
  - "Writer ratchet: parse Go AST CallExpr nodes and require every named mutator/deleter path to be dominated by the shared repository session."

requirements-completed: [PLAN-01, PLAN-04, PLAN-05, PLAN-06]

duration: 1h 15min
completed: 2026-09-09
---

# Phase 200 Plan 32: Complete Planning Writer Session Migration Summary

**Every named planning, specification, revision, timeline, candidate, and insert writer now derives from one locked repository snapshot and publishes all owned files—or none—through exact-baseline transactions.**

## Performance

- **Duration:** 1h 15min
- **Started:** 2026-09-08T23:42:26Z
- **Completed:** 2026-09-09T00:57:50Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Added a real Go AST call inventory covering plan branches, cleanup, staged/final publication, specification lifecycle, candidate/timeline/stage paths, and public phase insertion; comments and string matches cannot satisfy it.
- Added deterministic separate-process barriers for existing-plan metadata, intermediate iterations, Scout run-header/stage publication, insert candidates, and specification derivation, plus endpoint/overflow tests for scores, positions, and full SHA-256 identities.
- Held one repository mutation session from authoritative state/specification/candidate reads through branch selection and commit across all named writers.
- Replaced visible pre-state artifact writes and refresh deletions with prepared bytes and exact removal targets committed atomically with colony state and owner-readable session projections.
- Preserved Plan 31's canonical and independently derived acceptance authority while making candidate, timeline, stage, recovery, and legacy insert publication stale-safe.

## Task Commits

Each TDD gate and implementation task was committed atomically:

1. **Task 1 RED: Inventory planning writers and reproduce stale updates** - `b9c1231b` (test)
2. **Task 2 GREEN: Enclose plan-command writes and refresh deletion** - `3eed6346` (feat)
3. **Task 3 GREEN: Enclose specification, revision, timeline, and insert writers** - `34402994` (feat)

## Files Created/Modified

- `cmd/planning_writer_session_200_test.go` - AST call inventory, five process-race families, and numeric/hash mutation-boundary proofs.
- `cmd/codex_plan.go` - Session-first plan branches, prepared artifact publication, exact refresh/backup/fallback targets, and atomic state/session projections.
- `cmd/codex_plan_finalize.go` - Session-propagated immediate and staged finalizers, strict staged score decoding, intermediate iteration targets, timeline baselines, and atomic final publication.
- `cmd/specification.go` - Session-bound create/revise/approve authority and exact-baseline specification projection commits.
- `cmd/plan_revision.go` - Session-bound insert candidate derivation and all-target candidate/timeline/stage/state publication.
- `cmd/state_extra.go` - Session-bound public insert command, position validation, and atomic corrective recovery receipt plus state.

## Decisions Made

- Public functions own session acquisition; internal `InSession` functions must be used whenever one planning operation calls another mutating helper. This avoids both unlocked reads and nested-lock deadlocks.
- Worker-authored planning files remain authoritative only when existing claim/snapshot rules prove them; their exact bytes are still declared in the final transaction so cleanup and publication cannot split.
- Staged Route evidence accepts only plain whole JSON integers from 0 through 100. Legacy internal weighted scoring remains unchanged because it legitimately computes fractional intermediates before rounding.
- Insert endpoints remain inclusive at positions zero and `len(plan)`; every other position and every noncanonical identity fails before declaring candidate or receipt targets.

## Verification

- Task 2 exact suite (`TestPlanningWriterCoverage200`, process writers, numeric boundaries, refresh, and unchanged plan-only) - 34 passed.
- Task 3 exact suite (writer inventory/process/numeric plus specification concurrency and insert-phase coverage) - 45 passed.
- Combined final scoped suite - 50 passed.
- The same 50-test scoped suite under `go test -race` - passed.
- Separate-process writer suite with `-count=3` - 18 passed.
- Route, specification, insert, recovery, direct-plan, and finalizer compatibility sweep - 156 passed; one pre-existing invalid fixture failed before invoking the behavior under test, documented below.
- `TestPlanFinalizePreservesWorkerPhaseResearch` - passed after prepared publication was taught to retain current phase research.
- `go vet ./cmd` and `git diff --check` - passed.
- Protected `.planning/config.json` and Phase 199 pattern evidence retained their exact starting SHA-256 hashes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed a nested planning-session deadlock**

- **Found during:** Task 2 route-stage regression verification
- **Issue:** One Route evidence helper reacquired the public repository session while its caller already held the same cross-process lock.
- **Fix:** Added and propagated the session-bearing helper so the nested path reuses the caller's authority.
- **Files modified:** `cmd/codex_plan_finalize.go`
- **Verification:** All 25 Route-stage regressions passed.
- **Committed in:** `3eed6346`

**2. [Rule 1 - Bug] Preserved validated worker phase research during atomic publication**

- **Found during:** Task 2 compatibility verification
- **Issue:** The first prepared-publication implementation correctly regenerated canonical plan projections but also regenerated live worker phase research that legacy finalization preserved.
- **Fix:** Split planning-projection and phase-research preservation policy; the finalizer now regenerates canonical Scout/Route/plan projections while declaring exact live research bytes in the same transaction.
- **Files modified:** `cmd/codex_plan.go`, `cmd/codex_plan_finalize.go`
- **Verification:** `TestPlanFinalizePreservesWorkerPhaseResearch` and the final scoped suites passed.
- **Committed in:** `3eed6346`

**3. [Rule 2 - Missing Critical] Kept spawn-tree creation valid after eliminating early state commits**

- **Found during:** Task 2 direct-plan regression verification
- **Issue:** Removing an early standalone state write also removed the incidental directory/file initialization that the planning spawn recorder had relied on.
- **Fix:** Added an explicit session target for the empty spawn tree before operational spawn recording.
- **Files modified:** `cmd/codex_plan.go`
- **Verification:** `TestPlanDirectRealBelowTargetPersistsOnlyIntermediateIteration` passed.
- **Committed in:** `3eed6346`

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 2 missing critical function)
**Impact on plan:** All fixes preserve established behavior while completing the planned transaction boundary; no dependency, command surface, or out-of-scope source file was added.

## Issues Encountered

- `TestPlanFinalizeStallsOnlyAfterTwoLowImprovements` fails while constructing its test input: `planningConfidenceStopHistoryFixture` supplies placeholder assessment hashes that no longer match canonical content. The error occurs at `codex_plan_finalize_test.go:994`, before the production stop policy runs. That test file is outside Plan 32's six-file ownership and the same invalid-hash pattern affects other hand-built planning fixtures, so it was not edited.

## Deferred Issues

- Canonicalize the legacy planning-confidence fixture in its owning plan/test scope. This does not block Plan 32's AST, process, race-detector, numeric-boundary, route, specification, insert, or vet gates.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 33 can build its all-target build-start transaction on a planning/specification authority surface that no longer permits stale or partial neighboring writes.
- No Plan 32 correctness blocker remains; only the out-of-scope invalid test fixture described above is deferred.

## Self-Check: PASSED

- Summary file exists.
- Task commits `b9c1231b`, `3eed6346`, and `34402994` exist in repository history.
- The six declared source/test paths are the only implementation paths changed from the Plan 31 completion baseline.
- Protected `.planning/config.json` and Phase 199 pattern evidence retain their starting hashes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
