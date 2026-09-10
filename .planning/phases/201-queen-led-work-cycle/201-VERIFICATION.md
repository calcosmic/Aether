---
phase: 201-queen-led-work-cycle
verified: 2026-09-10T21:15:00Z
status: gaps_found
score: 92/101 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "The Queen chooses, per phase, whether reviewer judgment lands at build-end or the check step, and that choice is recorded with its reason as one durable fact bound to the exact attempt (D-01, 201-03)."
    status: failed
    reason: "queenApplyVerificationBoundary and attachVerificationBoundary (the write side of the boundary decision) are declared and unit-tested but have zero callers anywhere in cmd/*.go outside of tests. Confirmed by grep across the whole non-test tree. Nothing in production ever proposes or persists a build-end decision, so verificationBoundaryForAttempt always returns ok=false and the boundary is always the check-step default -- on every phase, always, with no exception. The Queen never actually gets to choose."
    artifacts:
      - path: "cmd/verification_boundary.go"
        issue: "queenApplyVerificationBoundary/attachVerificationBoundary exist and pass their own unit tests but are not called from cmd/queen_judgement.go, cmd/codex_build.go, or any other production dispatch path."
    missing:
      - "A real call site (most likely inside the Queen's caste-judgement dispatch, before a build attempt is sealed) that proposes a boundary choice from actual phase risk/content and calls queenApplyVerificationBoundary + attachVerificationBoundary to persist it."
  - truth: "When the boundary is the check step, the build closeout says plainly that the work is built, the deterministic checks passed, verification has not run yet, and names the one command to run next (D-03, 201-05); the closeout before verification renders no success-verdict token."
    status: failed
    reason: "buildUnverifiedCloseoutDetails and buildVerifiedCloseoutDetails (cmd/codex_build.go) are fully built and unit-tested (TestBuildBeforeVerificationSaysItIsNotVerifiedYet passes) but have zero production callers -- confirmed by grep across the whole non-test tree. The one existing production build-closeout call site (cmd/compatibility_cmds.go:1025, inside aether run's autopilot flow) constructs a generic LifecycleCloseoutDetails{Summary: \"...reached its declared build boundary.\"} instead. A real user running aether build, aether continue, or aether run today never sees the honest 'not verified yet' card this phase built -- they see the same closeout messaging as before Phase 201."
    artifacts:
      - path: "cmd/codex_build.go"
        issue: "buildUnverifiedCloseoutDetails/buildVerifiedCloseoutDetails (lines ~4716-4763) declared and tested, never called from any non-test file."
    missing:
      - "Wire the recorded verification-boundary decision and a real deterministic-check command list into the actual build-closeout call site(s) for the native/in-repo lane, the external/wrapper finalize lane, and the aether run autopilot lane."
  - truth: "Every one of the six work verdicts renders the same full closeout ceremony slots for a real closeout (D-05, 201-04); every build/check/autopilot closeout that carries a verdict shows it."
    status: failed
    reason: "colony.WorkOutcome and the equal-ceremony invariant are correct and unit-proven (TestEveryWorkVerdictRendersTheSameCeremonySlots, TestCloseoutCeremonyIsEqualAcrossVerdicts pass), but LifecycleCloseoutDetails.WorkOutcome is never set by any production call site in the whole tree -- confirmed by grep for '.WorkOutcome = ' / 'WorkOutcome:' across cmd/*.go outside tests: the only two non-declaration assignments live inside the still-orphaned buildUnverifiedCloseoutDetails/buildVerifiedCloseoutDetails (see prior gap) and buildFailedRepairHandback (also orphaned, see repair gap below). No real aether build, aether continue, or aether run invocation today ever produces a verdict-carrying closeout. This is the shared root cause behind this gap and the next three."
    artifacts:
      - path: "cmd/lifecycle_closeout.go"
        issue: "buildLifecycleCloseout/applyLifecycleCloseout correctly render a WorkOutcome-carrying closeout when given one, but every production caller of applyLifecycleCloseout (colonize, plan, run, build, entomb, init, seal, pause, resume) passes a LifecycleCloseoutDetails literal with no WorkOutcome field set."
    missing:
      - "At least one real production call site per lane (build attempt sealing, continue's accept/verify/advance result, and aether run's autopilot closeout) that determines the real colony.WorkOutcome for the result and passes it through LifecycleCloseoutDetails.WorkOutcome."
  - truth: "A result card names both the files credited to completed tasks and the files that were edited but credited to nothing, with where those uncredited files live (D-08, 201-07)."
    status: failed
    reason: "renderResultFilePrecisionCard (cmd/build_attempt.go) is pure, tested, and idempotent, but has zero production callers -- confirmed by grep; it is referenced only from cmd/result_file_precision_test.go. The underlying CreditedFiles/UncreditedFiles data IS durably attached for the external/wrapper build-finalize lane (attachResultFilePrecision is called from cmd/codex_build_finalize.go), but the card that would show it to a user is never rendered anywhere, on any lane. The native/in-repo build lane does not even record the data (documented gap in 201-07-SUMMARY.md)."
    artifacts:
      - path: "cmd/build_attempt.go"
        issue: "renderResultFilePrecisionCard declared at line ~861, called only from cmd/result_file_precision_test.go."
    missing:
      - "Call renderResultFilePrecisionCard from a real closeout render path once a BuildAttemptRecord's CreditedFiles/UncreditedFiles are populated, on at least the lane where the data is already recorded (external/wrapper build-finalize), and extend attachResultFilePrecision to the native/in-repo lane's own attempt-sealing point."
  - truth: "After a non-success outcome the Queen recommends exactly one next action matched to that outcome, gives a one-sentence reason, and lists the alternatives beneath it (D-07, 201-07)."
    status: failed
    reason: "recommendedActionForWorkOutcome is correct and reachable only through buildLifecycleCloseout, which -- per the shared root cause above -- never receives a real WorkOutcome in production. The recommendation therefore never renders for a real user today."
    artifacts:
      - path: "cmd/lifecycle_closeout.go"
        issue: "recommendedActionForWorkOutcome is gated behind details.WorkOutcome != nil inside buildLifecycleCloseout; no production caller ever sets that field."
    missing:
      - "Same fix as the D-05 gap: wire a real colony.WorkOutcome into a live closeout call site."
  - truth: "Content-level decision and learning deltas are bound to the exact attempt and shown at the decision that consumes them (CAP-066, 201-07)."
    status: failed
    reason: "attachBuildKnowledgeDeltas (cmd/build_attempt.go) is declared and unit-tested but has zero production callers anywhere -- confirmed by grep. No real source of a 'content-level decision or learning delta' (e.g. the existing memory-capture/consolidation pipeline) has been wired to call it, as 201-07-SUMMARY.md itself discloses. lifecycleCloseoutKnowledgeDeltaEvidence is also gated behind the same unwired WorkOutcome path."
    artifacts:
      - path: "cmd/build_attempt.go"
        issue: "attachBuildKnowledgeDeltas declared, never called outside its own test file."
    missing:
      - "Identify a real production writer (most likely the existing memory-capture/consolidation pipeline) and call attachBuildKnowledgeDeltas from it at attempt-finalize time."
  - truth: "The failed-repair handback names what is failing in plain language, what the repair attempted and why it did not take, the restored safe position, and one concrete question or action for the owner (D-11, 201-09)."
    status: failed
    reason: "buildFailedRepairHandback (cmd/work_repair.go) is declared and would assemble a correct four-part handback, but has zero production callers -- confirmed by grep. It is downstream of runBoundedRepairRound, which itself has no production caller (see gap below). The production check-repair path (applyBoundedCheckFixRepair, which IS wired into cmd/codex_continue.go) does restore the checkpoint and announce the restoration on a failed repair, but never assembles or renders the richer four-part diagnostic handback this truth describes."
    artifacts:
      - path: "cmd/work_repair.go"
        issue: "buildFailedRepairHandback declared at line ~615, never called from a non-test file."
    missing:
      - "Call buildFailedRepairHandback (or an equivalent built from applyBoundedCheckFixRepair's own restore-on-failure outcome) at the point where a failed check-fix repair round returns 'still_failing', and render it as the check's closeout."
  - truth: "Every build, check and autopilot closeout ends with the elapsed time and the reported cost for that exact attempt (D-06/WORK-05, 201-06)."
    status: failed
    reason: "appendLifecycleCloseoutSpendLine is correctly gated on closeout.WorkOutcome != nil (by design, to avoid an empty cost heading on non-work closeouts) -- but since aether run's own closeout (cmd/compatibility_cmds.go, applyLifecycleCloseout(result, \"run\", ...)) never sets WorkOutcome, the new cost-and-time block never appends to a real autopilot closeout screen. The direct aether build / aether continue lane is unaffected -- it already carries cost+elapsed via the pre-existing, independent cmd/ceremony_cmd.go path (renderSpendCostLine, live since Phase 196 plus this phase's elapsed addition) -- so this gap is specific to the aether run (autopilot) closeout the truth explicitly names."
    artifacts:
      - path: "cmd/lifecycle_closeout.go"
        issue: "appendLifecycleCloseoutSpendLine's WorkOutcome gate is correct in isolation, but no autopilot ('run') closeout ever satisfies it."
    missing:
      - "Same root cause as the D-05 gap: give aether run's own closeout a real WorkOutcome, or add an explicit non-WorkOutcome-gated cost/elapsed line specifically for the autopilot terminal report."
