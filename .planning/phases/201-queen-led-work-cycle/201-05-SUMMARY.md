---
phase: 201-queen-led-work-cycle
plan: "05"
subsystem: queen-orchestration
tags: [go, verification-boundary, build-dispatch, continue-dispatch, work-outcome, closeout]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 03)
    provides: "cmd/verification_boundary.go: queenApplyVerificationBoundary, attachVerificationBoundary, verificationBoundaryForAttempt -- the reconciliation function and the one attempt-bound read path this plan wires into both dispatch boundaries"
  - phase: 201-queen-led-work-cycle (plan 04)
    provides: "pkg/colony/work_outcome.go: colony.WorkOutcome, WorkOutcomeLabels(), and the equal-ceremony closeout ceremony this plan's unverified build card reuses"
provides:
  - "cmd/codex_build.go: queenBuildPostWaveDispatches now reads verificationBoundaryForAttempt (via loadLatestBuildAttempt(phase.ID)) and dispatches a post-wave reviewer (auditor/measurer/chaos) only when the recorded decision names build-end -- never re-deriving the choice from phase content"
  - "cmd/codex_continue.go: plannedContinueReviewDispatches and runCodexContinueReview read the same recorded decision; under a recorded build-end boundary the check step dispatches no reviewer and reads the build-end reviewer findings already bound to the attempt back into the check result (continueReviewReportFromBuildEndFindings) instead of re-running them"
  - "cmd/codex_build.go: buildUnverifiedCloseoutDetails / buildVerifiedCloseoutDetails -- LifecycleCloseoutDetails builders carrying colony.WorkOutcomePartial (unverified) or colony.WorkOutcomeSuccess (verified), ready to plug into the one existing production build-closeout call site (cmd/compatibility_cmds.go:1025)"
