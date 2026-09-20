---
phase: 201-queen-led-work-cycle
verified: 2026-09-11T10:30:00Z
status: passed
score: 101/101 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 92/101
  gaps_closed:
    - "D-01 (201-03/201-16): The Queen chooses, per phase, whether reviewer judgment lands at build-end or the check step, recorded with its reason on the exact attempt"
    - "D-03 (201-05/201-19): The check-step build closeout says plainly the work is built, checks passed, verification has not run yet, and names the next command — no success-verdict token before verification"
    - "D-05 (201-04/201-19/201-20): Every one of the six work verdicts renders the full closeout ceremony; build, check, and autopilot closeouts that carry a verdict show it"
    - "D-08 (201-07/201-18/201-19): The result card names credited files and uncredited files with where they live, on both build lanes"
    - "D-07 (201-07/201-19): After a non-success outcome the Queen recommends exactly one next action with a reason and alternatives"
    - "CAP-066 (201-07/201-18): Content-level decision and learning deltas are bound to the exact attempt and shown at the consuming decision"
    - "D-11 (201-09/201-17): The failed-repair handback names what is failing, what was tried and why it did not take, the restored safe position, and one action for the owner"
    - "D-06/WORK-05 (201-06/201-20): Every build, check and autopilot closeout ends with elapsed time and reported cost for that exact attempt — including aether run's own terminal closeout"
  gaps_remaining: []
  regressions: []
gaps: []
deferred: []
human_verification: []
---

# Phase 201: Queen-Led Work Cycle Verification Report

**Phase Goal:** Restore visible Queen judgment, proportionate teams, coherent waves, one verification boundary, recovery, and autopilot as a single work story.
**Verified:** 2026-09-11T10:30:00Z
**Status:** passed
**Re-verification:** Yes — after gap-closure plans 201-16 through 201-20 (initial verification 2026-09-10T21:15:00Z found 8 gaps / 9 failed truths, all rooted in orphaned rendering/write functions with zero production callers)

## Summary

The initial verification found that the phase's decision/data layer was correct and unit-tested but systematically unwired: `queenApplyVerificationBoundary`/`attachVerificationBoundary`, `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails`, `LifecycleCloseoutDetails.WorkOutcome`, `renderResultFilePrecisionCard`, `attachBuildKnowledgeDeltas`, and `buildFailedRepairHandback` all had zero production callers, so a real user of `aether build`/`aether continue`/`aether run` saw pre-Phase-201 closeouts everywhere. Five gap-closure plans (201-16..201-20) were executed to wire them. This re-verification confirmed **against the current code, not the SUMMARYs** — via grep of `cmd/*.go` excluding `_test.go`, reading each call-site chain end-to-end, and running the new end-to-end and AST call-graph guard tests — that every one of the 8 gaps is genuinely closed. Human UAT (`201-UAT.md`) has since passed 74/74 checkpoints. `go build ./cmd/aether` and `go vet ./cmd/...` are clean; no debt markers (`TBD`/`FIXME`/`XXX`) in any file the closure plans touched.

## Gap Closure Verification (the 8 previous gaps, each traced in current non-test code)

