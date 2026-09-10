---
phase: 201-queen-led-work-cycle
plan: "17"
subsystem: infra
tags: [go, cli, cobra, build-orchestration, verification, repair]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "buildFailedRepairHandback/applyBoundedCheckFixRepair (cmd/work_repair.go, plan 201-09) — the tested-but-unwired four-part handback assembler and the checkpoint/restore repair wrapper this plan connects and extends"
provides:
  - "applyBoundedCheckFixRepair's third return value, *repairHandback — non-nil only on the still-failing branch, assembled from the re-run floor's own first failing step, the fix attempt's own recorded reason, and whether the restore actually succeeded"
  - "renderFailedRepairHandback: a four-part plain-English block (what is failing, what was tried and why it did not work, where the project stands now, what to do next) for the owner's screen"
  - "codexContinueVerificationReport.RepairHandback, carried from applyBoundedCheckFixRepair into the report both continue lanes render from"
  - "renderContinueBlockedVisual renders the handback (dual-typed for the in-process and JSON-round-tripped shapes) after the blocking issues and before the closing next-step line, on both continue lanes that share this one renderer"
  - "TestFailedRepairHandsBackAllFourParts: end-to-end proof that a genuinely failing check drives all four handback parts onto the blocked screen, with the two negative cases (fix succeeds; no eligible fix attempt) rendering nothing"
  - "TestFailedRepairHandbackHasProductionCallers: an AST call-graph guard proving both buildFailedRepairHandback and renderFailedRepairHandback have production callers, with a synthetic-removal sub-case proving the guard can fail"
affects: [201-queen-led-work-cycle]

