---
phase: 201-queen-led-work-cycle
plan: "16"
subsystem: infra
tags: [go, cli, cobra, build-orchestration, verification]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "queenApplyVerificationBoundary/attachVerificationBoundary/verificationBoundaryForAttempt (cmd/verification_boundary.go, plan 201-03) — the tested-but-unwired reconciliation, write, and read functions this plan connects to production"
provides:
  - "aether build --verification-boundary/--boundary-why: a real input channel for the Queen's build-end-vs-check-step proposal, on both the plan-only and direct-dispatch build lanes"
  - "Exactly-once reconciliation (queenApplyVerificationBoundary) per build lane invocation, persisted onto the exact attempt via attachVerificationBoundary immediately after commitBuildStart on both lanes"
  - "A trailing variadic boundary seam on plannedBuildDispatchesWithJobProposals/queenBuildPostWaveDispatches so a build-end choice governs the SAME build's own post-wave reviewer dispatches, not only the following build's"
  - "TestBoundaryWriteSideHasProductionCallers: an AST call-graph guard (reusing continueDecisionPackageFuncs/continueDecisionDirectCallers) proving both write-side functions have production callers and both build lane entry points reach the recording path"
  - "TestQueenBoundaryChoiceReachesTheRecordedAttempt: end-to-end proof against three real plan-only builds that the recorded decision and the build's own dispatch list agree"
affects: [201-queen-led-work-cycle]

# Actuals (#2632)
actuals:
  tokens: 8297
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Trailing variadic decision-seam threading: a caller that already reconciled a pure decision passes it as an optional trailing arg so a shared planning function can prefer it over its own fallback read, with zero signature break for every existing caller"

key-files:
  created:
    - cmd/verification_boundary_wiring_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_workflow_cmds.go

key-decisions:
  - "Reconciliation call sites live inside prepareDirectCodexBuild (direct lane) and runCodexBuildPlanOnlyWithOptions (plan-only lane), matching the plan's named line numbers, rather than hoisting to the outer runCodexBuildWithOptions — this keeps the decision co-located with the dispatch planning it governs and reuses directCodexBuildPreparation as the carrier back to the outer function's persist step."
  - "queenBuildPostWaveDispatches keeps its original loadLatestBuildAttempt+verificationBoundaryForAttempt fallback behind the variadic boundary param so every existing caller (tests included) is unaffected; only the two production planning call sites now supply the freshly-reconciled decision."
  - "A verification-boundary write failure after commitBuildStart fails the build start and marks the attempt failed/interrupted via the existing finishAttempt/defer machinery on the direct lane, matching how this file already treats other evidentiary write failures on the build path."

requirements-completed: [WORK-01, WORK-04]

coverage:
  - id: D1
    description: "aether build gains --verification-boundary/--boundary-why flags that reach codexBuildOptions unchanged on both build lanes, with no repo-jargon enum spelling in the help text"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "cmd -run TestCLIFlagAudit (pre-existing unrelated failure noted below; flag registration itself verified by go build + go vet clean, and by TestQueenBoundaryChoiceReachesTheRecordedAttempt exercising both flags end-to-end)"
        status: pass
      - kind: integration
        ref: "cmd/verification_boundary_wiring_test.go#TestQueenBoundaryChoiceReachesTheRecordedAttempt"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both build lanes reconcile the boundary exactly once and persist the same decision onto the exact attempt via attachVerificationBoundary, failing the build start on a write failure"
    requirement: "WORK-01"
    verification:
      - kind: unit
        ref: "cmd -run TestVerificationBoundaryDefaultsToTheCheckStep"
        status: pass
      - kind: integration
        ref: "cmd/verification_boundary_wiring_test.go#TestQueenBoundaryChoiceReachesTheRecordedAttempt"
        status: pass
    human_judgment: false
  - id: D3
    description: "A recorded build-end choice governs the SAME build's own post-wave reviewer dispatch list (not only the following build's), while a no-proposal build dispatches none and TestOneFunctionDerivesTheVerificationBoundary/TestPhaseVerifiedOnce stay unchanged"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd -run TestOneFunctionDerivesTheVerificationBoundary"
        status: pass
      - kind: unit
        ref: "cmd -run TestPhaseVerifiedOnce"
        status: pass
      - kind: unit
        ref: "cmd -run TestBuildEndReviewersGateOnTheRecordedBoundary"
        status: pass
      - kind: unit
        ref: "cmd -run TestCheckStepReviewersGateOnTheRecordedBoundary"
        status: pass
      - kind: integration
        ref: "cmd/verification_boundary_wiring_test.go#TestQueenBoundaryChoiceReachesTheRecordedAttempt"
        status: pass
    human_judgment: false
  - id: D4
    description: "A call-graph guard proves queenApplyVerificationBoundary and attachVerificationBoundary each have production callers, both build lane entry points reach the recording path, and a synthetic disconnection is refused by name and file position"
    requirement: "WORK-04"
    verification:
      - kind: unit
        ref: "cmd/verification_boundary_wiring_test.go#TestBoundaryWriteSideHasProductionCallers"
        status: pass
    human_judgment: false

duration: 16min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 16: Wire the Queen's Verification-Boundary Choice into Production Summary

