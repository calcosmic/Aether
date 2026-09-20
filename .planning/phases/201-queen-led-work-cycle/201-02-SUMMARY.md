---
phase: 201-queen-led-work-cycle
plan: "02"
subsystem: continue-lifecycle
tags: [go, continue, verification, gates, accept-verify-advance, deterministic-floor]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 01)
    provides: "SYN-201-04's cited synthesis decision — the unified accept/verify/advance function operates on lane-normalized Go types, each continue lane keeps its own untrusted-input parsing responsibility, only the decision core is single-sourced"
provides:
  - "cmd/codex_verify_advance.go: runContinueAcceptVerifyAdvance, the one accept/verify/advance decision body the direct, plan-only/snapshot, and external finalize continue lanes all route through"
  - "An AST guard (TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody) that refuses, by name, any future top-level function that independently evaluates continue gates without calling the shared body"
  - "A real bug fix in the finalize lane's soft_block auto-resolve step (variable shadowing that silently discarded a resolved gate) surfaced and fixed as a direct consequence of wiring the shared decision body correctly"
affects: [201-03, 201-04, 201-05, 201-06, 201-07, 201-08, 201-09, 201-10, 201-11, 201-12, 201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 10996
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single-decision-body collapse (mirrors runDeterministicFloor's already-proven shape one layer up): runContinueAcceptVerifyAdvance takes an already-computed codexContinueGateReport and an optional *codexContinueReviewReport, and returns one typed continueAcceptVerifyAdvanceDecision -- never re-evaluates gates itself, so a lane's own pre-decision side effects (finalize's soft_block auto-resolve) are never silently discarded by a second, divergent gate evaluation"
    - "AST-guard call-graph derivation (mirrors cmd/rendered_fields_invariant_test.go's renderedFieldASTFuncs precedent): the guard's authority is 'which top-level function directly calls the shared gate evaluator without also calling the shared decision body' -- computed from go/parser + go/ast at test time, never a hand-maintained list of lane names"

key-files:
  created:
    - cmd/codex_verify_advance.go
    - cmd/codex_verify_advance_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/codex_continue_plan.go
    - cmd/codex_continue_finalize.go

key-decisions:
  - "runContinueAcceptVerifyAdvance takes an already-evaluated codexContinueGateReport as an input parameter rather than calling runCodexContinueGatesWithAutopilotBaseline internally. The finalize lane's existing soft_block auto-resolve pass mutates its own local gates value BEFORE the accept/verify/advance decision is made; having the shared body recompute gates from scratch would silently discard that resolution. Every lane still funnels through the one shared gate evaluator -- it is each lane's own responsibility to call it (and apply its own lane-specific recovery steps) before reaching the decision body."
  - "The direct lane (runCodexContinue) calls the shared decision body TWICE per invocation: once with review=nil right after gates are computed (to decide whether a reviewer needs to be dispatched at all, preserving the existing cost-avoidance behavior of never paying for a review wave when gates already fail), and once more with the real review report once one exists. A nil review can never itself cause a block."
  - "The plan-only/snapshot lane's call is advisory only: it has no reviewer report yet at plan-only time (dispatch hasn't happened), so it calls the shared body with review=nil purely to fold its own gates evaluation into the same typed decision shape queenDecide consumes -- the real, binding advancement decision is made later by the finalize lane once wrapper-dispatched review results exist."

requirements-completed: []
# WORK-04 and CEC-06 (this plan's declared requirements) are each also
# declared by sibling 201-* plans (WORK-04: 201-03/05/15; CEC-06: 201-04/06/
# 07/12/15) and therefore stay open per the shared-ID gate (#2388) until every
# declaring plan has a SUMMARY.md -- correct, expected behavior, not a gap in
# this plan's own work.

coverage:
  - id: D1
    description: "One accept/verify/advance decision body (runContinueAcceptVerifyAdvance) that the direct, plan-only, and external finalize continue lanes all reach through to determine whether a phase advances"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestRunContinueAcceptVerifyAdvanceAdvancesWithDeterministicFloorAsPassSource"
        status: pass
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestRunContinueAcceptVerifyAdvanceBlocksOnFailingFloorRegardlessOfPassingReviewer"
        status: pass
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestRunContinueAcceptVerifyAdvanceBlocksOnPassingFloorWithBlockingReviewerFinding"
        status: pass
      - kind: integration
        ref: "cmd/codex_verify_advance_test.go#TestDirectContinueLaneReachesAdvancementThroughTheSharedBody"
        status: pass
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestThreeContinueLanesProduceOneDecision"
        status: pass
    human_judgment: false
  - id: D2
    description: "The finalize lane accepts a supplied reconciliation record exactly as the direct lane accepts it -- the CONCERNS.md-named divergence is closed with one acceptance rule"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestRunContinueAcceptVerifyAdvanceAppliesOneReconciliationDisposition"
        status: pass
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane"
        status: pass
    human_judgment: false
  - id: D3
    description: "Evidence written by an interrupted attempt stays bound to the attempt that produced it, and a second attempt against the same phase never adopts or overwrites another attempt's evidence"
    requirement: CEC-06
    verification:
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestConcurrentLanesKeepEvidenceAttemptBound"
        status: pass
    human_judgment: false
  - id: D4
    description: "An AST guard refuses, by name, any future top-level function in the cmd package that evaluates continue gates independently of the shared decision body -- derived from the call graph, never a hardcoded list of lane names"
    requirement: WORK-04
    verification:
      - kind: unit
        ref: "cmd/codex_verify_advance_test.go#TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody"
        status: pass
    human_judgment: false

duration: 110min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 02: Queen-Led Work Cycle Continue Lane Collapse Summary

**Collapsed the direct, plan-only, and external finalize `aether continue` implementations onto one `runContinueAcceptVerifyAdvance` decision body in `cmd/codex_verify_advance.go`, closing the reconciliation-acceptance divergence and fixing a real gate-shadowing bug the refactor surfaced.**

## Performance

- **Duration:** 110 min
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- Created `cmd/codex_verify_advance.go` with `runContinueAcceptVerifyAdvance`, the single typed decision body every continue lane now reaches through to determine advance vs. block, the pass source, and the reconciliation disposition — modeled directly on `runDeterministicFloor`'s already-proven "one function, many thin callers" shape one layer up.
- Rewired the direct lane (`runCodexContinue`, `cmd/codex_continue.go`) to route both its pre-review gate check and its post-review final check through the shared body, proven end to end against a real `aether continue` fixture (`TestDirectContinueLaneReachesAdvancementThroughTheSharedBody`).
- Rewired the plan-only/snapshot lane (`runCodexContinuePlanOnly`, `cmd/codex_continue_plan.go`) to fold its advisory (pre-dispatch) gate evaluation through the same shared body, feeding `queenDecide` from the decision's own `Gates` field rather than a second, independently-held variable.
- Rewired the external finalize lane (`runCodexContinueFinalize`, `cmd/codex_continue_finalize.go`) to reach its final advancement verdict through the shared body, and found — while wiring it — that the finalize lane's soft_block auto-resolve step (`gates, autoResolved := autoResolveSoftBlockGates(...)`) shadowed the outer `gates` variable, silently discarding a resolved gate for every consumer past that if-block. Fixed by assigning with `=` instead of `:=` (Rule 1 auto-fix; see Deviations).
- Added an AST guard (`TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody`) that parses the whole `cmd` package with `go/parser`/`go/ast`, derives which functions call the shared decision body from the actual call graph (never a hardcoded name list), and fails by name (with file:line) when a synthetic fourth continue-decision implementation calls the shared gate evaluator without also calling the shared body.
- Added unit coverage locking the three core invariants: the deterministic floor is the sole source of a pass (a passing reviewer cannot rescue a failing floor), a reviewer verdict can only add a block on top of a passing floor, and a supplied reconciliation record is disposed identically regardless of which lane supplied it — plus a parity test (`TestThreeContinueLanesProduceOneDecision`) and a reconciliation-specific parity test (`TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane`) proving the direct and finalize lanes' own gate-construction paths produce byte-identical decisions for identical inputs.
- Added an attempt-bound-evidence test (`TestConcurrentLanesKeepEvidenceAttemptBound`, CEC-06) proving two attempts tracked against the same phase never read or write each other's durable evidence file.

## Task Commits

1. **Task 1: End-to-end "one continue decision, three lanes" — the direct path only** - `c99712f2` (feat) — also carries Task 3's AST guard test, since both live in the single new `cmd/codex_verify_advance_test.go` file authored in one pass.
2. **Task 2: Route the snapshot and external finalize lanes through the same body** - `5f1e12e5` (fix) — includes the soft_block gate-shadowing bug fix this task's wiring surfaced.

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/codex_verify_advance.go` - The one accept/verify/advance decision body (`runContinueAcceptVerifyAdvance`) and its typed decision result
- `cmd/codex_verify_advance_test.go` - Unit coverage of the decision body's own rules, an end-to-end direct-lane proof, a three-lane parity proof, a reconciliation-parity proof, an attempt-bound-evidence proof, and the AST single-body guard
- `cmd/codex_continue.go` - Direct lane (`runCodexContinue`) now reaches its advancement verdict only through the shared body (two call sites: pre-review and post-review)
- `cmd/codex_continue_plan.go` - Plan-only lane (`runCodexContinuePlanOnly`) now folds its advisory gate evaluation through the shared body
- `cmd/codex_continue_finalize.go` - Finalize lane (`runCodexContinueFinalize`) now reaches its final advancement verdict through the shared body; fixed the soft_block auto-resolve variable-shadowing bug this wiring surfaced

## Decisions Made

- `runContinueAcceptVerifyAdvance` takes an already-evaluated `codexContinueGateReport` as an input, rather than calling `runCodexContinueGatesWithAutopilotBaseline` internally. The plan's own pattern guidance suggested the decision body could own gate evaluation itself; in practice the finalize lane's soft_block auto-resolve pass needs to mutate its own gates value BEFORE the decision is made, and having the shared body recompute gates from scratch would have silently thrown that resolution away. Every lane still funnels through the one shared gate evaluator (`runCodexContinueGates`/`runCodexContinueGatesWithAutopilotBaseline`, unchanged, already shared before this plan) — it is each lane's own responsibility to call it (and apply any lane-specific recovery steps) before reaching the decision body. The AST guard enforces this indirectly: any OTHER function calling the gate evaluator without also calling the shared decision body is flagged.
- The direct lane calls the shared body twice (pre-review with `review=nil`, post-review with the real report) rather than once, to preserve the existing behavior of never dispatching a reviewer when gates already fail. A nil review can never cause a block on its own.
- Chose not to force `runCodexContinueVerificationSnapshot` (mentioned in the plan's Task 2 acceptance criteria as needing "at least one call" to the shared body) to call `runContinueAcceptVerifyAdvance`: that function produces the verification report BEFORE an assessment exists, so it structurally cannot supply the shared body's required `codexContinueAssessment` input. The AST guard's actual predicate (does a function consume an assessment AND independently evaluate gates) naturally exempts it — it doesn't decide advancement at all, only computes one of the decision's upstream inputs.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a variable-shadowing bug in the finalize lane's soft_block auto-resolve step**
- **Found during:** Task 2, wiring `runCodexContinueFinalize`'s final decision call
- **Issue:** `gates, autoResolved := autoResolveSoftBlockGates(phase.ID, gates, resolveDepth, phase.Mode)` used `:=` inside a nested `if !gates.Passed { ... }` block, which — because `gates` was declared at function scope, not this block's scope — created a NEW, shadowed local `gates` visible only inside that block. Everything past the block's closing brace (review dispatch, `advanceExternalContinue`, the persisted continue report) kept reading the STALE, pre-resolution outer `gates.Passed=false`, even after a soft_block gate had genuinely been auto-resolved. `advanceExternalContinue` never re-checks `.Passed` itself, so this bug was invisible as a functional block before this plan — but `runContinueAcceptVerifyAdvance` DOES gate on `gates.Passed`, so wiring the shared decision body correctly required fixing this first (a resolved gate must actually reach the decision, or every auto-resolved phase would now wrongly block).
- **Fix:** Declared `var autoResolved []string` and reassigned the OUTER `gates` with `=` instead of `:=`.
- **Files modified:** `cmd/codex_continue_finalize.go`
- **Verification:** `go build ./cmd/...` and `go vet ./cmd/...` clean; `TestThreeContinueLanesProduceOneDecision` and `TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane` exercise the finalize lane's gate-construction path and pass; no existing test (`TestContinueFinalizeAutoResolve_*` in `cmd/gate_test.go`) exercises the full finalize-lane call path that this bug lived on — those tests call `autoResolveSoftBlockGates` directly, so this fix carries no risk of changing an asserted-on value in the existing suite (confirmed by running the full `go test ./cmd -run 'Continue'` sweep before and after: unchanged, all passing).
- **Committed in:** `5f1e12e5` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug). **Impact on plan:** Necessary for the shared decision body to correctly reflect the finalize lane's own gate resolution; without it, the collapse this plan performs would have introduced a real regression (a previously-advancing auto-resolved phase would start wrongly blocking). No scope creep — no other unrelated behavior was touched.

## Issues Encountered

- The full, unfiltered `go test ./cmd` package suite (555s) hit a pre-existing, environment-level resource-exhaustion failure unrelated to this plan: `cmd/isolated_process_test.go`'s parallel "lane" fixtures (an entirely separate test area that spawns many isolated child OS processes) hit `isolated process child hub setup deadline: context deadline exceeded` under the full package's combined parallel load, causing dozens of unrelated lanes to report "missing executed tests" (tests that never got scheduled, not tests that failed). Confirmed this is pre-existing and unrelated: (1) `go test ./cmd -run 'TestIsolatedProcess'` passes cleanly in isolation (1.2s); (2) no `--- FAIL` line for any named test appears anywhere in the 555s run's output; (3) this matches a previously-documented, known environment-capacity limitation of this project's own test suite (kernel process-spawn cap under heavy parallel load), not something this plan's changes could cause or should fix. All of this plan's own tests were verified directly via targeted `go test ./cmd -run '<name>'` invocations, all passing, plus the broader `go test ./cmd -run 'Continue'` sweep (hundreds of continue-related tests, 123s, clean) run both before and after the finalize-lane bug fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `runContinueAcceptVerifyAdvance` (`cmd/codex_verify_advance.go`) is now the one place every later Phase 201 plan touching continue-lane advancement logic should add its rule — the AST guard (`TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody`) will fail by name if a future plan instead grows a fourth independent implementation.
- `go build ./cmd/...`, `go vet ./cmd/...`, and `go vet ./...` are clean. Every task-level `<verify>` command from `201-02-PLAN.md` passes. The broader `go test ./cmd -run 'Continue'` sweep (hundreds of tests) passes cleanly both before and after this plan's changes.
- WORK-04 and CEC-06 remain open in `REQUIREMENTS.md` (correctly — both are shared across several 201-* plans still in progress) and will close once every plan declaring them has summarized.
- Ready for `201-03-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/codex_verify_advance.go` — FOUND
- `cmd/codex_verify_advance_test.go` — FOUND
- `.planning/phases/201-queen-led-work-cycle/201-02-SUMMARY.md` — FOUND
- Commit `c99712f2` (Task 1 + Task 3) — FOUND in git log
- Commit `5f1e12e5` (Task 2) — FOUND in git log
- `go build ./cmd/...` — clean
- `go vet ./cmd/...` and `go vet ./...` — clean
- All plan `<verify>` commands re-run and passing: `TestDeterministicFloorIsTheOnlySourceOfAPass`, `TestBothContinueLanesApplyTheSameFloor`, `TestDirectContinueLaneReachesAdvancementThroughTheSharedBody`, `TestThreeContinueLanesProduceOneDecision`, `TestFinalizeLaneAcceptsReconciliationLikeTheDirectLane`, `TestConcurrentLanesKeepEvidenceAttemptBound`, `TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody`
