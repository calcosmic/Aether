---
phase: 201-queen-led-work-cycle
plan: "20"
subsystem: infra
tags: [go, cli, cobra, lifecycle, autopilot, closeout]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "buildWorkCloseoutDetails, LifecycleCloseoutDetails.WorkOutcome wiring, and the body/wrapper closeout split (plan 201-19) -- the shared closeout machinery this plan extends to the check lane and the autopilot lane"
provides:
  - "checkWorkOutcome (cmd/work_closeout.go): the check's one derivation of its own verdict from continueAcceptVerifyAdvanceDecision plus the verification report, total over the decision's two verdict values and reachable to all six declared work verdicts"
  - "checkWorkOutcomeResultKey / checkWorkOutcomeFromResult: the one new stable result-map key codex_continue.go stores its derived verdict under at all three places it saves a continue result, and the dual-typed reader that reads it back"
  - "checkWorkCloseoutDetails (cmd/work_closeout.go): reads the already-stored verdict and verification report off a continue result -- never recomputes -- and builds a verdict-carrying LifecycleCloseoutDetails with the check's own evidence and blockers"
  - "applyCheckWorkCloseout: the one shared render-time fold both continueCmd's ending screens (codex_workflow_cmds.go) and the wrapper's chat-path check closeout (ceremony_cmd.go) use, falling back to today's exact rendering when no verdict resolves"
  - "autopilotTerminalWorkOutcome + applyAutopilotTerminalCloseout (cmd/compatibility_cmds.go): total derivation of the autopilot run's terminal verdict from its own recorded trigger code (reusing the existing 201-11 stop-boundary pair) or dry-run flag, wired into runCompatibilityCmd's terminal closeout so `aether run` finally ends with a cost-and-time block"
  - "The autopilot's per-phase build closeout now calls buildWorkCloseoutDetails directly, so it can never disagree with the build lane's own card for the same sealed attempt"
  - "TestCheckWorkOutcome, TestCheckAndAutopilotCloseoutsCarryAVerdict, TestAutopilotTerminalVerdictCoversEveryStopCode, TestEveryWorkLaneCloseoutHasProductionCallers: unit, end-to-end, and AST call-graph proof that the check lane and the autopilot lane now carry a verdict, a recommendation, and exactly one cost block"
affects: [201-queen-led-work-cycle]

# Actuals (#2632)
actuals:
  tokens: 14242
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Verdict derived once, stored on the result map, read back at render time: checkWorkOutcome is called exactly once per continueAcceptVerifyAdvanceDecision a continue lane obtains (twice in cmd/codex_continue.go, once per decision -- preReviewDecision and finalDecision -- because only one of them is ever the decision that actually produces the reached result), and the resulting verdict is stored under one stable key; checkWorkCloseoutDetails only ever reads that key back, never recomputes it, matching the plan's explicit 'derive it once where the decision is made' rule"
    - "Named closeout-fold helpers as call-graph-provable seams: applyCheckWorkCloseout and applyAutopilotTerminalCloseout exist specifically so an AST direct-caller guard can prove a resolver is reachable from an anonymous cobra RunE closure, which a package-level func-declaration scan cannot see directly but CAN see one hop away through a named helper"

key-files:
  created:
    - cmd/work_closeout_lanes_test.go
  modified:
    - cmd/work_closeout.go
    - cmd/codex_continue.go
    - cmd/work_closeout_test.go
    - cmd/codex_workflow_cmds.go
    - cmd/ceremony_cmd.go
    - cmd/compatibility_cmds.go