# Actuals (#2632)
actuals:
  tokens: 7076
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dual-type rendering: a new report field is read from result[\"verification\"] via a switch on the in-process typed struct vs. the JSON-round-tripped map[string]interface{}, matching renderContinueVerificationDetail's own established precedent (repairHandbackFromVerificationValue)"
    - "Counter-file and marker-file shell fixtures for a genuinely-driven pass/fail transition in a test, instead of a hand-typed outcome literal — the deterministic floor's own two real invocations flip the result, proven against the real criterion-evidence gate (phaseCriterionEvidencePolicy/flattenPhaseCriterionEvidenceRequirements) rather than an empty manifest that would mask it"

key-files:
  created:
    - cmd/work_repair_handback_test.go
  modified:
    - cmd/work_repair.go
    - cmd/codex_continue.go
    - cmd/codex_visuals.go

key-decisions:
  - "The still-failing branch's checkpoint restore result (outcome.Restored) now drives buildFailedRepairHandback's own restored-position wording — a failed restore says the safe position could not be confirmed, rather than always claiming a restore that may not have happened. This existing function was silently wrong for that edge case before any caller reached it."
  - "The diagnosis is built from the re-run floor's own first failing step (firstFailingVerificationStep on the post-fix-attempt floor), never the pre-fix floor and never a literal — matching the plan's determinism requirement (fixed build/types/lint/tests order) when more than one step fails."
  - "A checkpoint-save failure or a buildFailedRepairHandback assembly error both fall back to no handback (never a crash) — consistent with the existing pre-201-09 no-checkpoint fallback and the plan's explicit prohibition against turning a blocked check into a crash."
  - "The 'fix attempt succeeds' test sub-case is driven through the real production pipeline (not a hand-built report) by supplying a bound criterion-evidence manifest derived from the same phaseCriterionEvidencePolicy/flattenPhaseCriterionEvidenceRequirements calls production uses at cmd/codex_build.go:2900-2901 — an empty manifest independently blocks ChecksPassed via the criteria gate regardless of the shell checks, which would have made a genuine 'fixed' outcome unreachable through this fixture family."

requirements-completed: [WORK-06]

coverage:
  - id: D1
    description: "applyBoundedCheckFixRepair returns a third value, *repairHandback, non-nil only on the still-failing branch, assembled from the re-run floor's own failing step, the fix attempt's own recorded reason, and the real restore outcome — never a hand-typed literal"
    requirement: "WORK-06"
    verification:
      - kind: unit
        ref: "cmd -run TestRepairRunsAtMostOnceAutomatically"
        status: pass
      - kind: unit
        ref: "cmd -run TestFailureBornSignalReachesTheRepairBrief"
        status: pass
      - kind: integration
        ref: "cmd/work_repair_handback_test.go#TestFailedRepairHandsBackAllFourParts"
        status: pass
    human_judgment: false
  - id: D2
    description: "codexContinueVerificationReport carries RepairHandback, and renderContinueBlockedVisual renders the four-part block (dual-typed) after the blocking issues and before the closing next-step line, on both continue lanes that share this renderer; a nil handback renders nothing"
    requirement: "WORK-06"
    verification:
      - kind: unit
        ref: "cmd -run TestBothCheckLanesLeaveTheSameOutcomeText"
        status: pass
      - kind: unit
        ref: "cmd -run TestAllThreeContinueLanesShareOneAcceptVerifyAdvanceBody"
        status: pass
      - kind: integration
        ref: "cmd/work_repair_handback_test.go#TestFailedRepairHandsBackAllFourParts"
        status: pass
    human_judgment: false
  - id: D3
    description: "A genuinely failing check, driven end-to-end with no reviewer dispatched, produces a handback whose four parts each independently derive from a real runtime call (repairCheckpointIdentity, compactFailureExcerpts, recommendedActionForWorkOutcome), and the rendered blocked screen carries all four; a fix that succeeds or no eligible fix attempt renders nothing, on both the typed and JSON-round-tripped shapes"
    requirement: "WORK-06"
    verification:
      - kind: integration
        ref: "cmd/work_repair_handback_test.go#TestFailedRepairHandsBackAllFourParts"
        status: pass
    human_judgment: false
  - id: D4
    description: "buildFailedRepairHandback and renderFailedRepairHandback each have at least one direct production caller, proven by a call-graph guard (not a hardcoded list), with a synthetic-removal sub-case demonstrating the guard can fail by name and position"
    requirement: "WORK-06"
    verification:
      - kind: unit
        ref: "cmd/work_repair_handback_test.go#TestFailedRepairHandbackHasProductionCallers"
        status: pass
    human_judgment: false

duration: 27min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 17: Close the D-11 Gap — the Failed-Repair Handback Summary

**When the one automatic repair still fails, the owner's screen now says plainly what is failing, what was tried and why it did not work, where the project stands, and the one thing to do next — closing the gap where all four parts existed and were unit-tested, but nothing ever called them.**

## Performance

- **Duration:** 27 min
- **Started:** 2026-09-10T22:53:11+02:00 (approx.)
- **Completed:** 2026-09-10T23:20:21+02:00
- **Tasks:** 3 completed
- **Files modified:** 3 modified, 1 created

## Accomplishments

- `applyBoundedCheckFixRepair` (`cmd/work_repair.go`) now returns a third value, `*repairHandback`, non-nil only on the still-failing branch. The diagnosis comes from `firstFailingVerificationStep` on the *re-run* floor (deterministic build/types/lint/tests order when more than one step fails), the attempted action from the fix attempt's own recorded `Reason`, and the restored position from whether `restoreRepairCheckpoint` actually succeeded — fixing a pre-existing bug in `buildFailedRepairHandback` where the restored-position text always claimed a successful restore regardless of the real outcome.
- Added `renderFailedRepairHandback`, a four-part plain-English block (what is failing / what was tried and why it did not work / where the project stands now / what to do next, with its reason and alternatives) written for a reader who has never opened a file here.
- `codexContinueVerificationReport` gained `RepairHandback`, populated from the new return value at its one call site in `runCodexContinueVerification`. `renderContinueBlockedVisual` renders the handback (via a new dual-typed reader, `repairHandbackFromVerificationValue`, matching the existing `renderContinueVerificationDetail` precedent) after the blocking issues and before the closing next-step line — additive only, and reached by both continue lanes through the one shared renderer.
- `cmd/work_repair_handback_test.go` (new): `TestFailedRepairHandsBackAllFourParts` drives a real, genuinely failing check (a fixture whose tests command produces real captured output, unlike the pre-existing `false`-based fixtures which always yield an empty `Output`) through the production path with no reviewer dispatched, and asserts each of the four parts independently, with every expected value derived from the same runtime calls production uses — never a typed literal. Both negative sub-cases (a fix attempt that genuinely succeeds, driven through a bound criterion-evidence manifest so the pass is real rather than manufactured; and no eligible fix attempt at all) render no handback section, verified on both the typed and JSON-round-tripped shapes. `TestFailedRepairHandbackHasProductionCallers` is an AST call-graph guard (reusing `continueDecisionPackageFuncs`/`continueDecisionDirectCallers` from plan 201-16) proving both new functions have a real caller, with a synthetic-removal sub-case proving the guard can fail by name and position.

## Task Commits

Each task was committed atomically:

1. **Task 1: Assemble the handback from the repair that actually ran** - `57158d5f` (feat)
2. **Task 2: Carry the handback onto the blocked check screen both lanes render** - `c5937beb` (feat)
3. **Task 3: Prove the handback reaches the owner with all four parts** - `bb928455` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/work_repair.go` - `applyBoundedCheckFixRepair` returns a third `*repairHandback` value; `buildFailedRepairHandback`'s restored-position text now branches on `outcome.Restored`; `repairHandback` gained JSON tags; added `renderFailedRepairHandback`.
- `cmd/codex_continue.go` - `codexContinueVerificationReport.RepairHandback` field; the call site threads the third return value through.
- `cmd/codex_visuals.go` - `repairHandbackFromVerificationValue` (dual-typed reader) and its one call site inside `renderContinueBlockedVisual`.
- `cmd/work_repair_handback_test.go` (new) - `TestFailedRepairHandsBackAllFourParts` and `TestFailedRepairHandbackHasProductionCallers`, plus three fixture builders (`handbackStillFailingFixture`, `handbackFixSucceedsFixture`, `handbackNoEligibleFixFixture`).

## Decisions Made

- Fixed a latent bug in `buildFailedRepairHandback` while wiring it: the restored-position text was always the "put back exactly" wording regardless of whether the restore call actually succeeded. Now branches on `outcome.Restored`, matching the plan's explicit behavior requirement.
- Chose to reuse `firstFailingVerificationStep` (already public in `cmd/check_fix_attempt.go`, D-02's own precedent) for the diagnosis's failing-step selection rather than adding a second implementation.
- Built the "fix attempt succeeds" test sub-case as a genuine production pass (real second shell invocation flips a counter file to a passing exit code, with a properly bound criterion-evidence manifest) rather than hand-constructing a `codexContinueVerificationReport{Outcome: "fixed"}` literal — discovered mid-task that an empty manifest independently blocks `ChecksPassed` via the criterion-evidence gate (`phaseCriterionEvidencePolicy` returns `bound_v1` for this fixture family's phase), which would have made a real "fixed" outcome unreachable and forced a less rigorous, hand-typed fixture.

## Deviations from Plan

**1. [Rule 1 - Bug] Fixed `buildFailedRepairHandback`'s restored-position wording to reflect the real restore outcome**
- **Found during:** Task 1
- **Issue:** The function's existing (pre-201-17) restored-position string always read "Your project has been put back exactly to the state it was saved in" regardless of `outcome.Restored`. The plan's own behavior requirement (`<behavior>`, Task 1) explicitly names this: "A failure to restore the checkpoint still produces a handback, and its restored-position wording says the safe position could not be confirmed rather than claiming a restore that did not happen."
- **Fix:** Added a branch on `outcome.Restored` with a distinct, honest message for the failed-restore case.
- **Files modified:** `cmd/work_repair.go`
- **Commit:** `57158d5f`

No other deviations — the rest of the plan executed as written.

## Issues Encountered

None blocking. One investigation detour: the initial "fix attempt succeeds" test sub-case, built with `codexContinueManifest{}` (matching the existing `false`-fixture precedent in `cmd/floor_fix_attempt_test.go`), consistently reported `ChecksPassed=false` even after the tests step itself genuinely passed on the second run. Traced to `evaluatePhaseCriterionEvidence`'s independent `bound_v1` criterion-evidence gate, which blocks on a missing/mismatched manifest regardless of the shell checks. Resolved by deriving a correctly-bound manifest from the same production calls (`phaseCriterionEvidencePolicy`, `flattenPhaseCriterionEvidenceRequirements`) rather than working around it with a hand-typed manifest shape.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

D-11 (the failed-repair four-part handback reaching production) is closed: `buildFailedRepairHandback` and `renderFailedRepairHandback` each now have real production callers, proven by an AST guard that can fail, and by an end-to-end test driving a genuinely failing check to a rendered blocked screen carrying all four parts. The remaining gaps in `201-VERIFICATION.md` (the closeout-rendering root cause — `LifecycleCloseoutDetails.WorkOutcome` never set in production, affecting D-05/D-07/D-08's rendering/CAP-066/D-06 for the `aether run` autopilot lane — plus `runBoundedRepairRound`'s continued non-use in favor of the parallel inline `applyBoundedCheckFixRepair` implementation) are unaddressed by this plan and remain tracked as separate gap-closure plans (201-18..201-20 per STATE.md).

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

All claimed files exist (`cmd/work_repair.go`, `cmd/codex_continue.go`, `cmd/codex_visuals.go`, `cmd/work_repair_handback_test.go`, this SUMMARY.md) and all three task commit hashes (`57158d5f`, `c5937beb`, `bb928455`) are present in `git log --oneline --all`.