| # | Old gap | Status | Production evidence (grep-traced, current tree) |
|---|---------|--------|--------------------------------------------------|
| 1 | D-01: boundary write side orphaned | ✓ CLOSED | `queenApplyVerificationBoundary` called at `cmd/codex_build.go:572` (direct-dispatch lane) and `:712` (plan-only lane); `attachVerificationBoundary` persists it at `cmd/codex_build.go:883` and `:1168`, immediately after `commitBuildStart` on both lanes. `aether build --verification-boundary/--boundary-why` is the real input channel. `TestQueenBoundaryChoiceReachesTheRecordedAttempt` (three real plan-only builds: default, build-end-with-reason, refused-no-reason) PASS; `TestBoundaryWriteSideHasProductionCallers` (AST guard) PASS. |
| 2 | D-03: honest "not verified yet" closeout orphaned | ✓ CLOSED | `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails` now called from `cmd/work_closeout.go:317`/`:315` inside `buildWorkCloseoutDetails`, which is reached from three production sites: `cmd/codex_workflow_cmds.go:471` (direct `aether build` RunE), `cmd/ceremony_cmd.go:278` (wrapper/chat lane), `cmd/compatibility_cmds.go:1124` (autopilot per-phase build closeout — same resolver, so the two cards can never disagree). Success verdict is only reachable via a build-end boundary with actually-passed build-end reviewers; check-step default renders partial/"not verified yet". `TestBuildBeforeVerification*`, `TestBuildCloseoutCarriesTheRealVerdict`, `TestBuildCloseoutVerdictHasProductionCallers` PASS. |
| 3 | D-05: `LifecycleCloseoutDetails.WorkOutcome` never set in production | ✓ CLOSED | `WorkOutcome:` is now set at 10+ production sites: all verdict constructors in `cmd/work_closeout.go` (:326, :350, :361, :375, :384, :394, :403), `checkWorkCloseoutDetails` (:200), the build closeout builders (`cmd/codex_build.go:4818`, `:4846`), the repair handback (`cmd/work_repair.go:685`), and `applyAutopilotTerminalCloseout` (`cmd/compatibility_cmds.go:884-886`). The check lane derives its verdict once per decision (`checkWorkOutcome`, called at `cmd/codex_continue.go:853` and `:1019`, stored at all three result-save points) and reads it back at render time on both check surfaces (`cmd/codex_workflow_cmds.go:580/:586` via `applyCheckWorkCloseout`; `cmd/ceremony_cmd.go:327` via `checkWorkCloseoutDetails`). `TestEveryWorkVerdict*`, `TestCloseoutCeremony*`, `TestCheckWorkOutcome`, `TestEveryWorkLaneCloseoutHasProductionCallers` PASS. |
| 4 | D-08: file-precision card never rendered; native lane never recorded the data | ✓ CLOSED (both halves) | Data: `attachResultFilePrecision` now called on the native/in-repo lane (`cmd/codex_build.go:1363`) as well as the external/wrapper finalize lane (`cmd/codex_build_finalize.go:904`). Card: `renderResultFilePrecisionCard` reached via `renderBuildResultFileSection` (`cmd/work_closeout.go:496`) from `cmd/codex_workflow_cmds.go:475` and `cmd/ceremony_cmd.go:281`. `TestBothBuildLanesAttachResultEvidenceToTheExactAttempt` (real native build + real external finalize), `TestAttemptEvidenceNeverCrossesAttempts`, `TestBuildResultEvidenceHasProductionCallers` PASS. |
| 5 | D-07: recommended next action never renders | ✓ CLOSED | `recommendedActionForWorkOutcome` (`cmd/lifecycle_closeout.go:255`) is gated on `details.WorkOutcome != nil`, and that gate is now satisfied in production by every site in row 3 — the recommendation renders on real build, check, and autopilot closeouts. Confirmed inside `TestCheckAndAutopilotCloseoutsCarryAVerdict` ("carries a verdict, a recommendation, and one cost block") PASS. |
| 6 | CAP-066: knowledge deltas never attached or shown | ✓ CLOSED | `attachBuildKnowledgeDeltas` called from both lanes (`cmd/codex_build.go:1366`, `cmd/codex_build_finalize.go:918`), fed by the new pure derivation `deriveBuildKnowledgeDeltas` (`cmd/build_knowledge_deltas.go`) from the attempt's own workers' handoffs. `lifecycleCloseoutKnowledgeDeltaEvidence` renders it on the WorkOutcome-carrying closeout (`cmd/lifecycle_closeout.go:268`), with an ID-based dedup guard (`excludeLifecycleEvidenceByID`) preventing the double-render the first live verdict would otherwise have caused. `TestDeriveBuildKnowledgeDeltas` and the "knowledge deltas reach the resolved details as evidence" subtest PASS. |
| 7 | D-11: failed-repair handback never called | ✓ CLOSED | `buildFailedRepairHandback` called at `cmd/work_repair.go:519` inside `applyBoundedCheckFixRepair`'s still-failing branch (assembled from the re-run floor's own first failing step and the real restore outcome — a failed restore no longer claims a restore happened). `applyBoundedCheckFixRepair` is wired at `cmd/codex_continue.go:2130`; the handback travels via `codexContinueVerificationReport.RepairHandback` (`cmd/codex_continue.go:2177`) and renders on the blocked screen via `renderFailedRepairHandback` (`cmd/codex_visuals.go:2423`, dual-typed reader for both lanes). `TestFailedRepairHandsBackAllFourParts` (genuinely failing check drives all four parts end-to-end; two negative cases render nothing) and `TestFailedRepairHandbackHasProductionCallers` PASS. |
| 8 | D-06: `aether run` closeout lacks elapsed+cost | ✓ CLOSED | `applyAutopilotTerminalCloseout` (`cmd/compatibility_cmds.go:876`) derives the run's terminal verdict from its own recorded trigger code (`autopilotTerminalWorkOutcome`, total over every stop code — `TestAutopilotTerminalVerdictCoversEveryStopCode` PASS), sets `details.WorkOutcome`, and is called from `runCompatibilityCmd`'s terminal path (`cmd/compatibility_cmds.go:330`), satisfying `appendLifecycleCloseoutSpendLine`'s gate. The "a real autopilot invocation carries a verdict and one cost block" and "a command carrying no work verdict shows no cost block" subtests of `TestCheckAndAutopilotCloseoutsCarryAVerdict` both PASS — cost appears where a verdict exists, never as an empty heading elsewhere. |

## Regression Check (sample of originally-verified must-haves)