affects: [201-06, 201-07, 201-08, 201-09, 201-10, 201-11, 201-12, 201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 13009
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Attempt-bound decision read via phase ID, not a threaded parameter: both queenBuildPostWaveDispatches and plannedContinueReviewDispatches/runCodexContinueReview derive attemptRel internally via loadLatestBuildAttempt(phase.ID) rather than accepting a new attemptRel parameter -- the phase ID was already available at every call site, so this closes the boundary read without changing either function's public signature across the ~15 existing production and test call sites that would otherwise have needed updating"
    - "Consume-not-rerun on the suppressed boundary: when the check step's own reviewer dispatch is suppressed by a recorded build-end decision, runCodexContinueReview does not simply skip review -- continueReviewReportFromBuildEndFindings reads the build-end reviewer worker runs already bound to buildAttemptRecord.WorkerRuns and folds their blockers/summaries into the check result, so a real finding from build-end judgement is never silently dropped"

key-files:
  created:
    - cmd/build_unverified_card_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_continue.go
    - cmd/boundary_double_dispatch_test.go
    - cmd/codex_build_test.go
    - cmd/review_depth_test.go

key-decisions:
  - "queenBuildPostWaveDispatches and plannedContinueReviewDispatches derive attemptRel internally via loadLatestBuildAttempt(phase.ID) rather than taking a new parameter. Threading attemptRel through plannedBuildDispatchesWithJobProposals / plannedContinueReviewDispatches would have required updating ~15 existing production and test call sites across the cmd package (build_print_brief.go, coherent_job_retry_command_test.go, dispatch_coalesce_test.go, review_depth_test.go, and a dozen more); deriving it from the phase ID already in scope keeps every existing signature stable while still reading the real per-attempt record for production callers that have one."
  - "With no verification-boundary decision recorded for an attempt -- the common case in production today, since no caller yet proposes and persists one (that write-side wiring is explicitly out of scope for this plan, matching 201-03's own stated boundary) -- build-end dispatches ZERO post-wave reviewers, unconditionally, even under --heavy or an explicit --castes proposal naming measurer/auditor/chaos. This follows the plan's own explicit behavior spec (\"nothing is dispatched at build-end\" with no record) rather than carving out an exception for an explicit Queen proposal; judgement for those castes now lands at `aether continue` by default. This required updating 7 pre-existing tests (TestIndependentSpecialistsShareAWave, TestBuildPlanOnlyExecutionPlanRunsWatcherAfterSpecialists, TestBuildPlanOnlyHeavyReviewAllowsPolicyMeasurerAndChaos, TestBuildPlanOnlyCLIForwardsVerificationDepth, TestBuildCLIForwardsVerificationDepth, TestBuildCLINormalPathForwardsQueenTeamFlags, TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount, plus 3 tests in review_depth_test.go) that previously asserted measurer/chaos present in the build manifest by default -- each now records an explicit build-end boundary fixture (attemptWithVerificationBoundaryRecorded) when the test's real subject is the caste-SELECTION policy, or asserts the new absence directly when the test's subject was the boundary-crossing behavior itself."
  - "buildUnverifiedCloseoutDetails / buildVerifiedCloseoutDetails are built and fully unit-tested (TestBuildBeforeVerificationSaysItIsNotVerifiedYet, TestUnverifiedCardKeepsTheFullCeremony) but are NOT wired into any live `aether build` call site in this plan -- the plan's own file scope for this task was cmd/codex_build.go plus the new test file, and the one existing production closeout call site for a build result (cmd/compatibility_cmds.go:1025, inside the autopilot `aether run` flow) sits outside that scope and would need its own plan to thread a real verification-boundary decision and deterministic-check command list through. This mirrors 201-03's own precedent (recording mechanism built, wiring left to this plan) -- see Next Phase Readiness."

requirements-completed: [WORK-02]
# WORK-04 (this plan's other declared requirement) is also declared by
# 201-02 [complete], 201-03 [complete], and 201-15 [pending] and therefore
# stays open per the shared-ID gate (#2388) until 201-15 also has a
# SUMMARY.md -- correct, expected behavior, not a gap in this plan's own
# work.

coverage:
  - id: D1
    description: "Build-end reviewer dispatch (queenBuildPostWaveDispatches) gates on the recorded verification-boundary decision instead of dispatching unconditionally whenever a reviewer caste was selected"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestBuildEndReviewersGateOnTheRecordedBoundary"
        status: pass
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestReviewerCountAtZeroOneAndCeiling"
        status: pass
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestDeterministicChecksAreUnchangedByTheBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Check-step reviewer dispatch (plannedContinueReviewDispatches, runCodexContinueReview) reads the same recorded boundary and, under build-end, dispatches no reviewer of its own while consuming the build-end findings already bound to the attempt"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestCheckStepReviewersGateOnTheRecordedBoundary"
        status: pass
      - kind: unit
        ref: "cmd/boundary_double_dispatch_test.go#TestNoCasteIsDispatchedAtBothBoundaries"
        status: pass
    human_judgment: false
  - id: D3
    description: "A low-risk one-task change dispatches exactly one Builder plus the free program checks on both build lanes, unchanged by this plan's boundary gating"
    requirement: WORK-02
    verification:
      - kind: unit
        ref: "cmd/one_task_bug_fix_test.go#TestOneTaskBugFixIsOneWorkerPlusChecks"
        status: pass
    human_judgment: false
  - id: D4
    description: "An unverified build closeout (built, checks passed, not yet reviewed) carries a non-success work verdict, names the exact deterministic checks that ran, shares no token with the success verdict's label, and renders the identical full ceremony a verified card renders -- built as reusable, tested infrastructure not yet wired into a live `aether build` call site"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/build_unverified_card_test.go#TestBuildBeforeVerificationSaysItIsNotVerifiedYet"
        status: pass
      - kind: unit
        ref: "cmd/build_unverified_card_test.go#TestUnverifiedCardKeepsTheFullCeremony"
        status: pass
    human_judgment: true
    rationale: "The rendered card is proven against the shared lifecycle-closeout test fixture, not against a live `aether build` invocation -- wiring into the one real production call site (cmd/compatibility_cmds.go:1025) is deferred to a later plan, so no automated test proves an owner running `aether build` today sees this card."

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 05: Eliminate Double Verification and the Honest Unverified Build Card Summary

**Both the build-time and check-time reviewer dispatchers now read one recorded verification-boundary decision instead of independently deciding whether to review, closing the doubled build-plus-check review CONCERNS.md named -- plus reusable, tested infrastructure for an honest "built but not yet verified" closeout card.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-10T13:11:07Z (approx., immediately following 201-04)
- **Completed:** 2026-09-10T13:56:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `queenBuildPostWaveDispatches` (`cmd/codex_build.go`) now reads (never re-derives) the verification-boundary decision recorded on the phase's current build attempt via `loadLatestBuildAttempt(phase.ID)` + `verificationBoundaryForAttempt`. With the recorded choice naming build-end, it dispatches exactly the reviewers the Queen's selected team and the recorded decision justify; with check-step recorded, or no decision recorded at all (today's common case), it dispatches nothing -- judgement lands at `aether continue` instead.
- `plannedContinueReviewDispatches` and `runCodexContinueReview` (`cmd/codex_continue.go`) read the same recorded decision. Under a recorded build-end boundary, the check step dispatches no reviewer of its own; `runCodexContinueReview` instead reads the build-end reviewer worker runs already bound to the attempt (`continueReviewReportFromBuildEndFindings`) and folds their real findings into the check result, so a genuine build-end blocker is never silently dropped just because the check step didn't re-run it.
- Extended `cmd/boundary_double_dispatch_test.go` with `TestBuildEndReviewersGateOnTheRecordedBoundary` (check-step/build-end/no-record cases, each reviewer carrying a non-blank reason), `TestReviewerCountAtZeroOneAndCeiling` (the zero/one/ceiling threshold cases as exact integers), `TestDeterministicChecksAreUnchangedByTheBoundary` (every non-reviewer build dispatch is byte-identical regardless of the recorded boundary), and `TestCheckStepReviewersGateOnTheRecordedBoundary` (the check-step half, including a real build-end finding surfacing in the check result). Strengthened the pre-existing `TestNoCasteIsDispatchedAtBothBoundaries` to walk the same phase fixtures against both explicitly recorded boundary values, not just the unrecorded default.
- Added `buildUnverifiedCloseoutDetails` and `buildVerifiedCloseoutDetails` (`cmd/codex_build.go`), which build the `LifecycleCloseoutDetails` for a finished build carrying a work verdict from plan 201-04's vocabulary: `colony.WorkOutcomePartial` (never success) with the deterministic check commands as evidence when verification has not happened yet, or `colony.WorkOutcomeSuccess` when the build's own reviewers ran and passed. Proven with `cmd/build_unverified_card_test.go`'s two new tests: the unverified card names the real check commands and the projection's own next command, shares no token with the success verdict's label (derived from `colony.WorkOutcomeLabels()` inside the test), and renders the identical full canonical slot set a verified card renders.
- Fixed 7 pre-existing tests whose assertions depended on the OLD unconditional build-end dispatch behavior, updating each to either record an explicit build-end boundary fixture (when the test's real subject was caste-selection policy) or assert the new default absence directly (when the test's subject was the boundary-crossing behavior itself) -- see Deviations.

## Task Commits

1. **Task 1: Gate build-end reviewer dispatch on the stored boundary** - `d9a3d7b2` (feat)
2. **Task 2: Gate check-step reviewer dispatch on the same record and prove single verification** - `f99d6735` (feat)
3. **Task 3: Render the honest unverified build closeout** - `e76ca57e` (feat)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/codex_build.go` - `queenBuildPostWaveDispatches` boundary gate; `buildDeterministicCheckEvidence`, `buildUnverifiedCloseoutDetails`, `buildVerifiedCloseoutDetails`
- `cmd/codex_continue.go` - `reviewJudgementAlreadyLandedAtBuildEnd`, `continueReviewReportFromBuildEndFindings`, boundary gate on `plannedContinueReviewDispatches` and `runCodexContinueReview`
- `cmd/boundary_double_dispatch_test.go` - shared fixture helper `attemptWithVerificationBoundaryRecorded`; new tests for both boundary gates; strengthened `TestNoCasteIsDispatchedAtBothBoundaries`
- `cmd/build_unverified_card_test.go` (new) - `TestBuildBeforeVerificationSaysItIsNotVerifiedYet`, `TestUnverifiedCardKeepsTheFullCeremony`
- `cmd/codex_build_test.go` - regression fixes for the new default (no boundary recorded -> no build-end reviewer), including a fixture attempt marked terminal so `runCodexBuildPlanOnlyWithOptions`'s own "already has an active build attempt" guard does not block the test's own real build call
- `cmd/review_depth_test.go` - regression fixes for the 3 `TestBuildDispatch_*` tests that previously asserted measurer/chaos present by default

## Decisions Made

- `queenBuildPostWaveDispatches` and `plannedContinueReviewDispatches` derive `attemptRel` internally via `loadLatestBuildAttempt(phase.ID)` rather than accepting a new parameter -- avoids a signature change across ~15 existing call sites while still reading the real per-attempt record.
- With no recorded boundary decision (today's common case, since the write side is intentionally out of this plan's scope), build-end dispatches zero post-wave reviewers unconditionally, even under `--heavy` or an explicit `--castes` proposal -- the plan's own behavior spec is unconditional on this point. Required updating 7 pre-existing tests to match (see key-decisions in frontmatter for the full list).
- `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails` are built and fully tested but not wired into a live `aether build` call site -- the one existing production build-closeout call (`cmd/compatibility_cmds.go:1025`, inside `aether run`'s autopilot flow) sits outside this task's declared file scope (`cmd/codex_build.go` + new test file only) and needs its own plan to thread a real recorded boundary and check-command list through.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated 7 pre-existing tests broken by the new default (no boundary recorded -> no build-end reviewer)**
- **Found during:** Task 1, after implementing the boundary gate and running the broader `go test ./cmd -run 'Build'` sweep
- **Issue:** `TestIndependentSpecialistsShareAWave`, `TestBuildPlanOnlyExecutionPlanRunsWatcherAfterSpecialists`, `TestBuildPlanOnlyHeavyReviewAllowsPolicyMeasurerAndChaos`, `TestBuildPlanOnlyCLIForwardsVerificationDepth`, `TestBuildCLIForwardsVerificationDepth`, `TestBuildCLINormalPathForwardsQueenTeamFlags`, `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` (`cmd/codex_build_test.go`), and `TestBuildDispatch_LightMode_Chaos30Percent`, `TestBuildDispatch_HeavyMode_IncludesChaosAndMeasurer`, `TestBuildDispatch_FinalPhase_HeavyRegardlessOfLight` (`cmd/review_depth_test.go`) all asserted measurer/chaos present in the build-time dispatch list by default under `--heavy` / full depth / explicit `--castes` -- exactly the unconditional-dispatch behavior Task 1 deliberately removes. This is a direct, intended consequence of implementing the plan's own explicit behavior spec ("with no boundary record present... nothing is dispatched at build-end"), not scope creep.
- **Fix:** Each test now records an explicit build-end verification-boundary fixture (via the new shared helper `attemptWithVerificationBoundaryRecorded`, or an inline equivalent for `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` where the fixture attempt additionally needed marking terminal so the real build entrypoint's own "already has an active attempt" guard did not block it) before exercising the assertion the test was actually built to prove (caste-selection policy). `TestBuildPlanOnlyExecutionPlanRunsWatcherAfterSpecialists` and `TestBuildCLINormalPathForwardsQueenTeamFlags` instead had their assertions flipped to assert the new, correct absence directly, since their real subject (execution-stage collapsing; whether an explicit `--castes` proposal survives to the caste decision record even when it is not build-end-dispatched) still holds.
- **Files modified:** `cmd/codex_build_test.go`, `cmd/review_depth_test.go`
- **Verification:** All 10 tests re-run individually and in the full `go test ./cmd -run 'Build'` (311s) and `go test ./cmd -run 'Continue'` (106s) sweeps -- clean.
- **Committed in:** `d9a3d7b2` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking -- pre-existing test suite depended on exactly the behavior this plan's own spec requires removing). **Impact on plan:** Necessary and in-scope: these are the same tests the plan's own acceptance criteria implicitly govern (the build-time dispatch list), updated to match the new, explicitly-specified default. No unrelated behavior was touched.

## Issues Encountered

- The full, unfiltered `go test ./cmd` package suite hit the same pre-existing, environment-level resource-exhaustion condition documented in `201-02-SUMMARY.md` and `201-03-SUMMARY.md`'s own Issues Encountered sections (isolated child-process resource contention under heavy parallel load, a known kernel process-spawn-cap limitation of this project's own test suite, not something this plan's changes could cause or fix). All of this plan's own targeted verification ran clean: the full `go test ./cmd -run 'Build'` sweep (311s, 0 failures) and `go test ./cmd -run 'Continue'` sweep (106s, 0 failures), plus every task-level `<verify>` command from `201-05-PLAN.md` individually.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `verificationBoundaryForAttempt` is now read at both dispatch boundaries; the doubled review CONCERNS.md named is structurally gone for as long as no caller records a build-end decision (today's default) or after a caller reconciles and records one.
- **Open gap for a later plan:** nothing in production yet calls `queenApplyVerificationBoundary` + `attachVerificationBoundary` to actually PROPOSE and PERSIST a build-end decision -- the write side remains unbuilt (matching 201-03's own stated scope boundary, now inherited one plan further). Until it exists, build-end reviewer dispatch and the verified (success) build closeout can never actually fire in production; only the check-step default path runs.
- **Open gap for a later plan:** `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails` are fully built and tested but not wired into `cmd/compatibility_cmds.go:1025` (the one existing production build-closeout call site, inside `aether run`'s autopilot flow) or any other live `aether build` path -- a future plan needs to thread a real deterministic-check command list and the recorded boundary decision into that call site.
- `go build ./...` and `go vet ./...` are clean. Every task-level `<verify>` command from `201-05-PLAN.md` passes. Targeted regression sweeps (`Build`, 311s; `Continue`, 106s) both pass with zero failures.
- WORK-02 is now complete (declared only by this plan). WORK-04 remains open (shared with 201-02 [complete], 201-03 [complete], 201-15 [pending]) and will close once 201-15 also summarizes.
- Ready for `201-06-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/build_unverified_card_test.go` — FOUND
- `.planning/phases/201-queen-led-work-cycle/201-05-SUMMARY.md` — FOUND
- Commit `d9a3d7b2` (Task 1) — FOUND in git log
- Commit `f99d6735` (Task 2) — FOUND in git log
- Commit `e76ca57e` (Task 3) — FOUND in git log
- `go build ./...` — clean
- `go vet ./...` — clean
- All plan `<verify>` commands re-run and passing: `TestBuildEndReviewersGateOnTheRecordedBoundary`, `TestDeterministicChecksAreUnchangedByTheBoundary`, `TestReviewerCountAtZeroOneAndCeiling`, `TestNoCasteIsDispatchedAtBothBoundaries`, `TestCheckStepReviewersGateOnTheRecordedBoundary`, `TestOneTaskBugFixIsOneWorkerPlusChecks`, `TestDeterministicFloorIsTheOnlySourceOfAPass`, `TestBothContinueLanesApplyTheSameFloor`, `TestBuildBeforeVerificationSaysItIsNotVerifiedYet`, `TestUnverifiedCardKeepsTheFullCeremony`
- Full unfiltered `go test ./cmd` (555s): zero `--- FAIL` lines; the run's own "missing executed tests" accounting hit the documented pre-existing environment ceiling (isolated child-process resource contention under parallel load, same as `201-02-SUMMARY.md`/`201-03-SUMMARY.md`'s own Issues Encountered), not a regression
- Targeted `go test ./cmd -run 'Build'` (311s) and `-run 'Continue'` (106s): both clean, zero failures