deferred: []
human_verification: []
---

# Phase 201: Queen-Led Work Cycle Verification Report

**Phase Goal:** Restore visible Queen judgment, proportionate teams, coherent waves, one verification boundary, recovery, and autopilot as a single work story.
**Verified:** 2026-09-10T21:15:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Summary

All 15 plans landed with clean `go build ./...` and `go vet ./...`, no debt markers (`TBD`/`FIXME`/`XXX`) in any of the 50 files this phase touched, and every explicitly named test in every plan's `must_haves.artifacts[].contains` field passes when run individually (never the full ~20-minute suite, per the owner's stated verification policy). The individual mechanisms this phase built — the shared accept/verify/advance decision body, the six-verdict work-outcome vocabulary, the verification-boundary reconciliation, the bounded checkpointed repair primitives, the eight-segment telemetry record, the turnaround target, and the coherent-job identity threading — are all correct, well-tested, and internally consistent. `go test ./cmd -run 'TestClassicContract'` (101.8s, the full Classic-contract corpus, both platforms) also passes clean, confirming no regression to the pre-existing corpus this phase extended.

However, goal-backward tracing from "does a real user of `aether build`/`aether continue`/`aether run` actually see this" surfaced that a materially larger set of this phase's headline, user-facing deliverables never reach production than the three gaps the phase's own code review already disclosed. The code review (`201-REVIEW.md`) found three orphaned functions (`runBoundedRepairRound`, `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails`, and `queenApplyVerificationBoundary`/`attachVerificationBoundary`). Tracing the closeout-rendering chain further (`LifecycleCloseoutDetails.WorkOutcome`, which every one of those orphaned functions and several others feed into) shows the gap is systemic: **no production call site anywhere in the tree ever sets `LifecycleCloseoutDetails.WorkOutcome`**, which means the entire "six-verdict equal-ceremony closeout" (D-05, this phase's central deliverable), the recommended-next-action card (D-07), the credited/orphaned-files card (D-08's rendering, not just its data), the knowledge-delta evidence (CAP-066), the failed-repair handback (D-11), and the new cost/time block for the `aether run` autopilot lane specifically (D-06) are all fully built, individually unit-tested, and **completely inert for a real user today**. A person running `aether build`, `aether continue`, or `aether run` right now sees the same closeout presentation they saw before this phase, on every one of those points.

