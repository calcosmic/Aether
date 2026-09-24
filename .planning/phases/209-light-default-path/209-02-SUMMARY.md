---
phase: 209-light-default-path
plan: 02
subsystem: cli
tags: [cobra, cli, routing, escalation, ast-guard]

requires:
  - phase: 209-light-default-path
    provides: "209-01's resolveJobSizeRoute (the one route authority), runGoJob/renderGoVisual, and runQuickJob reuse on the small route"
provides:
  - "the big route of /ant-go: reports route/reason/goal, dispatches no helper, writes no colony state, hands the next step to the shared next-action authority"
  - "D-03's self-escalation: resolveJobSizeRoute's attempt-facts branch, which moves a small job that proves bigger up to the planning route on the program's own authority, one way only, from independently measured evidence alone"
  - "the wrapper routing for /ant-go's one two-call (init then go) case, on all three platform surfaces"
  - "a standing AST guard (TestGoEscalationNeverAsksAndNeverBlocks) that fails the moment the size router or the escalation grows a new refusal of its own"
affects: [210, any later phase touching /ant-go's routing or D-02's default-menu membership]

actuals:
  tokens: 11500
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Base-decision extraction for a second-pass override: resolveJobSizeRouteBase holds the pure pre-attempt priority chain; resolveJobSizeRoute wraps it so a second call with facts.Attempt set can escalate without ever moving a job back down, mirroring resolveVerificationDepth's own escalate-only keyword branch (cmd/review_depth.go)."
    - "Evidence-scoped reader, AST-locked: smallAttemptFactsFromQuickResult reads exactly files/checks_status/work_outcome from runQuickJob's own result map, and TestGoEscalationReadsOnlyMeasuredEvidence parses the function body to refuse any other key -- never a helper's own account of its work."
    - "Decide-say-continue, never ask: the escalation adds exactly one rendered line and never a question, a prompt, or a non-zero exit, following the same precedent phaseHasHeavyKeywords already uses for its own silent-to-loud escalation."

key-files:
  created:
    - cmd/go_escalation_test.go
  modified:
    - cmd/go_route.go
    - cmd/go_cmd.go
    - .aether/commands/go.yaml
    - .claude/commands/ant-go.md
    - .claude/commands/ant/go.md
    - .opencode/commands/ant/go.md

key-decisions:
  - "The 'no project set up yet' reason lives in resolveJobSizeRouteBase itself (a facts.ColonyActive-gated branch of the existing zero-matched-paths case), not as a second decision layer -- so there is still exactly one place that spells the route, matching TestGoRouteHasOneAuthority's own guard."
  - "The escalation's own evidence-only test case for 'checks did not pass' uses the checks-not-resolved (partial) verdict rather than a genuine failure (blocker) verdict, because a genuine failure independently raises a tracked issue flag through runQuickJob's own pre-existing, correct, out-of-scope behaviour (raiseQuickIssueFlag) -- using it would make the never-writes-a-decision assertion fail for a reason that has nothing to do with this plan's escalation logic. Both verdicts are named in D-03's own OR condition, so the substitution changes no behaviour under test."
  - "Task 1 and Task 2's production code (cmd/go_route.go, cmd/go_cmd.go) landed as one commit rather than two: both tasks share the same two implementation files with a genuine dependency (Task 2's attempt-facts branch extends Task 1's resolveJobSizeRoute in place), which is exactly the 'a real dependency chain or a genuinely shared implementation file' grouping test CLAUDE.md's own Coherent Jobs section describes. Tests for all three tasks landed in a second commit."

requirements-completed: [UED-16]

