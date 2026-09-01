---
phase: 193-free-checks-are-the-floor
plan: 01
subsystem: verification
tags: [go, continue-pipeline, criterion-evidence, deterministic-checks]

# Dependency graph
requires: []
provides:
  - "runDeterministicFloor(ctx, root, phase, manifest, watcher, timeout) — the single deterministic-verification body (shell steps, claims, criterion evidence) both continue lanes call"
  - "continueWatcherDecision — the single decision point for whether continue dispatches a reviewer worker at all"
  - "evaluateCriterionCheckDetail / criterionCheckOutcome — the three-outcome watcher check (dispatched+pass, dispatched+fail, no-dispatch/deterministic proof)"
  - "deterministicFloorSatisfies — the deterministic-evidence test a watcher-bound criterion falls back to when no reviewer was dispatched"
affects: [193-02, 193-03, 193-04, 193-05]

# Actuals (#2632)
actuals:
  tokens: 15716
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared deterministic-floor extraction: both continue lanes (in-process and wrapper/external) call one function so parity is structural, not a discipline"
    - "Reviewer-can-only-add-a-block: the deterministic floor is computed with the already-resolved watcher value; a live dispatch afterward can only ADD a failure block, never retroactively supply the pass"

key-files:
  created:
    - cmd/blackbox_zero_reviewer_test.go
    - cmd/deterministic_floor.go
    - cmd/deterministic_floor_test.go
  modified:
    - cmd/criterion_evidence.go
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_continue_test.go

key-decisions:
  - "syntheticCriterionRequirements' default check set drops \"watcher\" (D-06): unbound criteria get claims + the matching free check, never a reviewer caste by default."
  - "The deterministic floor is computed with the build-time watcher (already resolved from the manifest, no live action needed), not a live continue-time dispatch — avoids re-running shell verification a second time just to build a reviewer's brief, and structurally guarantees a dispatched reviewer can only ADD a block, never supply the pass (assumption_delta_decision in the plan)."
  - "The zero-executed-checks warning is now plain English (\"no tests to run in this project\") with no clause handing verification responsibility to a reviewer (D-01)."
  - "runCodexContinueVerificationSnapshot's watcher-block check now excludes a \"skipped\"-status watcher, fixing a real asymmetry the wrapper lane had (isSuccessfulExternalBuildStatus(\"skipped\") is false, so a never-dispatched-or-skipped watcher used to read as a hard block there)."

patterns-established:
  - "criterionCheckOutcome struct + evaluateCriterionCheckDetail: evaluateCriterionCheck stays as a thin 3-value wrapper so existing callers (cmd/verify_out_of_band.go) compile unchanged."

requirements-completed: []  # FLOOR-01/02/03 are also declared by later plans in this phase (193-02..193-05); requirements.ready-ids confirms none are ready to mark complete from this plan alone.

coverage:
  - id: D1
    description: "A phase with zero reviewer workers advances when the free checks pass and is blocked when they fail, proven end to end through the real compiled aether continue binary (FLOOR-02, both directions)."
    requirement: FLOOR-02
    verification:
      - kind: e2e
        ref: "cmd/blackbox_zero_reviewer_test.go#TestZeroReviewerPhaseAdvancesWhenFreeChecksPass"
        status: pass
      - kind: e2e
        ref: "cmd/blackbox_zero_reviewer_test.go#TestZeroReviewerPhaseIsBlockedWhenFreeChecksFail"
        status: pass
    human_judgment: false
  - id: D2
    description: "A project with no resolvable verification command still runs claimed-files-exist and each-criterion-has-evidence, advances on those alone, and warns in plain English rather than handing verification to a reviewer (FLOOR-01 empty, D-01)."
    requirement: FLOOR-01
    verification:
      - kind: e2e
        ref: "cmd/blackbox_zero_reviewer_test.go#TestZeroExecutedChecksStillRunsClaimsAndCriteria"
        status: pass
      - kind: unit
        ref: "cmd/codex_continue_test.go#TestRunCodexContinueVerificationWarnsWhenAllCommandsAreSkipped"
        status: pass
    human_judgment: false
  - id: D3
    description: "A dispatched reviewer that failed still blocks even though every free check is green — the deterministic floor and a reviewer verdict do not merge into a pass (D-06 adjacency)."
    requirement: FLOOR-03
    verification:
      - kind: e2e
        ref: "cmd/blackbox_zero_reviewer_test.go#TestDispatchedReviewerThatFailedStillBlocks"
        status: pass
    human_judgment: false
  - id: D4
    description: "A criterion bound to a watcher check is satisfied by deterministic evidence when no reviewer was dispatched, or its status is skipped, and still blocks when a dispatched one failed."
    requirement: FLOOR-03
    verification:
      - kind: unit
        ref: "cmd/criterion_evidence.go#evaluateCriterionCheckDetail (three-outcome watcher case, exercised transitively by every blackbox test's underlying criteria path)"
        status: pass
    human_judgment: true
    rationale: "No test in this plan binds a criterion to Checks:[\"watcher\"] AND exercises a live-dispatched-and-passed continue-time watcher against it (the narrow combination the Task 2 design decision changes) — flagged for 193-02/193-03 to add and confirm against FLOOR-03 directly."
  - id: D5
    description: "Both continue lanes (in-process aether continue and the wrapper continue --plan-only/continue-finalize path) share one deterministic-floor body and agree field by field for the same inputs."
    verification:
      - kind: unit
        ref: "cmd/deterministic_floor_test.go#TestBothContinueLanesApplyTheSameFloor"
        status: pass
      - kind: unit
        ref: "cmd/deterministic_floor_test.go#TestDeterministicFloorStepOrderIsFixed"
        status: pass
    human_judgment: false
  - id: D6
    description: "The deterministic floor is the only source of a pass — no reviewer verdict (passing, skipped, or absent) can flip ChecksPassed from false to true."
    verification:
      - kind: unit
        ref: "cmd/deterministic_floor_test.go#TestDeterministicFloorIsTheOnlySourceOfAPass"
        status: pass
    human_judgment: false