This is not a case of missing tests or incorrect logic — every named artifact exists, is substantive, and passes its own test. It is a wiring gap: the decision/data layer this phase built is real and correct, but the "render it to the person running the command" step was consistently deferred, plan after plan, without ever landing in a later plan in this same phase. Given this project's own explicitly stated Definition of Done ("A requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet... not a checked box in a summary") and its documented history of exactly this failure pattern recurring across multiple prior milestones, these are reported as gaps rather than accepted deviations.

**What does genuinely work in production**, confirmed by direct grep-traced call graphs (not SUMMARY claims):
- The single accept/verify/advance decision body (`runContinueAcceptVerifyAdvance`) is reached by all three continue lanes — confirmed via the passing AST/call-graph guard test.
- The verification-boundary **read** path (`verificationBoundaryForAttempt`) is wired into both `cmd/codex_build.go` and `cmd/codex_continue.go`, so double-dispatch is genuinely prevented — verification really does happen exactly once, always at the check-step default.
- Job/attempt identity threading (`attemptCoherentJobWaves`/`attemptCoherentJobDependencies`) is wired into the shared decision body.
- Check-lane repair's checkpoint/save/restore/announce discipline works in production via `applyBoundedCheckFixRepair` (`cmd/codex_continue.go`), including the same-run failure-born signal delivery (D-12) — it just doesn't call the "canonical" `runBoundedRepairRound` function; it reimplements the same discipline inline.
- The status dashboard's running colony total (`computeColonyRunningSpendTotal`) is wired into `cmd/status.go`.
- Eight-segment job telemetry (`writeJobTelemetryRecord`) is wired into both the build and continue production paths.
- The three approved turnaround levers (loop-scoped test narrowing, caste model routing, cycle-point-aware deterministic floor) are all wired into `cmd/codex_build.go` / `cmd/codex_build_finalize.go` / `cmd/deterministic_floor.go`.
- The `/ant-quick` proportionate one-worker path is wired to the real `quick` command entry point.
- The credited/uncredited file **data** (not the card) is durably attached for the external/wrapper build-finalize lane.