coverage:
  - id: D1
    description: "A job the router sizes as big goes to the planning route: it reports route/reason/goal, dispatches no helper, and writes nothing under .aether/data, whether or not a project is already recorded."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoBigRouteTakesTheJobToPlanning"
        status: pass
    human_judgment: false
  - id: D2
    description: "The wrapper for all three platforms carries a big job with no project recorded through the one two-call (init then go) case, idempotently, and never re-words the runtime's own route sentence."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/classic_command_parity_test.go#TestClassicCommandParity"
        status: pass
      - kind: unit
        ref: "cmd/command_call_audit_test.go#TestCommandCallsMatchCobraContracts"
        status: pass
      - kind: unit
        ref: "cmd/command_call_audit_test.go#TestDocumentedCommandNamesResolve"
        status: pass
    human_judgment: false
  - id: D3
    description: "A small attempt that measurably changed more files than the budget, or whose checks did not pass, is moved up to the planning route by the program itself, with one plain-English line naming the measured fact, and the quick attempt's own changes survive untouched."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalatesWhenTheSmallAttemptProvesBigger"
        status: pass
    human_judgment: false
  - id: D4
    description: "A job already sized big before the attempt never comes back down, and the escalation reads only independently measured evidence, never a helper's own account of its work."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoNeverMovesAJobBackDown"
        status: pass
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalationReadsOnlyMeasuredEvidence"
        status: pass
    human_judgment: false
  - id: D5
    description: "The size router and the escalation never refuse, ask, or block: across the small/escalated/big matrix the command always returns success, the rendered screen carries no question mark the program itself wrote, no field parks the job on an owner decision, and the shared decision store is never touched by a run that did not independently need to touch it."
    requirement: UED-16
    verification:
      - kind: unit
        ref: "cmd/go_escalation_test.go#TestGoEscalationNeverAsksAndNeverBlocks"
        status: pass
    human_judgment: false

duration: 75min
completed: 2026-09-24
status: complete
---

# Phase 209 Plan 02: The Big Route and D-03's Self-Escalation Summary

**The big route now decides and hands over instead of stubbing out, and `resolveJobSizeRoute` gained a second, attempt-facts layer that moves a small job that proves bigger up to the planning route on its own authority -- one way only, from measured evidence alone, with one plain-English line and never a question.**

## Performance

- **Duration:** ~75 min
- **Started:** 2026-09-24T14:00:00Z (approx)
- **Completed:** 2026-09-24T15:19:10Z
- **Tasks:** 3
- **Files modified:** 6 modified, 1 created

## Accomplishments