**`aether build --verification-boundary build_end --boundary-why "<reason>"` now actually reaches, reconciles once, and persists onto the real build attempt — closing the D-01 gap where the choice was fully built and unit-tested but never called from production.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-09-10T22:35:00+02:00 (approx.)
- **Completed:** 2026-09-10T22:51:18+02:00
- **Tasks:** 3 completed
- **Files modified:** 2 modified, 1 created

## Accomplishments

- Added `QueenVerificationBoundary`/`QueenVerificationBoundaryWhy` to `codexBuildOptions` and registered `--verification-boundary`/`--boundary-why` on `buildCmd`, threaded verbatim (no validation at the flag layer) into both the plan-only and direct-dispatch lanes.
- Both build lanes (`prepareDirectCodexBuild` for the direct lane, `runCodexBuildPlanOnlyWithOptions` for the plan-only lane) now call `queenApplyVerificationBoundary` exactly once, before dispatches are planned, hold the result in a local variable, and pass it into `plannedBuildDispatchesWithJobProposals` → `queenBuildPostWaveDispatches` via a new trailing variadic `boundary ...verificationBoundaryDecision` parameter — so a build-end choice governs that build's own post-wave reviewer dispatches instead of only the next build's.
- `attachVerificationBoundary(receipt.AttemptPath, decision)` now runs immediately after `commitBuildStart` succeeds on both lanes; a write failure fails the build start (direct lane also marks the attempt failed via the existing `finishAttempt` machinery), matching how this file already treats other evidentiary write failures.
- `cmd/verification_boundary_wiring_test.go` (new): an AST call-graph guard proving both write-side functions have production callers and both build lane entry points reach the recording path (with a synthetic-disconnection sub-case proving the guard can fail), plus an end-to-end test driving three real plan-only builds that confirms the recorded decision and the build's own dispatch list agree.

## Task Commits

Each task was committed atomically:

1. **Task 1: Give the Queen a boundary proposal channel on aether build** - `4af3dae1` (feat)
2. **Task 2: Reconcile the boundary once and record it on the attempt it belongs to** - `8a3f2c3b` (feat)
3. **Task 3: Prove the boundary write side is wired and cannot quietly unwire** - `bad61074` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/codex_build.go` - Added the two `codexBuildOptions` boundary fields; `directCodexBuildPreparation.VerificationBoundary` carrier field; reconciliation call sites in `prepareDirectCodexBuild` and `runCodexBuildPlanOnlyWithOptions`; `attachVerificationBoundary` persistence after both `commitBuildStart` calls; trailing variadic `boundary` param on `plannedBuildDispatchesWithJobProposals` and `queenBuildPostWaveDispatches`.
- `cmd/codex_workflow_cmds.go` - Registered `--verification-boundary`/`--boundary-why` flags on `buildCmd` with plain-English help text (no underscore-spelled enum); read the flags and threaded them into both `codexBuildOptions` construction sites in `buildCmd`'s `RunE`.
- `cmd/verification_boundary_wiring_test.go` (new) - `TestBoundaryWriteSideHasProductionCallers` (AST call-graph guard) and `TestQueenBoundaryChoiceReachesTheRecordedAttempt` (three real plan-only builds).

## Decisions Made

- Kept the reconciliation call sites inside `prepareDirectCodexBuild` and `runCodexBuildPlanOnlyWithOptions` (matching the plan's named line numbers) rather than hoisting the call to the outer `runCodexBuildWithOptions`, since the direct lane calls `prepareDirectCodexBuild` twice (a discarded readiness rehearsal, then the real preparation) — the decision travels back to the outer function via the new `directCodexBuildPreparation.VerificationBoundary` field from the real (second) call, so the persisted decision is always the one the real dispatch list was actually planned against.
- `queenBuildPostWaveDispatches` keeps its original `loadLatestBuildAttempt` + `verificationBoundaryForAttempt` fallback behind the new variadic parameter, so every existing caller (including `cmd/boundary_double_dispatch_test.go` and `cmd/codex_build_test.go`) is unaffected by this change.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

`go test ./cmd -run '^TestCLIFlagAudit$'` (Task 1's own named verify command) fails on a pre-existing, unrelated mismatch (`.claude/commands/ant/help.md:9: subcommand "help" not registered in Go runtime`) that predates this plan — confirmed via `git stash` + rerun on the unmodified tree, same failure. Out of scope per the deviation rules' scope boundary (only auto-fix issues directly caused by this task's changes); not fixed here, and not re-flagged as a new gap since it is unrelated to the verification-boundary wiring this plan closes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

D-01 (the Queen's build-end-vs-check-step choice reaching production) is closed: `queenApplyVerificationBoundary` and `attachVerificationBoundary` now each have real production callers, proven by an AST guard that can fail, and by three real builds whose recorded attempt and dispatch list agree. The remaining eight gaps in `201-VERIFICATION.md` (the closeout-rendering root cause: `LifecycleCloseoutDetails.WorkOutcome` never set in production, affecting D-03/D-05/D-06(run)/D-07/D-08's rendering/D-11/CAP-066) are unaddressed by this plan and are tracked as separate gap-closure plans (201-17..201-20 per STATE.md).

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

All claimed files exist (`cmd/codex_build.go`, `cmd/codex_workflow_cmds.go`, `cmd/verification_boundary_wiring_test.go`, this SUMMARY.md) and all three task commit hashes (`4af3dae1`, `8a3f2c3b`, `bad61074`) are present in `git log --oneline --all`.