key-decisions:
  - "checkWorkOutcome's total mapping over continueAdvanceVerdict's two values resolves to all SIX declared verdicts (not five), by treating a BLOCKED decision with zero executed verification steps as interrupted (nothing ever ran to produce a blocking reason, so the check itself never got underway) rather than folding it into blocker. This was necessary to satisfy Task 1's acceptance criterion that every one of the six verdicts be reachable from checkWorkOutcome, since the plan's five behavior bullets only explicitly named five outcomes; the sixth (interrupted) is a reasonable, symmetrical extension of the same 'nothing to verify' carve-out the plan's own no-change bullet already applies on the advancing side."
  - "cmd/codex_continue_finalize.go (the external/wrapper 'continue-finalize' lane behind --classic-ceremony) was deliberately NOT wired to store the check verdict. It is not in the plan's declared files_modified list, its blocked-before-review path never reaches the shared runContinueAcceptVerifyAdvance decision body at all (an existing, pre-201-20 architectural gap distinct from what this plan closes), and wiring it would require restructuring that path rather than adding a call site. checkWorkCloseoutDetails/ceremonyContinueRawResult correctly fall back to today's exact rendering for a raw map from this lane (no stored verdict key), so nothing regresses -- but the heavy/classic-ceremony check closeout does not yet carry a verdict, only the default/fast `aether continue` (the documented primary daily-driver path, .claude/commands/ant/continue.md line 59) and the autopilot's own check calls (which reuse the SAME direct lane via runAutopilotContinue = runCodexContinue) do."
  - "The autopilot's per-phase build closeout and terminal run closeout were extracted into two small named functions (applyAutopilotTerminalCloseout is new; the per-phase site now calls buildWorkCloseoutDetails inline) specifically so an AST direct-caller guard can prove each resolver has a production caller -- runCompatibilityCmd's RunE is an anonymous cobra closure invisible to a package-level FuncDecl scan, so the call had to be one hop away through a named function for TestEveryWorkLaneCloseoutHasProductionCallers's blanket 'at least one caller' check to pass without weakening the guard."
  - "autopilotTerminalWorkOutcome treats autopilotTriggerColonyNotRunnable (a Stop-disposition trigger outside the four declared stop-boundary groups from 201-11) as a blocker, and autopilotTriggerReplanDue (a queued checkpoint, not a failure) as partial, matching autopilotTriggerMaxPhasesReached's own verdict -- both are extensions beyond the plan's five explicitly named cases, needed to make the mapping genuinely total over every trigger code finishAutopilotInvocation can be called with (confirmed by reading its own code: it is called for stop, pause, AND queue_and_continue dispositions alike, never only for the five the plan named)."
  - "cmd/autopilot_policy.go (where the plan's action text suggested placing the new derivation function, 'next to the existing autopilot policy helpers') was NOT modified; the function lives in cmd/compatibility_cmds.go instead, a file the plan does declare, since it only needed to CALL the existing autopilotStopBoundaryForTriggerCode/autopilotStopBoundaryWorkOutcome pair, not extend their own declared boundary catalogue."
  - "buildRunDryRunResult's finish closure now sets current_phase (working.CurrentPhase) alongside current_state -- it previously set neither, so a dry-run's no-change verdict would have resolved a phase of 0 and the cost-and-time block would have silently rendered empty, per the plan's own explicit 'ensure the run result carries a resolvable phase number' instruction."

requirements-completed: [WORK-04, WORK-05]

coverage:
  - id: D1
    description: "checkWorkOutcome derives the check's verdict once per decision, total over the decision's two verdict values, reachable to all six declared verdicts, and codex_continue.go stores it on the result map at all three places it saves a continue result"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd -run TestCheckWorkOutcome"
        status: pass
      - kind: unit
        ref: "cmd -run TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real advancing check and a real blocked check each end with the resolved verdict label, a recommended next action (command, reason, alternatives), and exactly one cost-and-time block, on both continueCmd's direct lane and the wrapper's chat-path closeout; a verdict-free closeout shows no cost block"
    requirement: "WORK-05"
    verification:
      - kind: integration
        ref: "cmd -run TestCheckAndAutopilotCloseoutsCarryAVerdict"
        status: pass
    human_judgment: false
  - id: D3
    description: "The autopilot's per-phase build closeout verdict equals buildWorkCloseoutDetails' own verdict for the same sealed attempt, and the terminal run closeout carries a verdict derived from the run's own recorded stop code or dry-run flag, total over every declared trigger code"
    requirement: "WORK-05"
    verification:
      - kind: integration
        ref: "cmd -run TestCheckAndAutopilotCloseoutsCarryAVerdict"
        status: pass
      - kind: unit
        ref: "cmd -run TestAutopilotTerminalVerdictCoversEveryStopCode"
        status: pass
    human_judgment: false
  - id: D4
    description: "buildWorkCloseoutDetails, checkWorkCloseoutDetails, and autopilotTerminalWorkOutcome each have at least one production caller; the build render path, the check render path (both lanes), and both autopilot closeout sites reach their own resolver; a synthetic disconnection is refused by name"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd -run TestEveryWorkLaneCloseoutHasProductionCallers"
        status: pass
    human_judgment: false

