---
phase: 209-light-default-path
plan: 07
subsystem: cli-wrappers
tags: [ant-go, ant-quick, escalation, work-outcome, definition-of-done]

requires:
  - phase: 209-light-default-path
    provides: "209-02's escalation rule (jobSizeAttemptEscalationReason) and 209-05's real-session measurement that found the gap (defect register entry 61, 209-TIMING.md run 5)"
provides:
  - "A falsifiable test (TestEscalationNeedsAFailedCheckNotAnUnrunnableOne) that drives the real escalation decision over three cases -- genuine failure, unrunnable checks, file-count budget -- proven red on case 2 before the fix"
  - "jobSizeAttemptEscalationReason escalates only on a genuine check failure (WorkOutcomeBlocker); an unrunnable-checks verdict (WorkOutcomePartial) no longer folds into it"
affects: [210]

actuals:
  tokens: 1472
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Derive test facts through the same conversion function the runtime uses (smallAttemptFactsFromQuickResult over a result map shaped like runQuickJob's own, with the verdict itself derived through quickWorkVerdict) rather than hand-typing a struct the runtime could not produce."

key-files:
  created: []
  modified:
    - cmd/go_route.go
    - cmd/go_escalation_test.go

key-decisions:
  - "Narrowed the switch on colony.WorkOutcomeBlocker only (dropped WorkOutcomePartial from the escalation switch) rather than checking attempt.ChecksStatus == quickChecksFailed directly. The two are 1:1 in the real result map (quickWorkVerdict derives Blocker only from a non-passed, non-not-resolved status, and Partial only from the not-resolved status), so either reads correctly; the enum form keeps using colony.WorkOutcome as the single source of truth this file already committed to, and avoids introducing a status-string comparison alongside the existing verdict-based one."
  - "Kept the empty-status fallback (previously the bare literal \"unresolved\") but changed it to the existing quickChecksFailed constant, since the only way ChecksStatus arrives empty on a Blocker verdict is an unexpected/unrecognized check outcome, which quickWorkVerdict itself treats as failed by default."

patterns-established:
  - "None new -- this plan restores an existing project rule (the quick path's own passed/failed/not-resolved three-way status, cmd/command_truth.go:455) rather than establishing a new one."

requirements-completed: [UED-16]

coverage:
  - id: D1
    description: "A quick attempt whose checks genuinely FAILED still escalates to the planning route, unchanged."
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_genuinely_failed"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalatesWhenTheSmallAttemptProvesBigger/checks_failed"
        status: pass
    human_judgment: false
  - id: D2
    description: "A quick attempt whose checks could not be RUN at all does not escalate, and the reason sentence never claims the checks 'did not pass'."
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_could_not_be_run"
        status: pass
    human_judgment: false
  - id: D3
    description: "A quick attempt that changed more files than the small-job budget still escalates, on the file-count signal alone, regardless of check status."
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/file_count_over_budget_survives_unresolved_checks"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalatesWhenTheSmallAttemptProvesBigger/file_count_over_budget"
        status: pass
    human_judgment: false
  - id: D4
    description: "No new refusal, setting, or owner-facing question was added to the size router or the escalation."
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalationNeverAsksAndNeverBlocks"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 07: Escalate on a Failed Check, Not an Unrunnable One Summary

**Fixed `/ant-go`'s size escalation to stop treating "the project's checks could not be run" as "the project's checks failed" -- restoring the quick path's own passed/failed/not-checked three-way status instead of folding two of them together.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-24T20:10:00Z (approx.)
- **Completed:** 2026-09-24T20:22:14Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `TestEscalationNeedsAFailedCheckNotAnUnrunnableOne` to `cmd/go_escalation_test.go`, driving the real `jobSizeAttemptEscalationReason` decision over three cases with facts built through `smallAttemptFactsFromQuickResult` from a result map shaped like `runQuickJob`'s own (verdict itself derived via `quickWorkVerdict`, never hand-typed).
- Proved case 2 (checks could not be run) RED against the committed rule before touching `go_route.go` -- recorded below.
- Changed `jobSizeAttemptEscalationReason` (`cmd/go_route.go`) so only a genuine check failure (`colony.WorkOutcomeBlocker`) escalates; an unrunnable-checks verdict (`colony.WorkOutcomePartial`) no longer does. The file-count signal is untouched -- it is checked first and independently.

## The RED proof (Task 1, before Task 2's change)

Command:
```
go test ./cmd -count=1 -timeout 600s -run 'TestEscalationNeedsAFailedCheckNotAnUnrunnableOne' -v
```

Output:
```
=== RUN   TestEscalationNeedsAFailedCheckNotAnUnrunnableOne
=== RUN   TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_genuinely_failed
=== RUN   TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_could_not_be_run
    go_escalation_test.go:679: did not expect escalation when the checks could not be run at all, facts={FilesChanged:1 ChecksStatus:not_checked Verdict:partial}, reason="the project's own checks did not pass on the quick attempt (status: not_checked)"
=== RUN   TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/file_count_over_budget_survives_unresolved_checks
--- FAIL: TestEscalationNeedsAFailedCheckNotAnUnrunnableOne (0.00s)
    --- PASS: TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_genuinely_failed (0.00s)
    --- FAIL: TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/checks_could_not_be_run (0.00s)
    --- PASS: TestEscalationNeedsAFailedCheckNotAnUnrunnableOne/file_count_over_budget_survives_unresolved_checks (0.00s)
FAIL
FAIL	github.com/calcosmic/Aether/cmd	0.919s
```

Case 2 failed for exactly the right reason -- a `not_checked` verdict wrongly produced the "did not pass" escalation. Cases 1 and 3 already passed against the committed (pre-fix) rule, confirming they are the correct guard: the fix must not touch them.

## After Task 2's change

Same command, all three subtests pass. Full scoped verification (the plan's `<verification>` block, minus `TestGoRoute`/`TestGoSmallRoute`/`TestQuickVerdict`/`TestQuickUsesResolvedChecks` prefixes matching their real names):

```
go test ./cmd -count=1 -timeout 600s -run 'TestEscalationNeedsAFailedCheckNotAnUnrunnableOne|TestGoEscalation|TestGoBigRoute|TestGoRoute|TestGoSmallRoute|TestGoNeverRefusesTheOwner|TestQuickVerdict|TestQuickUsesResolvedChecks|TestGoPlanningHandoff|TestClassicCommandParity'
```
Result: `PASS`, `ok github.com/calcosmic/Aether/cmd 3.362s`. Every named test ran (`TestClassicCommandParity`, `TestGoSmallRouteRunsTheJobEndToEnd`, `TestGoNeverRefusesTheOwner`, `TestGoBigRouteLeavesPlanningReadyToRun`, `TestGoBigRouteTakesTheJobToPlanning` (both subtests), `TestGoEscalationReadsOnlyMeasuredEvidence`, `TestGoEscalationNeverAsksAndNeverBlocks` (all 7 subtests), `TestEscalationNeedsAFailedCheckNotAnUnrunnableOne` (all 3 subtests), `TestGoPlanningHandoffCanActuallyStartPlanning` (all 4 subtests), `TestGoRouteIsComputedNotConstant`, `TestGoRouteIgnoresHowTheSentenceIsWorded`, `TestGoRouteDirectoryOverBudgetEscalates`, `TestGoRouteAcceptedPlanWithOutstandingWorkAlwaysEscalates`, `TestGoRouteHasOneAuthority` (both subtests), `TestQuickVerdictFollowsTheProjectsOwnChecks` (all 3 subtests), `TestQuickUsesResolvedChecksNotGoOnly`) -- none skipped.

Also ran the adjacent existing suite not in the plan's literal `<verification>` string but exercising the same function, as an extra guard: `TestGoEscalatesWhenTheSmallAttemptProvesBigger` (file count over budget, checks failed, stays small -- all pass) and `TestGoNeverMovesAJobBackDown` (pass).

`go build ./cmd/aether` and `go vet ./cmd` both clean before each commit.

## Task Commits

Each task was committed atomically:

1. **Task 1: A check that fails on today's rule** - `f9ef68f6` (test)
2. **Task 2: Escalate on a failed check, not an unrunnable one** - `9b2135c5` (fix)

_Note: this plan's docs/metadata commit follows this SUMMARY's own commit._

## Files Created/Modified

- `cmd/go_escalation_test.go` - Added `TestEscalationNeedsAFailedCheckNotAnUnrunnableOne` (3 subtests: genuine failure, unrunnable checks, file-count-over-budget-survives)
- `cmd/go_route.go` - `jobSizeAttemptEscalationReason` now escalates on `colony.WorkOutcomeBlocker` only, not `colony.WorkOutcomePartial`; empty-status fallback uses `quickChecksFailed` constant instead of a bare `"unresolved"` literal; doc comment explains the restored distinction and names entry 61

## Decisions Made

- Checked `attempt.Verdict == colony.WorkOutcomeBlocker` (removing `colony.WorkOutcomePartial` from the switch) rather than comparing `attempt.ChecksStatus` against the status constants directly. In the real result map the two are 1:1 (`quickWorkVerdict` derives `Blocker` only from a status that is neither `passed` nor `not_checked`, and `Partial` only from `not_checked`), so both forms are equivalent in practice; the enum form was chosen to keep `colony.WorkOutcome` as the one signal this function already reads, without adding a second, string-based comparison alongside it.
- No bare `"not_checked"` literal was introduced anywhere in `cmd/go_route.go` (confirmed by grep) -- the fix is expressed purely in terms of the existing `colony.WorkOutcome` enum and the `quickChecksFailed` constant.

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched the plan's `<action>` and `<verify>` blocks precisely; no auto-fixes, no architectural questions, no scope creep.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Defect register entry 61 can be closed on this evidence: a quick job in a project with no check command configured now completes and is reported honestly as unchecked, without a spurious "bigger than it looked" escalation. A genuinely failed check, and a job that changed too many files, both still escalate exactly as before.
- Phase 209 (A Light Default Path) now has plans 01-07 all summarized. Per STATE.md, phase 209 was already at "GAP CLOSURE PLAN 06 EXECUTED" with re-verification pending before advancing to Phase 210; this plan closes gap-closure work for entry 61 on top of that. Re-verify phase 209 as a whole (209-VERIFICATION.md) before advancing to Phase 210.
- No blockers.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*