## Goal Achievement

### Observable Truths (condensed — 101 declared across 15 plans; every FAILED item is listed individually above and in the table; remaining 92 are VERIFIED and summarized by plan)

| # | Plan | Truth (summarized) | Status | Evidence |
|---|------|---------------------|--------|----------|
| 1 | 201-01 | Every mechanism has an evidence-backed disposition; corpus accepts SYN-201 ids | ✓ VERIFIED | `.planning/.../201-CLASSIC-SYNTHESIS.md` (272 lines, signed), `mechanisms.json`/`schema.json` widened and load-tested |
| 2 | 201-02 | One shared accept/verify/advance body reached by all 3 continue lanes | ✓ VERIFIED | `TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody` PASS (real call-graph AST guard) |
| 3 | 201-03 | Boundary reconciliation function correct (default, refusal, ordering, single-derivation) | ✓ VERIFIED | `TestVerificationBoundaryDefaultsToTheCheckStep`, `TestOneFunctionDerivesTheVerificationBoundary` PASS |
| 3b | 201-03 | **Queen actually chooses build-end vs check-step per phase, in production (D-01)** | ✗ FAILED | See gaps — write side has zero production callers |
| 4 | 201-04 | Six-verdict closed vocabulary, total mapping, backward-compatible | ✓ VERIFIED | `TestWorkOutcomeVocabularyIsClosed`, `TestWorkOutcomeLifecycleMappingIsTotal`, `TestAbsentWorkOutcomeIsNotSuccess`, `TestWorkOutcomeLabelsAreComplete` PASS |
| 4b | 201-04 | **Equal-ceremony closeout actually renders for a real verdict, in production (D-05)** | ✗ FAILED | See gaps — `LifecycleCloseoutDetails.WorkOutcome` never set in production |
| 5 | 201-05 | Reviewer judgment exactly once, read from stored boundary (never doubled) | ✓ VERIFIED | `TestPhaseVerifiedOnce` PASS; `verificationBoundaryForAttempt` genuinely wired into both build and continue dispatch |
| 5b | 201-05 | **Honest "not verified yet" build closeout card renders in production (D-03)** | ✗ FAILED | See gaps — zero production callers |
| 6 | 201-06 | Elapsed/cost reporting discipline correct; status running total wired and sums correctly | ✓ VERIFIED (build/continue, status) | `TestStatusRunningTotalEqualsTheSumOfItsRows`/`...WritesNothing` PASS; `computeColonyRunningSpendTotal` wired into `cmd/status.go`; `cmd/ceremony_cmd.go` direct lane carries cost+elapsed live |
| 6b | 201-06 | **`aether run` (autopilot) closeout also ends with elapsed+cost** | ✗ FAILED | See gaps — gated on the same unwired WorkOutcome |
| 7 | 201-07 | Credited/uncredited split, plan-reality gate, idempotent rendering, correct at the type/function level | ✓ VERIFIED (logic) | `TestResultCardNamesCreditedAndOrphanedFiles` PASS; `attachResultFilePrecision`/`attachBuildPlanRealityReport` wired into `cmd/codex_build_finalize.go` (external lane only) |
| 7b | 201-07 | **Result card actually renders to a user (D-08); next-action recommended (D-07); knowledge deltas shown (CAP-066)** | ✗ FAILED (all three) | See gaps |
| 8 | 201-08 | One job/attempt identity across card, jobs, waves, leases, receipts, findings, fan-in | ✓ VERIFIED | `TestOneJobIdentityReachesEverySurface` PASS; `attemptCoherentJobWaves`/`attemptCoherentJobDependencies` wired into `runContinueAcceptVerifyAdvance` |
| 9 | 201-09 | Checkpointed repair discipline (save/one-wave/verify/restore) actually functions for the check lane | ✓ VERIFIED (via `applyBoundedCheckFixRepair`, not the canonical function) | `TestRepairRunsAtMostOnceAutomatically` PASS; `applyBoundedCheckFixRepair` wired into `cmd/codex_continue.go` with real checkpoint save/restore/announce calls |
| 9b | 201-09 | **Failed-repair four-part handback renders (D-11)** | ✗ FAILED | See gaps — `buildFailedRepairHandback` zero production callers |
| 10 | 201-10 | Failure evidence + blocker truth + same-run repair signal delivery | ✓ VERIFIED | `TestFailureBornSignalReachesTheRepairBrief` PASS; `deliverFailureBornRepairSignal` confirmed wired inside `applyBoundedCheckFixRepair` |
| 11 | 201-11 | Autopilot stop-boundary naming, shared result model, quick path | ✓ VERIFIED | `TestAutopilotSelectsTransitionsFromOneAcceptedGoal` PASS (its own subtest explicitly and honestly documents `runBoundedRepairRound`'s non-wiring as a known gap); `runQuickScout` wired to the real `quick` command |
| 11b | 201-11 | Goal-level transition selector (`selectAutopilotGoalTransition`) actually drives the controller's choice | ⚠ CAVEAT (not counted as failed) | The function's own doc comment states it is additive and "never changes which branch...is taken" — it is computed and reported, not decision-driving. The observable outcome (autopilot does progress automatically) is still achieved through the pre-existing state machine plus this phase's continue-side repair wiring, so the phase-level behavior holds even though this specific artifact doesn't drive it as literally worded. |
| 12 | 201-12 | Eight-segment telemetry, honest-unmeasured discipline, attempt-bound | ✓ VERIFIED | `TestUnmeasuredSegmentIsNeverDerived` PASS; `writeJobTelemetryRecord` wired into both `cmd/codex_build.go` and `cmd/codex_continue.go` |
| 13 | 201-13 | Real one-plan measurement, owner-ratified target, stored for comparison | ✓ VERIFIED | `.planning/.../201-TURNAROUND-BASELINE.md` (200 lines) with a recorded `aether decision-answer` id binding the ratification; owner-approved single-run deviation accepted per stated verification context |
| 14 | 201-14 | Loop-vs-boundary test scoping, brief slimming, model routing, all wired | ✓ VERIFIED | `TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary` PASS; `resolveCasteModelRoute` wired into `cmd/codex_build.go:4464`; `runDeterministicFloorAtCyclePoint` wired into both loop (`codex_build_finalize.go`) and boundary (`deterministic_floor.go`) call sites |
| 15 | 201-15 | Exact-cardinality coverage ledger, causal state-assertion proofs, no rendered-text-only cases | ✓ VERIFIED (as narrowly claimed) | `TestClassicCoverage201ExactSets`, `TestClassicCoverage201ProofResolution`, `TestClassicCoverage201RejectsDecoys` PASS. Note: several cited "proofs" (e.g. `TestBuildBeforeVerificationSaysItIsNotVerifiedYet`) are unit tests of the still-orphaned functions above, not end-to-end command invocations — the coverage ledger proves causal correctness of the isolated function, not production reachability. This does not make 201-15's own narrower claim false, but it should not be read as independent evidence that the wiring gaps above are closed. |

**Score:** 92/101 truths verified (9 FAILED — see gaps in frontmatter; 1 caveat noted, not counted against the score since the phase-level observable behavior it supports is independently achieved).

### Required Artifacts

All 27 named artifacts across the 15 plans exist on disk and are substantive (checked with `wc -l`; none under a stub threshold, none containing `TBD`/`FIXME`/`XXX`/placeholder text). Wiring status:

| Artifact | Provides | Exists | Substantive | Wired to production |
|----------|----------|--------|-------------|----------------------|
| `.planning/.../201-CLASSIC-SYNTHESIS.md` | Signed mechanism study | ✓ | ✓ | N/A (document) |
| `cmd/testdata/classic-contract/v1/{mechanisms,schema,cases}.json` | SYN-201 registry + corpus | ✓ | ✓ | ✓ (loaded by `TestClassicContract*`, passing) |
| `cmd/codex_verify_advance.go` | Shared accept/verify/advance body | ✓ | ✓ | ✓ (all 3 lanes) |
| `cmd/verification_boundary.go` | Boundary decision type + reconciliation | ✓ | ✓ | ⚠ Read side wired, **write side orphaned** |
| `pkg/colony/work_outcome.go` | Six-verdict vocabulary | ✓ | ✓ | ⚠ Type correct, **never fed into a real closeout** |
| `cmd/lifecycle_closeout.go` | Closeout rendering | ✓ | ✓ | ⚠ Renders correctly when given a verdict; **never given one in production** |
| `cmd/spend_cost_line.go` | Cost/elapsed rendering | ✓ | ✓ | ✓ (build/continue direct lane, status); ✗ not on `aether run`'s closeout |
| `cmd/status.go` (running total additions) | Colony-wide running total | ✓ | ✓ | ✓ |
| `cmd/build_attempt.go` (result-precision additions) | Credited/uncredited file data + card | ✓ | ✓ | ⚠ Data wired (external lane only); **card never rendered** |
| `cmd/attempt_artifacts.go` | Attempt-bound claims/verification paths | ✓ | ✓ | ✓ |
| `cmd/coherent_jobs.go` (identity additions) | One job/attempt identity | ✓ | ✓ | ✓ |
| `cmd/work_repair.go` | Bounded checkpointed repair | ✓ | ✓ | ⚠ Canonical function orphaned; **equivalent behavior wired via `applyBoundedCheckFixRepair`** for check lane only; handback rendering orphaned |
| `cmd/memory_feed.go` / `cmd/failure_evidence_test.go` scope | Failure evidence, blocker truth, same-run signal | ✓ | ✓ | ✓ |
| `cmd/autopilot_policy.go` (goal-level additions) | Goal-level transition selector, stop boundary | ✓ | ✓ | ⚠ Computed/reported; not decision-driving (see caveat above) |
| `cmd/command_truth.go` (quick additions) | Proportionate quick path | ✓ | ✓ | ✓ (durability of the attempt record itself is a separate, lesser gap — see code review WR-01, not re-litigated here) |
| `cmd/job_telemetry.go` | Eight-segment timing record | ✓ | ✓ | ✓ |
| `.planning/.../201-TURNAROUND-BASELINE.md`, `cmd/turnaround_target.go` | Measured baseline + ratified target | ✓ | ✓ | N/A (comparison utility, self-contained) |
| `cmd/caste_model_routing.go`, `cmd/verification_scope.go`, `cmd/deterministic_floor.go` (cycle-point additions) | Three approved turnaround levers | ✓ | ✓ | ✓ |
| `cmd/classic_coverage_201_test.go` | Exact-cardinality coverage ledger | ✓ | ✓ | N/A (meta-test) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/codex_continue_finalize.go` | `cmd/codex_verify_advance.go` | shared decision body | ✓ WIRED | Confirmed by passing AST guard |
| `cmd/codex_build.go` / `cmd/codex_continue.go` | `cmd/verification_boundary.go` | `verificationBoundaryForAttempt` (read) | ✓ WIRED | Confirmed by grep, both files call it in non-test code |
| `cmd/queen_judgement.go` (or any Queen dispatch site) | `cmd/verification_boundary.go` | `queenApplyVerificationBoundary` + `attachVerificationBoundary` (write) | ✗ NOT WIRED | Zero production callers — this is the D-01 gap |
| `cmd/codex_build.go` / `cmd/compatibility_cmds.go` | `cmd/codex_build.go`'s closeout builders | `buildUnverifiedCloseoutDetails`/`buildVerifiedCloseoutDetails` | ✗ NOT WIRED | Zero production callers — the D-03 gap |
| Any build/continue/run completion point | `pkg/colony/work_outcome.go` | `LifecycleCloseoutDetails.WorkOutcome` | ✗ NOT WIRED | Confirmed by exhaustive grep for `.WorkOutcome =` / `WorkOutcome:` — the two non-declaration production-file matches are both inside the two already-orphaned functions above and `buildFailedRepairHandback` (also orphaned) |
| `cmd/codex_build_finalize.go` | `cmd/build_attempt.go` | `attachResultFilePrecision`/`attachBuildPlanRealityReport` | ✓ WIRED (external lane only) | Confirmed; native/in-repo lane not wired (disclosed gap, not independently re-flagged) |
| Any closeout render path | `cmd/build_attempt.go` | `renderResultFilePrecisionCard` | ✗ NOT WIRED | Zero production callers |
| `cmd/codex_continue.go` | `cmd/work_repair.go` | `applyBoundedCheckFixRepair` (checkpoint/restore/announce, D-12 signal delivery) | ✓ WIRED | Confirmed — real production repair discipline, though it bypasses the "canonical" `runBoundedRepairRound` |
| Failed check-repair round | `cmd/work_repair.go` | `buildFailedRepairHandback` | ✗ NOT WIRED | Zero production callers — the D-11 gap |
| `cmd/codex_build.go` / `cmd/codex_continue.go` | `cmd/job_telemetry.go` | `writeJobTelemetryRecord` | ✓ WIRED | Confirmed |
| `cmd/codex_build.go` | `cmd/caste_model_routing.go` | `resolveCasteModelRoute` | ✓ WIRED | Confirmed |
| `cmd/codex_build_finalize.go` / `cmd/deterministic_floor.go` | `cmd/verification_scope.go` | `deriveVerificationScopeAtCyclePoint` | ✓ WIRED | Confirmed at both loop and boundary call sites |
| `cmd/autopilot_policy.go` (`buildAutopilotPreflightCore`) | `cmd/codex_verify_advance.go` | `runAutopilotContinue = runCodexContinue` | ✓ WIRED | Confirmed via existing seam |
| `cmd/command_truth.go` | `quick` command entry point | `runQuickScout` | ✓ WIRED | Confirmed |

### Requirements Coverage

| Requirement | Source Plan(s) | Description (plain-English) | Status | Evidence |
|-------------|-----------------|------------------------------|--------|----------|
| SYNTH-03 | 201-01 | Reconstruct Classic work-cycle mechanisms and compare with current Go before choosing changes | ✓ SATISFIED | 272-line signed synthesis document, widened corpus, existing loader test |
| CEC-06 | 201-02, 06, 07, 12 | Claims/checks/findings/time/cost/files/provenance/incomplete-work stay linked to the exact attempt and visible at the decision that consumes them | ⚠ PARTIALLY SATISFIED | Attempt-binding itself is real and correctly implemented throughout (confirmed by `TestOneJobIdentityReachesEverySurface`), but "visible at the decision that consumes them" fails for several evidence types (result-precision card, knowledge deltas) because the rendering layer is orphaned — see gaps |
| WORK-01 | 201-03, 08 | Queen selects the smallest capable team, names workers/jobs, records the reason, subject to safety floors | ⚠ PARTIALLY SATISFIED | Team-selection/job-identity threading (201-08) is fully wired and verified; the specific build-end-vs-check-step boundary judgment (201-03, D-01) never actually exercises in production |
| WORK-02 | 201-05 | A one-task low-risk change uses one Builder plus free checks; named risks add only justified specialists | ✓ SATISFIED | Confirmed by passing tests and pre-existing dispatch policy this phase preserved |
| WORK-03 | 201-08, 15 | Related tasks form dependency-safe jobs with exact leases/receipts/lineage/waves; disjoint work runs in parallel safely | ✓ SATISFIED | `TestOneJobIdentityReachesEverySurface` passing, wired into shared decision body |
| WORK-04 | 201-02, 03, 05, 15 | Build/continue/run consume the same accepted result model, verify once at the authoritative boundary, render one coherent narrative | ⚠ PARTIALLY SATISFIED | "Verify once" is genuinely achieved (no double-dispatch); "render one coherent narrative" fails for the reasons above — the narrative that would actually surface to a user (verdict, honest-unverified card, cost/time on run) is unwired |
| WORK-05 | 201-04, 06, 07, 10 | Success/no-change/partial/blocker/flags/findings/checks/review/time/cost/file precision persist against the attempt and govern advance | ⚠ PARTIALLY SATISFIED | The data model and its persistence are correct and tested; several of the *rendered* facts this requirement names (six-verdict ceremony, cost/time on autopilot closeout, file-precision card) never reach a real screen |
| WORK-06 | 201-09, 10 | Failures write canonical evidence; repair/Swarm waves checkpoint first, stay bounded, verify, restore on failure | ⚠ PARTIALLY SATISFIED | Genuinely functions for the check/continue lane via `applyBoundedCheckFixRepair`; the canonical bounded-repair function and the failed-repair handback are both orphaned; build-lane repair is not evidenced at all |
| WORK-07 | 201-11 | From one accepted goal, the controller selects transitions automatically, stopping only at a declared boundary | ⚠ PARTIALLY SATISFIED | Automatic progression is genuinely observed (via the pre-existing state machine plus this phase's continue-side repair and stop-boundary naming), but the specific new goal-level transition selector this plan built is explicitly non-decision-driving by its own doc comment — see caveat above |
| WORK-08 | 201-12, 13, 14 | Runtime records eight per-job timing segments; briefs stay lean; representative turnaround improves without weakening TDD/safety | ✓ SATISFIED | Telemetry, turnaround target, and all three approved speed levers are confirmed wired into real production call sites |

No orphaned requirements: cross-checking `.planning/REQUIREMENTS.md`'s Phase 201 row (`SYNTH-03, CEC-06, WORK-01..08`) against every plan's declared `requirements:` field confirms all 10 IDs are claimed by at least one plan, and every plan's `requirements-completed` frontmatter is consistent with the shared-ID gating already documented in the SUMMARYs (e.g. WORK-04 stays open until 201-15's own SUMMARY exists, per plan 201-05's explicit note — and 201-15 does exist).

### Anti-Patterns Found

None. Scanned all 50 files named in `201-REVIEW.md`'s `files_reviewed_list` for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented"/"coming soon" — zero matches. The two code-review warnings (WR-01: `quickAttemptRecord` doc-comment overclaims durability it doesn't have; WR-02: a `os.Stat` error stored in a variable named as if it were a boolean) are real but cosmetic/naming issues, not correctness defects, and are already fully documented in `201-REVIEW.md` — not re-litigated here as they don't affect goal achievement.

### Behavioral Spot-Checks

`go build ./...` — clean. `go vet ./...` — clean. `go test ./cmd -run 'TestClassicContract' -v` (both platforms, full corpus including the 199/200 baseline plus this phase's 201-01 additions) — 101.8s, all PASS, no regression. `go test ./pkg/colony -run 'WorkOutcome'` — all PASS. Every individually-named test cited in every plan's `must_haves.artifacts[].contains` field was run by name and passed (see full list in the investigation log above this report was generated from) — none were run as part of a full-suite sweep.

### Gaps Summary

Nine truths, grouped by two root causes:

**Root cause 1 — the closeout rendering layer was built but never wired to a real completion point (affects D-03, D-05, D-06 for the `run` lane, D-07, D-08's rendering, D-11, CAP-066).** Every one of these functions is individually correct and passes its own unit test. The single missing piece, repeated across six of the phase's fifteen plans, is a production call site that supplies a real `colony.WorkOutcome` (or, for the file-precision card, calls the render function directly) at the point a build, check, or autopilot run actually finishes. A real user today sees the pre-Phase-201 closeout on every one of these points.

**Root cause 2 — the Queen's build-end-vs-check-step choice mechanism (D-01) has a correct reconciliation function but no production writer**, so the boundary is always the check-step default; "one verification boundary" is achieved only in the narrow sense of "never double-dispatches," not in the sense of "the Queen actually gets to choose per phase."

A third, smaller finding — the bounded-repair round's *canonical* function (`runBoundedRepairRound`) and its failed-repair handback are unwired, but the underlying checkpoint/save/restore/announce *behavior* genuinely works in production via a parallel inline implementation (`applyBoundedCheckFixRepair`) for the check lane. This is architecturally duplicative (flagged, correctly, by the code review) but not a functional absence — it is listed as a gap only for the handback-rendering piece, which is fully absent.

None of these nine items were found to regress or break anything that worked before this phase; they are additive machinery that is correct, tested, and disconnected. Given this project's own stated Definition of Done and its repeated documented history of exactly this pattern recurring, these are reported as gaps rather than silently accepted as "built, tested, ready for a later plan."

---

_Verified: 2026-09-10T21:15:00Z_
_Verifier: Claude (gsd-verifier)_