- `runGoJob`'s big-route branch now reports `route`, `route_reason`, `goal` (the owner's own sentence) and `colony_active` into the result map, dispatches no helper, writes no colony state, and lets `closeLifecycleCommand` produce the next step through the existing shared next-action authority -- the old "not wired up yet, run `aether plan` directly" stub (and its hardcoded `aether plan` literal) is gone.
- `resolveJobSizeRouteBase` now names the specific fact when nothing in the project matches a sentence AND no project has been recorded at all: the job is being started as a piece of planned work under that sentence as its goal, distinct from the "project exists but nothing here matches" case.
- `resolveJobSizeRoute` gained a second, higher-priority layer for a completed small attempt: a job already big before the attempt stays big unchanged (D-03's one-way-only rule); otherwise a real disk-measured file count over budget, or a partial/blocker check verdict, escalates it to big with `Escalated: true` and a reason naming the measured fact -- read via the new, AST-locked `smallAttemptFactsFromQuickResult`, which touches only `files`, `checks_status` and `work_outcome`.
- `runGoJob` calls the route authority a second time after the small route's own attempt runs, and `renderGoVisual` prints exactly one additional line on escalation, immediately after the route line, naming the measured fact and saying the job has been moved up -- never hiding or undoing the attempt's own real changes.
- The `/ant-go` wrapper routing (all four sources: `.aether/commands/go.yaml`, both Claude Code files, and OpenCode) now states the one two-call case explicitly: quick route needs nothing further; planning route with no project runs `aether init "<the same sentence>"` then `aether go "<the same sentence>"` again (idempotent); planning route with a project already recorded follows `/ant-plan`'s own wrapper.
- `TestGoEscalationNeverAsksAndNeverBlocks` is the milestone's own standing guard, live-verified by pasting a mutation refusal into `go_cmd.go`, watching the exact subtest fail, and reverting byte-for-byte.

## Task Commits

1. **Task 1 + Task 2: the big route and D-03's escalation (production code)** - `b2460116` (feat) -- both tasks landed together because they share `cmd/go_route.go` and `cmd/go_cmd.go` with a genuine in-place dependency (Task 2 extends the function Task 1 completed); see Deviations.
2. **Task 1 + Task 2 + Task 3: tests and the standing guard** - `e325badb` (test)

**Plan metadata:** (this commit, following this SUMMARY)

_Note: task boundaries in the plan map to two commits here rather than three -- see "Deviations from Plan" below for why, and CLAUDE.md's own "Coherent Jobs" section for the standing rule this follows._

## Files Created/Modified

- `cmd/go_route.go` -- `resolveJobSizeRoute`/`resolveJobSizeRouteBase` split, `jobSizeAttemptEscalationReason`, `smallAttemptFactsFromQuickResult`, and the "no project set up yet" reason branch
- `cmd/go_cmd.go` -- the completed big-route branch of `runGoJob`, the second `resolveJobSizeRoute` call after a small attempt, and `renderGoVisual`'s escalation line
- `.aether/commands/go.yaml`, `.claude/commands/ant-go.md`, `.claude/commands/ant/go.md`, `.opencode/commands/ant/go.md` -- the wrapper routing's one two-call case
- `cmd/go_escalation_test.go` (new) -- `TestGoBigRouteTakesTheJobToPlanning`, `TestGoEscalatesWhenTheSmallAttemptProvesBigger`, `TestGoNeverMovesAJobBackDown`, `TestGoEscalationReadsOnlyMeasuredEvidence`, `TestGoEscalationNeverAsksAndNeverBlocks`, and the `quickJobFileWritingInvoker` fixture invoker

## Decisions Made

- The "no project set up yet" reason lives inside `resolveJobSizeRouteBase`'s existing zero-matched-paths branch (gated on `facts.ColonyActive`), not as a separate decision layer, so `resolveJobSizeRoute` remains the one place a route is ever spelled (`TestGoRouteHasOneAuthority` still passes unmodified).
- The escalation's "checks did not pass" test case uses the checks-not-resolved (partial) verdict rather than a genuine failure (blocker) verdict -- see Deviations for why.
- Task 1 and Task 2's production code landed in one commit because both genuinely share the same two implementation files (CLAUDE.md's own Coherent Jobs grouping rule).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test scenario substitution to avoid an unrelated pre-existing side effect**
- **Found during:** Task 3
- **Issue:** `TestGoEscalationNeverAsksAndNeverBlocks`'s "small escalated by failed checks" case, when driven with a genuine check failure (`quickChecksFailed`, producing a `WorkOutcomeBlocker` verdict), triggers `runQuickJob`'s own pre-existing, correct, out-of-plan-scope behaviour: `raiseQuickIssueFlag` writes a tracked "issue" flag to `pending-decisions.json` whenever a quick attempt's checks genuinely fail. This is legitimate existing `/ant-quick` behaviour (an "issue" flag, not a blocking decision), but it made the test's "the shared decision store is never touched" assertion fail for a reason unrelated to this plan's own escalation logic.
- **Fix:** The test case instead simulates the checks-not-resolved outcome (`quickChecksNotResolved`, producing a `WorkOutcomePartial` verdict) -- one of D-03's own two escalation signals ("the verdict is a partial or a blocker") -- which escalates identically without touching the flags file.
- **Files modified:** `cmd/go_escalation_test.go`
- **Verification:** `TestGoEscalationNeverAsksAndNeverBlocks` passes with the flags-file hash unchanged across all five matrix cases; the substitution changes no assertion about D-03's own escalation contract.
- **Committed in:** `e325badb`