duration: 70min
completed: 2026-09-11
status: complete
---

# Phase 201 Plan 20: Give the Check Step and the Autopilot the Same Honest Closeout the Build Now Has Summary

**The check's verdict is now derived once at the shared accept/verify/advance decision and carried onto every screen that renders it; the autopilot's per-phase card now reuses the build lane's own verdict instead of a generic literal; and `aether run`'s terminal screen finally ends with a cost-and-time block, derived from the run's own recorded stop code rather than never being set at all.**

## Performance

- **Duration:** 70 min
- **Started:** 2026-09-11 (continuing from 201-19)
- **Completed:** 2026-09-11
- **Tasks:** 3 completed
- **Files modified:** 6 modified, 1 created

## Accomplishments

- `checkWorkOutcome` (`cmd/work_closeout.go`, new): the one derivation of the check's own verdict, called once per `continueAcceptVerifyAdvanceDecision` a continue lane obtains, from recorded facts only (the decision's verdict, its `PartialSuccess` flag, the verification report's executed-step count, and whether any step timed out). Total over the decision's two verdict values and reaches all six declared work verdicts (advance: no-change / partial / success; block: timeout / interrupted / blocker), proven by `TestCheckWorkOutcome`.
- `cmd/codex_continue.go` now stores that verdict under one new stable key (`check_work_outcome`) alongside the existing `verification` key at all three places it saves a continue result -- the gate-blocked-before-review path, the review-blocked path, and the advancing path -- computed once per decision (`preReviewWorkOutcome`, `finalWorkOutcome`) and reused across the paths that share a decision.
- `checkWorkCloseoutDetails` and `applyCheckWorkCloseout` (`cmd/work_closeout.go`, new): read the already-stored verdict and verification report back off a continue result -- never recompute -- and fold them into the shared closeout ceremony (verdict, recommended next action, evidence, blockers, one cost-and-time block). Wired into both of `continueCmd`'s ending screens (`cmd/codex_workflow_cmds.go`) and the wrapper's chat-path check closeout (`cmd/ceremony_cmd.go`, via a new `ceremonyContinueRawResult` helper mirroring `closeoutContinueDirectVisual`'s own raw-map resolution), each falling back to today's exact rendering (including the plain cost line) when no verdict resolves.
- `autopilotTerminalWorkOutcome` and `applyAutopilotTerminalCloseout` (`cmd/compatibility_cmds.go`, new): derive the autopilot run's terminal verdict from its own recorded trigger code -- reusing the existing `autopilotStopBoundaryForTriggerCode`/`autopilotStopBoundaryWorkOutcome` pair from plan 201-11 for the four declared stop boundaries, plus direct handling for `colony_not_runnable` (blocker), `replan_due` (partial, a queued checkpoint rather than a failure), `cancelled` (interrupted), `worker_timeout` (timeout), and `colony_complete` (success) -- or the no-change verdict for a dry-run invocation. Total over every one of the 15 declared trigger codes, proven by `TestAutopilotTerminalVerdictCoversEveryStopCode` (iterating the real catalogue, both typed and JSON-round-tripped). Wired into `runCompatibilityCmd`'s terminal closeout, so `aether run` now ends with a cost-and-time block where before it carried none.
- The autopilot's per-phase build closeout (`cmd/compatibility_cmds.go`) now calls `buildWorkCloseoutDetails` directly instead of a generic summary literal, so the autopilot's per-phase card and the build lane's own card can never disagree about the same sealed attempt's verdict.
- `buildRunDryRunResult`'s `finish` closure now sets `current_phase` so the cost-and-time block has a resolvable phase to read for a dry-run's no-change verdict, rather than silently rendering empty.
- `cmd/work_closeout_lanes_test.go` (new): `TestCheckAndAutopilotCloseoutsCarryAVerdict` drives a real advancing check, a real blocked check (a genuinely failing documented verification command), and a real autopilot dry-run invocation, asserting each finished screen carries the resolved verdict label, the recommended command/reason/alternatives, and exactly one cost-and-time block heading -- plus a negative case proving a verdict-free closeout renders no cost block. `TestAutopilotTerminalVerdictCoversEveryStopCode` iterates the real trigger-code catalogue. `TestEveryWorkLaneCloseoutHasProductionCallers` is an AST call-graph guard (reusing `continueDecisionPackageFuncs`/`continueDecisionDirectCallers` from plan 201-02, and `workCloseoutFindCobraRunE`/`workCloseoutBodyCallsTarget` from plan 201-19) proving all three resolvers have production callers, that the build render path, the check render path (both lanes), and both autopilot closeout sites reach them, with a synthetic-removal sub-case proving the guard can fail.

## Task Commits

Each task was committed atomically:

1. **Task 1: Derive the check verdict once at the shared decision and carry it** - `761da85f` (feat)
2. **Task 2: Give the check screen and both autopilot closeouts a verdict and a cost block** - `1f657f23` (feat)
3. **Task 3: Prove all three lanes end with a verdict and one cost block** - `62a16b50` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/work_closeout.go` - `checkWorkOutcome`, `checkWorkOutcomeResultKey`, `checkWorkOutcomeFromResult`, `checkVerificationEvidenceFromValue`, `checkBlockersFromResult`, `checkWorkOutcomeSummary`, `checkWorkCloseoutDetails`, `applyCheckWorkCloseout`.
- `cmd/codex_continue.go` - Computes `preReviewWorkOutcome`/`finalWorkOutcome` via `checkWorkOutcome` right after each `continueAcceptVerifyAdvanceDecision` is obtained; stores the result under `checkWorkOutcomeResultKey` at all three result-map construction sites.
- `cmd/work_closeout_test.go` - `TestCheckWorkOutcome` (new).
- `cmd/codex_workflow_cmds.go` - `continueCmd`'s two ending screens now build the body then call `applyCheckWorkCloseout` instead of unconditionally calling `appendSpendCostLine`.
- `cmd/ceremony_cmd.go` - `renderCeremonyCloseout`'s `workflow == "continue"` branch resolves `checkWorkCloseoutDetails` against the same raw completion map `closeoutContinueDirectVisual` rendered from, applying and rendering the verdict-carrying closeout when one resolves; new `ceremonyContinueRawResult` helper.
- `cmd/compatibility_cmds.go` - New `autopilotTriggerCodeFromResult`, `autopilotTerminalWorkOutcome`, `applyAutopilotTerminalCloseout`; the per-phase build closeout now calls `buildWorkCloseoutDetails`; the terminal run closeout calls `applyAutopilotTerminalCloseout`; `buildRunDryRunResult`'s `finish` closure now sets `current_phase`.
- `cmd/work_closeout_lanes_test.go` (new) - `TestCheckAndAutopilotCloseoutsCarryAVerdict`, `TestAutopilotTerminalVerdictCoversEveryStopCode`, `TestEveryWorkLaneCloseoutHasProductionCallers`.

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: `cmd/codex_continue_finalize.go` (the external/wrapper "continue-finalize" lane behind `--classic-ceremony`) was deliberately left unwired -- it is not in this plan's declared file list, its pre-review blocked path never reaches the shared decision body at all (a distinct, pre-existing architectural gap), and the resolvers this plan built gracefully fall back to today's exact rendering for its output. This means the default/fast `aether continue` path (the documented daily-driver flow) and the autopilot's own check calls (which reuse that same direct lane) both carry the new verdict, recommendation, and cost block; the heavy/classic-ceremony wrapper path does not yet.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing functionality] `checkWorkOutcome`'s mapping needed a sixth branch to satisfy "every verdict reachable"**
- **Found during:** Task 1
- **Issue:** The plan's five behavior bullets for `checkWorkOutcome` only explicitly named five outcomes (success, partial, timeout, blocker, no-change), but Task 1's own acceptance criteria required all six declared verdicts to be reachable.
- **Fix:** Added a symmetrical carve-out on the block side: a blocked decision with zero executed verification steps resolves to interrupted (nothing ever ran, so the check itself never got underway) rather than folding into blocker.
- **Files modified:** `cmd/work_closeout.go`
- **Commit:** `761da85f`

**2. [Rule 2 - Missing functionality] `autopilotTerminalWorkOutcome` needed to cover two trigger codes the plan did not name**
- **Found during:** Task 2
- **Issue:** The plan named five explicit trigger-code cases (the four boundary-mapped groups plus cancelled/worker_timeout/max_phases/colony_complete), but `finishAutopilotInvocation` (the sole caller) can be reached with any of the 15 declared trigger codes, including `colony_not_runnable` (a Stop-disposition trigger outside the four boundary groups) and `replan_due` (a queued checkpoint). Task 2's own acceptance criterion required every terminating trigger code to map to a declared verdict.
- **Fix:** Mapped `colony_not_runnable` to blocker (itself a "could not get past" condition) and `replan_due` to partial (matching `max_phases_reached`'s own verdict, since a queued replan is a natural pause rather than a failure).
- **Files modified:** `cmd/compatibility_cmds.go`
- **Commit:** `1f657f23`

**3. [Rule 1 - Bug] A dry-run autopilot result carried no `current_phase`, so its cost block would have silently rendered empty**
- **Found during:** Task 2
- **Issue:** `buildRunDryRunResult`'s `finish` closure set `current_state` but never `current_phase`; since a dry-run resolves to the no-change verdict (a real, non-nil `WorkOutcome`), `appendLifecycleCloseoutSpendLine`'s gate would pass but the phase it resolved (`intValue(result["current_phase"])`) would be 0, and the cost block would render with nothing to show.
- **Fix:** Added `"current_phase": working.CurrentPhase` to the `finish` closure's map literal.
- **Files modified:** `cmd/compatibility_cmds.go`
- **Commit:** `1f657f23`

**4. [Rule 1 - Bug] `autopilotTerminalWorkOutcome` was unreachable by the call-graph guard because its only caller was an anonymous cobra closure**
- **Found during:** Task 3
- **Issue:** `TestEveryWorkLaneCloseoutHasProductionCallers`'s blanket "at least one production caller" check failed for `autopilotTerminalWorkOutcome`: it was called only from `runCompatibilityCmd`'s `RunE`, an anonymous closure assigned inside a package-level `var ... = &cobra.Command{...}` literal, which `continueDecisionPackageFuncs` (indexing only top-level `*ast.FuncDecl` nodes) cannot see as a caller.
- **Fix:** Extracted the terminal closeout's fold into a new named function, `applyAutopilotTerminalCloseout`, called by the RunE closure (verified by the existing `workCloseoutFindCobraRunE` AST lookup) and itself calling `autopilotTerminalWorkOutcome` (a normal, discoverable direct call).
- **Files modified:** `cmd/compatibility_cmds.go`, `cmd/work_closeout_lanes_test.go`
- **Commit:** `1f657f23` (production fix), `62a16b50` (test adjusted to match)

No other deviations -- Tasks 1-3 otherwise executed as written, including the shared-render-fold pattern, the AST call-graph guard reuse, and the dual-typed (in-process struct vs. JSON-round-tripped map) verdict/verification readers.

## Issues Encountered

None blocking beyond the four auto-fixes documented above.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. `cmd/codex_continue_finalize.go`'s check verdict wiring is scoped out (see Decisions Made / key-decisions) rather than stubbed -- the resolvers it would need already exist and fall back safely; no placeholder or empty-value code was introduced for it.

## Next Phase Readiness

D-05 (equal-ceremony closeout), D-06 (elapsed/cost on every closeout, including `aether run`), and D-07 (recommended next action) are now closed for the check lane's default/fast path and for both of the autopilot's closeout sites, per `201-VERIFICATION.md`'s root-cause analysis. This closes out the last of the five gap-closure plans this phase's verification report identified (201-16 through 201-20); Phase 201's remaining follow-up items are the ones already disclosed as intentionally out of scope: `cmd/codex_continue_finalize.go`'s own pre-review blocked path (which never reaches the shared `runContinueAcceptVerifyAdvance` decision body, a distinct pre-existing gap) and its check-verdict wiring, and `runBoundedRepairRound`'s continued non-use in favor of the parallel inline `applyBoundedCheckFixRepair` implementation (documented in 201-17's own summary).

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-11*

## Self-Check: PASSED

All claimed files exist (`cmd/work_closeout.go`, `cmd/codex_continue.go`, `cmd/work_closeout_test.go`, `cmd/codex_workflow_cmds.go`, `cmd/ceremony_cmd.go`, `cmd/compatibility_cmds.go`, `cmd/work_closeout_lanes_test.go`, this SUMMARY.md) and all three task commit hashes (`761da85f`, `1f657f23`, `62a16b50`) are present in `git log --oneline --all`.
