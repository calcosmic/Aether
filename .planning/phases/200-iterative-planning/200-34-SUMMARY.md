---
phase: 200-iterative-planning
plan: 34
subsystem: build-lifecycle
tags: [go, tdd, transactions, dispatch-ordering, process-race, authority-binding]

requires:
  - phase: 200-31
    provides: Canonical accepted specification, candidate, timeline, and plan-revision authority
  - phase: 200-33
    provides: Receipt-last atomic build-start transaction and closed variant effect matrix
provides:
  - Canonical transaction callers for direct, plan-only, Queen-led, external-unbound, automatic check-fix, and coherent child-retry starts
  - Receipt-before-dispatch ordering with stale/fault refusal at every production start boundary
  - Six-family AST ratchet and deterministic accepted-revision/public-build process race
affects: [plan-35-test-fixture-migration, plan-36-helper-retirement, build, build-finalize, continue, retry]

tech-stack:
  added: []
  patterns: [prepare-then-commit, receipt-gated dispatch, closed caller-family ratchet, deterministic process scheduling seam]

key-files:
  created:
    - cmd/build_start_callers_200_test.go
  modified:
    - cmd/codex_build.go
    - cmd/forced_reviewer_waiver.go
    - cmd/codex_build_finalize.go
    - cmd/check_fix_attempt.go
    - cmd/coherent_job_retry.go

key-decisions:
  - "Plan-only and Queen-led builds share one preparation path; the Queen selects a closed start variant before that path commits, so no second attempt or manifest rewrite occurs."
  - "Worker briefs are derived before build start but written only after the receipt-last transaction succeeds; stale or faulted starts cannot publish runnable brief artifacts."
  - "External finalize preserves an already-bound attempt and invokes the canonical transaction only for the unbound compatibility lane."
  - "A test-only pre-commit scheduling seam pauses a fully prepared public request outside the repository session, making the accepted-revision race deterministic without weakening production authority checks."

patterns-established:
  - "Production caller boundary: prepare a complete buildStartRequest, call commitBuildStart exactly once, then perform dispatch or completion work from its durable receipt."
  - "Caller coverage proof: executable AST calls, relative ordering, runtime fault callbacks, replay counts, and a real second process jointly guard against split start writers."

requirements-completed: [PLAN-06]

duration: 27min
completed: 2026-09-09
---

# Phase 200 Plan 34: Production Build-Start Caller Migration Summary

**All six production build-start families now commit one complete authority-bound transaction before any worker dispatch, completion processing, or retry work can proceed.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-09-09T01:38:02Z
- **Completed:** 2026-09-09T02:04:36Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Replaced direct and host-prepared split attempt, pointer, manifest, lifecycle-state, stale-cleanup, and reviewer-window writes with one `commitBuildStart` request.
- Made Queen-led preparation select its own closed transaction variant while reusing the plan-only implementation exactly once.
- Moved external-unbound completion/claims, automatic check-fix provenance, and coherent child parent/job provenance into their respective atomic start transactions.
- Delayed worker-brief publication and every actual dispatch until the durable start receipt exists; transaction refusal returns before worker invocation.
- Added an AST ratchet over all six production caller families plus callback, replay, and real two-process race proofs.

## Task Commits

Each task followed a fail-first RED/GREEN boundary and was committed atomically:

1. **Task 1 RED: Expose split direct and host starts** - `efeadae2` (test)
2. **Task 1 GREEN: Commit direct, plan-only, Queen-led, and reviewer effects atomically** - `073e48c9` (feat)
3. **Task 2 RED: Expose split auxiliary starts** - `2386b7ef` (test)
4. **Task 2 GREEN: Commit external, check-fix, and child-retry starts atomically** - `7adff93a` (feat)
5. **Task 3 RED: Require complete caller/order/process-race proof** - `97f00c7e` (test)
6. **Task 3 GREEN: Add deterministic stale-revision boundary and complete enforcement** - `a8325159` (feat)
7. **Rule 1 fix: Preserve the canonical plan-only state baseline across read-only repair** - `c5e0534f` (fix)

## Files Created/Modified