All 10 sampled wirings from the initial verification remain intact in current non-test code:

| Must-have | Status | Evidence |
|-----------|--------|----------|
| Shared accept/verify/advance body reached by all 3 continue lanes | ✓ HOLDS | `runContinueAcceptVerifyAdvance` referenced from `codex_continue.go`, `codex_continue_plan.go`, `codex_continue_finalize.go` |
| Boundary read path prevents double-dispatch | ✓ HOLDS | `verificationBoundaryForAttempt` in `codex_build.go`, `codex_continue.go` (and now also `work_closeout.go`) |
| Job/attempt identity threading | ✓ HOLDS | `attemptCoherentJobWaves` in `codex_verify_advance.go`, `memory_feed.go` |
| Bounded check-repair with checkpoint/restore + D-12 signal | ✓ HOLDS | `applyBoundedCheckFixRepair` at `codex_continue.go:2130`; `deliverFailureBornRepairSignal` called at `work_repair.go:489` |
| Status running colony total | ✓ HOLDS | `computeColonyRunningSpendTotal` in `cmd/status.go` |
| Eight-segment telemetry on both lanes | ✓ HOLDS | `writeJobTelemetryRecord` in `codex_build.go`, `codex_continue.go` |
| Caste model routing lever | ✓ HOLDS | `resolveCasteModelRoute` in `codex_build.go` |
| Cycle-point deterministic floor (loop + boundary) | ✓ HOLDS | `runDeterministicFloorAtCyclePoint` in `codex_build_finalize.go`, `deterministic_floor.go`; `deriveVerificationScopeAtCyclePoint` in `deterministic_floor.go` |
| Proportionate `/ant-quick` path | ✓ HOLDS | `runQuickScout` in `command_truth.go`, `codex_build.go`, `memory_feed_continue.go` |
| Loop-vs-boundary verification scoping | ✓ HOLDS | `verification_scope.go` wired as above |

No regressions found.

## Behavioral Spot-Checks

- `go build ./cmd/aether` — clean.
- `go vet ./cmd/...` — clean.
- Targeted named tests, all PASS, `-count=1` (never the full ~20-minute suite, per owner policy):
  - `TestBuildBeforeVerification* | TestEveryWorkVerdict* | TestCloseoutCeremony*` plus all five closure plans' AST call-graph guards (`TestBoundaryWriteSideHasProductionCallers`, `TestFailedRepairHandbackHasProductionCallers`, `TestBuildResultEvidenceHasProductionCallers`, `TestBuildCloseoutVerdictHasProductionCallers`, `TestEveryWorkLaneCloseoutHasProductionCallers`) — 4.4s, ok.
  - End-to-end lane tests: `TestQueenBoundaryChoiceReachesTheRecordedAttempt`, `TestBothBuildLanesAttachResultEvidenceToTheExactAttempt`, `TestAttemptEvidenceNeverCrossesAttempts`, `TestBuildCloseoutCarriesTheRealVerdict`, `TestCheckAndAutopilotCloseoutsCarryAVerdict`, `TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus` — all PASS.
  - `TestFailedRepairHandsBackAllFourParts` — 20.7s, ok.
- Anti-pattern scan: zero `TBD`/`FIXME`/`XXX` across all 13 production files touched by 201-16..201-20.
- Human UAT: `201-UAT.md` passed 74/74 checkpoints (owner-run, post-closure).

## Informational Caveats (not gaps)

1. **Heavy/classic-ceremony check lane carries no verdict.** `cmd/codex_continue_finalize.go` (the opt-in `--classic-ceremony` lane) was deliberately not wired to store the check verdict — its blocked-before-review path never reaches the shared decision body, a pre-existing architectural gap distinct from and older than this phase's scope, disclosed in 201-20's key-decisions. It falls back byte-identically to today's rendering (nothing regresses). The documented primary daily-driver path (default `aether continue`) and autopilot's check calls both carry the verdict.
2. **`selectAutopilotGoalTransition` remains additive/reporting, not decision-driving** (carried forward from the initial verification's caveat on 201-11). The observable outcome — autopilot progresses automatically and stops at declared boundaries, now with a terminal verdict and cost block — is achieved.

## Gaps Summary

None. All 8 previously-reported gaps are closed with production call sites confirmed in current non-test code, exercised by passing end-to-end tests, and locked against silent disconnection by AST call-graph guard tests. The phase goal — visible Queen judgment (boundary choice with reason on the attempt), proportionate teams, coherent waves, one verification boundary (read AND write sides now live), recovery (four-part failed-repair handback on the blocked screen), and autopilot as a single work story (same verdict resolver as the build lane, terminal cost/time block) — is achieved in the codebase.

---

_Verified: 2026-09-11T10:30:00Z_
_Verifier: Claude (gsd-verifier, re-verification after 201-16..201-20)_