duration: 50min
completed: 2026-08-22
status: complete
---

# Phase 193 Plan 01: Tracer and Deterministic Floor Summary

**The program's own free checks (build, types, lint, tests, claimed-files-exist, criterion evidence) now decide phase advancement on their own — zero reviewer workers dispatched, proven end to end through the real `aether continue` binary in both directions — and both continue lanes (in-process and wrapper) compute that result from one shared function instead of two independently-maintained copies.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-08-22T12:17:38Z (approx, per STATE.md session start)
- **Completed:** 2026-08-22T13:07:11Z
- **Tasks:** 2 completed
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments

- A phase with no reviewer worker dispatched (`--skip-watchers`) advances when the project's real build/types/lint/tests commands all resolve and pass, and blocks — with the failing check named — when one fails, proven by running the real compiled `aether` binary against a fixture colony (FLOOR-02, both directions).
- A project with nothing mechanical to check (no resolvable verification command) still runs claimed-files-exist and each-criterion-has-evidence, advances on those alone, and warns in plain English ("no tests to run in this project") instead of silently handing verification responsibility to a reviewer (D-01, FLOOR-01 empty case).
- A dispatched reviewer that fails still blocks even though every free check is green — the deterministic floor and a reviewer verdict never merge into a pass.
- `syntheticCriterionRequirements`'s synthetic default no longer names a reviewer caste (D-06): unbound criteria get `claims` + the matching free check.
- Both continue lanes (`runCodexContinueVerification` and `runCodexContinueVerificationSnapshot`) now call one shared `runDeterministicFloor`, closing the real behavioural gap the wrapper lane had (a never-dispatched or "skipped"-status watcher used to read as a hard block there, and the floor's warnings were silently dropped on that lane).

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "a phase with no reviewer is still checked, and the checks decide"** — `623229e2` (feat)
2. **Task 2: One floor body, called by both continue lanes** — `ee805ec1` (feat)

_Note: both tasks were TDD (RED confirmed by temporarily reverting the implementation files and re-running the new tests before restoring; for Task 2, RED was confirmed by removing `cmd/deterministic_floor.go`, which fails the build)._

## Files Created/Modified

- `cmd/criterion_evidence.go` — `syntheticCriterionRequirements` default checks drop `"watcher"`; new `criterionCheckOutcome` type; `evaluateCriterionCheckDetail` (three-outcome watcher case) with `evaluateCriterionCheck` as a thin wrapper; new `deterministicFloorSatisfies`.
- `cmd/codex_continue.go` — new `continueWatcherDecision` (single dispatch decision, five existing non-dispatch reasons, never consults the deterministic result); `runCodexContinueVerification` now computes `runDeterministicFloor` first, then decides/dispatches the reviewer, then only adds a block on a dispatched failure.
- `cmd/codex_continue_plan.go` — `runCodexContinueVerificationSnapshot` now shares `runDeterministicFloor`; fixed the skipped-watcher-should-not-block asymmetry; now populates `Warnings`.
- `cmd/codex_continue_test.go` — updated `TestRunCodexContinueVerificationWarnsWhenAllCommandsAreSkipped`'s assertion and comment to match the intentionally-changed plain-English warning text (D-01).
- `cmd/blackbox_zero_reviewer_test.go` (new) — 4 black-box tests driving the real compiled binary, plus a local `writeZeroReviewerVerificationCommands` helper and a `prepareZeroReviewerChecksOnlyFixture` variant fixture (kept out of `cmd/blackbox_harness_test.go` per the plan's read_first note).
- `cmd/deterministic_floor.go` (new) — `deterministicFloorResult` and `runDeterministicFloor`, the shared floor body.
- `cmd/deterministic_floor_test.go` (new) — `TestBothContinueLanesApplyTheSameFloor` (5-fixture table), `TestDeterministicFloorIsTheOnlySourceOfAPass`, `TestDeterministicFloorStepOrderIsFixed`.

## Decisions Made

- **Floor uses the build-time watcher, not a live continue-time dispatch.** `runDeterministicFloor` is called with whatever watcher value is already resolved (`evaluateContinueWatcherVerification(manifest)`) before any live dispatch this call might still make. Dispatch (when `continueWatcherDecision` says yes) reuses `floor.Steps`/`floor.Claims` for the reviewer's brief context rather than re-running shell verification a second time. The tradeoff: a criterion bound to `required_checks: watcher` no longer gets satisfied by a *live continue-time* dispatched watcher's pass — only by (a) a build-time watcher already recorded as passed, or (b) `deterministicFloorSatisfies`'s deterministic proof when nothing was dispatched. A dispatched watcher's *failure* still adds a block regardless. This is consistent with the plan's `assumption_delta_decision` ("the reviewer verdict... can only ever add a block, never supply the pass by itself") and is locked by `TestDeterministicFloorIsTheOnlySourceOfAPass`, but no test in this plan exercises the narrow "bound to watcher AND live-dispatched-and-passed" combination — flagged below for 193-02/193-03.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing test asserted the old (now-intentionally-changed) warning text**
- **Found during:** Task 1, after implementing D-01's plain-English warning change
- **Issue:** `TestRunCodexContinueVerificationWarnsWhenAllCommandsAreSkipped` (`cmd/codex_continue_test.go`, not in this plan's `files_modified`) asserted `strings.Contains(warnings, "no deterministic verification command")` — the exact old wording the plan's action item 6 directs replacing with "no tests to run in this project".
- **Fix:** Updated the assertion to check for the new plain-English text and to assert the warning no longer mentions "watcher"; updated the test's header comment to describe the D-01 contract instead of the old "hands verification to the watcher" behavior.
- **Files modified:** `cmd/codex_continue_test.go`
- **Verification:** `go test ./cmd -run 'TestRunCodexContinueVerificationWarnsWhenAllCommandsAreSkipped$' -count=1` — PASS
- **Committed in:** `623229e2` (part of Task 1's commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — pre-existing test text updated to match an intentional, plan-directed behavior change)
**Impact on plan:** Minimal — a one-file, two-assertion update outside the plan's declared `files_modified` list, required because the plan's own D-01 change altered text that file was pinned to.

## Issues Encountered

None. `go test ./cmd -count=1` (the full package, ~5,900+ test names) reports **zero failures** on the combined Task 1 + Task 2 state — no pre-existing test was left broken for plan 193-02 to inherit, and no build-side stage assertions in `cmd/codex_build_test.go` were touched or found failing (this plan did not modify `cmd/codex_build.go`/`cmd/codex_build_finalize.go`, so D-08's build-side changes remain 193-02's territory as planned).

`go test ./... -race` was run scoped to the changed/new test functions in `cmd` (`TestDeterministicFloor*`, `TestBothContinueLanesApplyTheSameFloor`, `TestZeroReviewerPhase*`, `TestZeroExecutedChecks*`, `TestDispatchedReviewerThatFailedStillBlocks`, `TestRunCodexContinueVerification*`) rather than the entire ~5,900-test `cmd` package under `-race`, which would have taken substantially longer than this plan's time budget; all scoped race runs passed (`ok github.com/calcosmic/Aether/cmd 57.637s`). The full non-race suite (`go test ./cmd -count=1`) did run to completion and passed.

## User Setup Required

None — no external service configuration required. This plan is entirely local Go code and tests.

## Next Phase Readiness

Ready for 193-02 and 193-03 (wave 2). Both build on `runDeterministicFloor`, `continueWatcherDecision`, `evaluateCriterionCheckDetail`, and `deterministicFloorSatisfies` exactly as named in this plan's frontmatter `key_links`. Flagged for those plans: add a test binding a criterion to `Checks:["watcher"]` with a live continue-time-dispatched-and-passed watcher, to confirm the narrow compatibility case noted in "Decisions Made" above still satisfies FLOOR-03's `required_checks: watcher` compatibility promise (D-06) the way the design intends.

---
*Phase: 193-free-checks-are-the-floor*
*Completed: 2026-08-22*

## Self-Check: PASSED

All created/modified files and both task commits (`623229e2`, `ee805ec1`) verified present.