- `cmd/build_start_callers_200_test.go` - Exact caller-family AST coverage, pre-commit writer and dispatch-order checks, durable callback/fault proof, bound replay/Queen non-duplication, and an accepted-revision process race.
- `cmd/codex_build.go` - Canonical direct/plan-only/Queen callers, pure worker-brief preparation, receipt-gated persistence/dispatch, request construction, stale target enumeration, and deterministic test scheduling seam.
- `cmd/forced_reviewer_waiver.go` - Retained the legacy test-facing reopen helper while removing its production caller and documenting transaction ownership.
- `cmd/codex_build_finalize.go` - Atomic external-unbound start with checkpoint, completion, claims, lifecycle promotion, and latest pointer; bound attempts remain preserved.
- `cmd/check_fix_attempt.go` - Atomic check-fix attempt, pointer, provenance, and reviewer close before its single builder dispatch.
- `cmd/coherent_job_retry.go` - Atomic non-latest child attempt with parent attempt, job name, selected unfinished tasks, and dispatch journal.

## Decisions Made

- The host-prepared path passes its start variant into shared preparation rather than rewriting a plan-only attempt into Queen-led state afterward.
- Brief bytes are prepared before the transaction for manifest hashing, but their files are published only after receipt; stale cleanup itself remains an atomic transaction effect.
- The deterministic race hook is nil in production and runs before repository-session acquisition, which tests the real stale-authority boundary rather than manufacturing an in-lock mutation.
- Tests count only syntactically valid attempt IDs because durable `.start-receipt.json` files currently share the attempt directory with journals.

## Verification

- Task 1 focused caller/plan-only/Queen/reviewer gate - passed (13 tests at the task boundary).
- Task 2 focused caller/external/check-fix/retry gate - passed (17 tests at the task boundary).
- Final combined Plan 34 focused suite - passed in 23.995s.
- Caller AST/order/process-race proof repeated three times - passed in 27.295s.
- Final caller/process suite under `go test -race` - passed.
- Canonical-state regression `TestBuildPlanOnlyUsesPriorPhaseEvidenceWithoutMutatingState` - passed.
- `go vet ./cmd`, implementation-range `git diff --check`, and exact six-file scope check - passed.
- The known repository-wide `go test ./...` command was not run because the plan excludes it and the repository documents its approximately 11-minute runtime.

## TDD Gate Compliance

- Task 1 RED failed because direct and plan-only production functions had no `commitBuildStart` call and still assembled split start state; GREEN replaced both paths.
- Task 2 RED failed because external-unbound finalize and the auxiliary callers still used partial helpers; GREEN moved their complete effects into the transaction.
- Task 3 RED failed to compile on the deliberately required `BuildStartBeforeCommit` scheduling seam; GREEN added the narrow seam and made the deterministic process proof pass.
- The extended regression suite then exposed an aliasing bug in plan-only state repair; the pre-existing regression failed first and passed after `c5e0534f` deep-cloned the canonical baseline.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Protected the canonical plan-only baseline from slice aliasing**
- **Found during:** Final extended plan-only regression verification
- **Issue:** The in-memory prior-phase task repair shared nested slices with the state used to hash the transaction request, making a deliberately read-only projection appear stale against disk.
- **Fix:** Deep-cloned the canonical state before applying the projection and explicitly bound the manifest to the canonical plan hash.
- **Files modified:** `cmd/codex_build.go`
- **Commit:** `c5e0534f`

## Deferred Issues

- `listBuildAttemptsForPhase` in `cmd/build_attempt.go` currently decodes `.start-receipt.json` files as empty attempt records because it excludes completion packets but not start receipts. Production callers touched here tolerate the empty record, and the Plan 34 tests filter by `validBuildAttemptID`; Plan 36 already owns `cmd/build_attempt.go` for helper retirement and is the scoped place to exclude receipt files.
- Three legacy `cmd/codex_build_test.go` fixtures seed a sole phase with ID 3 or 4 and invoke public phase 1. The canonical transaction correctly refuses that mismatched authority, while valid-ID Plan 34 gates pass. Plan 35 already owns this fixture file and should normalize those legacy IDs without weakening their Queen-policy assertions.

## Known Stubs

None.

## User Setup Required

None - no packages, credentials, or external services were added.

## Next Phase Readiness

- Plan 35 can migrate the first bounded fixture set to the now-production-proven transaction-only helper.
- Plan 36 can migrate the remaining fixtures, exclude receipt files from attempt enumeration, and remove the legacy partial helper declarations.
- No Plan 34 production, race, verification, or scope blocker remains.

## Self-Check: PASSED

- All six declared source/test files exist and are the only implementation paths changed from the Plan 33 closeout baseline.
- Commits `efeadae2`, `073e48c9`, `2386b7ef`, `7adff93a`, `97f00c7e`, `a8325159`, and `c5e0534f` exist in repository history.
- The summary exists, focused/repeated/race/vet gates pass, and no tracked file was deleted.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remains present and unstaged.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