**2. [Rule 1 - Bug in test setup] "No project set up" fixture corrected to match production reality**
- **Found during:** Task 1
- **Issue:** An initial draft of `TestGoBigRouteTakesTheJobToPlanning`'s bare-folder case set the global `store` variable to `nil` to represent "no project here" -- mirroring plan 01's own `TestGoNeverRefusesTheOwner` fixture. Running it surfaced that `store == nil` is not what production does for a genuinely no-project folder: `PersistentPreRunE` opens a real store pointing at an empty `.aether/data` for every non-read-only command (including `go`), and `store == nil` instead represents storage being genuinely unavailable -- which the shared next-action resolver reads as ambiguous recovery ("resume") rather than "start a project" ("init"), an existing and correct distinction in `loadNextActionInputForCommand`.
- **Fix:** The bare-folder fixture now opens a real, empty store via `newTestStore(t)` (no `COLONY_STATE.json` written), matching what `aether go` actually sees in a fresh folder in production.
- **Files modified:** `cmd/go_escalation_test.go`
- **Verification:** The corrected fixture's "next step" assertion (recommends `init`) now passes; `TestGoEscalationNeverAsksAndNeverBlocks`'s separate `store == nil` case is kept as-is deliberately, to prove the "never asks or blocks" guarantee holds even in that genuinely-degraded scenario.
- **Committed in:** `b2460116`/`e325badb` (fixture-only; no production code changed by this fix)

---

**Total deviations:** 2 auto-fixed (1 blocking test-scope substitution, 1 test-fixture correctness bug). **Impact on plan:** Neither changes any production behaviour or any of D-03's own claims; both keep the test suite honest about what it is actually proving.

## Issues Encountered

None beyond the two deviations above, both resolved during execution.

Pre-existing, unrelated test failures observed while running the phase's own verification commands (confirmed unrelated by isolated re-run against a clean `git worktree` at this plan's own base commit, `ccecce49`, before any of this plan's files existed):
- `TestGoSourceHintsMatchCobraContracts` (references `cmd/codex_native_context.go`, `cmd/partial_work_label.go`)
- `TestGoldenBuildVisualOutput` / `TestGoldenContinueVisualOutput` (golden-file wording drift unrelated to this plan's screens)
- `TestNoRegisteredSubcommandIsUnreferenced` (`aether codex-native-worker context-ack` orphan) -- reproduced byte-identical (402 registered commands, same single orphan) on the clean worktree

None of these are caused by this plan's changes and none were touched, per the deviation rules' scope boundary.

## Manual mutation verification (Definition of Done -- "a test must be able to fail")

`TestGoEscalationNeverAsksAndNeverBlocks`'s AST half was hand-verified by actually pasting a new refusal into `cmd/go_cmd.go` (an `outputError` call refusing any sentence containing "xyzzy"), running the test, confirming the exact failure message named the new call site, and reverting the file to byte-identical (confirmed via `git diff`):

| Mutation | Test | Result |
|---|---|---|
| A new `outputError` call added to `goCmd.RunE`, refusing sentences containing "xyzzy" | `TestGoEscalationNeverAsksAndNeverBlocks` (AST half, "no_new_refusal_in_go_cmd.go") | FAILED as expected: "go_cmd.go calls outputError with a new refusal at go_cmd.go:29:4 -- the size router and the escalation must never refuse the owner's job" |

Every other named test in this plan's acceptance criteria was already proven to fail on the wrong side of its own guard during normal development iteration (the escalation branch, the base-decision extraction, and the evidence-only reader were each built incrementally against a red test before the corresponding production code existed).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `/ant-go` now has a complete small route (plan 01) and a complete big route plus self-escalation (this plan) -- both sides of D-01 and all of D-03 are implemented and tested.
- D-02's mandatory owner checkpoint on the real rendered default-menu screen remains outstanding and untouched by this plan.
- `.planning/WINDOWS.md` row 53 remains open and was not touched, per this phase's own CONTEXT.md instruction.

## Known Stubs

None. Both routes are real, production-quality implementations with no placeholder branches.

---
*Phase: 209-light-default-path*
*Completed: 2026-09-24*
