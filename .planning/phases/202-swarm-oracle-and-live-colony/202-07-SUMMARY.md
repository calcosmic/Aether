---
phase: 202-swarm-oracle-and-live-colony
plan: "07"
subsystem: swarm
tags: [swarm, checkpoint, rollback, three-strike, live-events]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle
    provides: "saveRepairCheckpoint/restoreRepairCheckpoint (cmd/work_repair.go) -- the one proven checkpoint/restore primitive this plan adapts for Swarm, never re-implements."
  - phase: 202-05
    provides: "cmd/swarm_lens.go's swarmComparison/swarmHypothesis/compareSwarmHypotheses -- the structured, ranked repair selection this plan checkpoints around and re-derives per strike for the architectural case."
provides:
  - "cmd/swarm_repair_checkpoint.go: swarmRepairCheckpointIdentity, saveSwarmRepairCheckpoint/restoreSwarmRepairCheckpoint (thin adapters over the Phase 201 primitives), and announceSwarmCheckpointSaved/Restored emitting both plain-English text and live.recovery.changed events."
  - "cmd/swarm_cmd.go: runSwarmDestroy now saves a checkpoint before the fix wave dispatches, restores it on a failed verification (never on a pass), and reports a restore failure as the distinct repair_failed_not_restored status -- never a claimed rollback."
  - "cmd/swarm_cmd.go: the third-strike escalation branch now augments swarmArchitecturalConcernResult's existing payload with a plain-language case naming all three attempts and a structural-change proposal traceable to the recorded hypotheses' shared causes -- cmd/swarm_strikes.go itself is untouched."
affects: [202-08, 202-09, 202-10, 202-11, 202-12, 202-13, 202-14, 202-15]

# Actuals (#2632)
actuals:
  tokens: 12021
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Swarm-shaped adapter over a proven primitive: swarmRepairCheckpointIdentity/saveSwarmRepairCheckpoint/restoreSwarmRepairCheckpoint call saveRepairCheckpoint/restoreRepairCheckpoint directly -- never runBoundedRepairRound or applyBoundedCheckFixRepair, which are the phase-and-check-shaped wrappers Phase 201's continue flow owns."
    - "Restore-seam test variable (swarmRestoreRepairCheckpointFunc, mirroring newSwarmWorkerInvoker) lets a test force the 'restore itself failed' case without corrupting real filesystem state, while production always calls the direct adapter."
    - "Third-strike case built entirely from durable history: loadSwarmResultRecordByID re-reads each strike's own already-persisted swarmResultRecord, and hypothesesFromSwarmRuns/detectSwarmSharedCauses (both from 202-05, unmodified) re-derive the shared cause across attempts -- no worker is dispatched to build the case, and cmd/swarm_strikes.go's counting/persistence path is never touched."

key-files:
  created:
    - cmd/swarm_repair_checkpoint.go
    - cmd/swarm_repair_checkpoint_test.go
  modified:
    - cmd/swarm_cmd.go

key-decisions:
  - "Checkpoint scope is derived from the selected repair's supporting lenses' own evidence Locations (swarmRepairCheckpointPaths), not the whole root by default -- an empty scope (no selected repair, or no lens evidence looked like a file path) falls back to saveRepairCheckpoint's existing whole-root behavior unchanged."
  - "Whether the repair held is decided from the verification wave's own outcome (summarizeSwarmOutcome(watcherRuns)) exclusively, not the run's combined outcome -- so an investigation-wave hiccup earlier in the run can never be conflated with the fix wave's own verification result."
  - "A restore failure is reported as a new, distinct status (repair_failed_not_restored) rather than folded into the existing completed/failed/blocked vocabulary -- the JSON result carries backup_path and a runnable `cp -r` recovery command, and the rollback-succeeded announcement text is never rendered on this path."
  - "The architectural case is additive to swarmArchitecturalConcernResult's existing payload (case/attempts/structural_change_proposal keys added; strike_count/evidence_ids/next and every other existing key kept in shape) so TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates and TestSwarmThreeStrikeExternalFinalizeReplayKeepsOneEscalation pass unmodified."
  - "The structural-change proposal prefers a cause two or more recorded hypotheses independently named across the three attempts (detectSwarmSharedCauses over the combined per-strike hypotheses); absent a shared cause it falls back to the first recorded hypothesis's claim, and absent any hypothesis at all it says plainly that no structural cause could be determined."

patterns-established:
  - "Test seam for a restore-only failure path: a package-level function variable (swarmRestoreRepairCheckpointFunc) wraps the direct adapter call at exactly one call site, so a test can force a real, otherwise-unreachable failure branch without touching the filesystem."

requirements-completed: [LIVE-04]

coverage:
  - id: D1
    description: "Swarm saves a checkpoint, announced before it happens, before the ranked repair is applied."
    requirement: "LIVE-04"
    verification:
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmCheckpointAnnouncementsAreOrdered"
        status: pass
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmRepairCheckpointsBeforeTheFixWave"
        status: pass
    human_judgment: false
  - id: D2
    description: "The ranked repair is applied through the existing authorized dispatch path and then re-verified; verification failure restores the checkpoint exactly and announces the restore."
    requirement: "LIVE-04"
    verification:
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmRepairRollsBackOnFailedVerification"
        status: pass
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmCheckpointRoundTripsDeclaredPaths"
        status: pass
    human_judgment: false
  - id: D3
    description: "A restore that itself fails stops the run, states plainly the project was not put back, names the saved copy's location and the recovery command, and never reports a rollback that did not happen."
    requirement: "LIVE-04"
    verification:
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmRestoreFailureIsReportedHonestly"
        status: pass
    human_judgment: false
  - id: D4
    description: "No mid-flight approval prompt appears during the automatic repair, on either the passing or the failing verification branch."
    requirement: "LIVE-04"
    verification:
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSwarmRepairAsksNothingMidFlight"
        status: pass
    human_judgment: false
  - id: D5
    description: "The third consecutive failure still fires the existing three-strike escalation and now renders a plain-language case naming all three attempts, why patching is not working, and the proposed structural change -- sourced from recorded evidence, with zero workers dispatched and cmd/swarm_strikes.go untouched."
    requirement: "LIVE-04"
    verification:
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestThirdStrikeRendersAnArchitecturalCase"
        status: pass
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestArchitecturalCaseNamesAllThreeAttempts"
        status: pass
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestArchitecturalCaseProposalComesFromRecordedEvidence"
        status: pass
      - kind: unit
        ref: "cmd/swarm_repair_checkpoint_test.go#TestSecondStrikeDoesNotEscalate"
        status: pass
      - kind: unit
        ref: "cmd/swarm_cmd_test.go#TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates"
        status: pass
      - kind: unit
        ref: "cmd/swarm_cmd_test.go#TestSwarmThreeStrikeExternalFinalizeReplayKeepsOneEscalation"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 07: Swarm Repair Checkpoint, Rollback, and Architectural Case Summary

**Swarm's automatic repair is now bracketed by Phase 201's proven checkpoint/restore primitive (save before the fix wave, restore only on failed verification, honest not-restored reporting if restore itself fails), and the third consecutive failure on one target renders a plain-language case instead of a bare refusal.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-09-11T11:05:00Z
- **Completed:** 2026-09-11T11:45:17Z
- **Tasks:** 3
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- `cmd/swarm_repair_checkpoint.go`: a Swarm-shaped adapter (`swarmRepairCheckpointIdentity`, `saveSwarmRepairCheckpoint`, `restoreSwarmRepairCheckpoint`) over Phase 201's `saveRepairCheckpoint`/`restoreRepairCheckpoint` -- never a second checkpoint mechanism, and never the phase-and-check-shaped `runBoundedRepairRound`/`applyBoundedCheckFixRepair` wrappers. Checkpoint scope is derived from the selected repair's supporting lenses' evidence, falling back to a whole-root checkpoint when no scope is known.
- `announceSwarmCheckpointSaved`/`announceSwarmCheckpointRestored`: plain-English save/restore announcements plus matching `live.recovery.changed` events (via the existing `emitColonyLiveRecoveryChanged` per-lane helper) with `RecoveryState` values `checkpoint_saved`/`checkpoint_restored`, so the live cockpit sees both moments in the replayed stream.
- `runSwarmDestroy` (`cmd/swarm_cmd.go`): saves a checkpoint immediately before the fix wave dispatches (only when a repair was actually selected), decides whether the repair held from the verification wave's own outcome alone, releases the checkpoint on a pass, restores it on a fail, and -- when the restore itself fails -- stops with a distinct `repair_failed_not_restored` status naming the saved copy's directory and a runnable `cp -r` recovery command, never describing it as a rollback that happened. No interactive prompt appears at any point.
- Third-strike escalation (`augmentSwarmArchitecturalCase`): additive to `swarmArchitecturalConcernResult`'s existing payload (`case`, `attempts`, `structural_change_proposal` keys added; every existing key kept in shape). Reads each strike's durable `swarmResultRecord` (`loadSwarmResultRecordByID`) to render what each of the three attempts tried and how it failed, and reuses 202-05's `hypothesesFromSwarmRuns`/`detectSwarmSharedCauses` across the three attempts' recorded workers to source the proposed structural change from a corroborated shared cause. `cmd/swarm_strikes.go` (the counting/persistence machinery) is untouched.

## Task Commits

1. **Task 1: A Swarm-shaped adapter over the one checkpoint primitive** - `8c479148` (feat)
2. **Task 2: Bracket the fix wave with save, verify and conditional rollback** - `5e49b936` (feat)
3. **Task 3: Turn the third strike into a plain-language architectural case** - `6658d345` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/swarm_repair_checkpoint.go` - Swarm's checkpoint identity, save/restore adapters, and save/restore announcements+live events
- `cmd/swarm_repair_checkpoint_test.go` - all new tests for Tasks 1-3 (checkpoint identity/round-trip/announcements, fix-wave bracketing/rollback/restore-failure/no-prompt, architectural case rendering/naming/proposal-traceability/second-strike-no-escalation)
- `cmd/swarm_cmd.go` - `runSwarmDestroy`'s fix-wave bracket (save/verify/restore/not-restored), `swarmRestoreRepairCheckpointFunc` test seam, `swarmRepairNotRestoredStatus` constant, and the escalation branch's `augmentSwarmArchitecturalCase` plus its `loadSwarmResultRecordByID`/`swarmArchitecturalAttempts`/`swarmArchitecturalStructuralChangeProposal` helpers

## Decisions Made

- Checkpoint scope comes from the selected repair's supporting lenses' evidence locations, not the whole root by default (falls back to whole-root when no scope is known) -- keeps the checkpoint tight to what the repair is actually expected to touch.
- Repair-held/failed is decided from the verification wave's own outcome exclusively, never the run's combined outcome, so an investigation-wave hiccup can never be mistaken for a failed fix.
- A restore failure gets its own distinct status (`repair_failed_not_restored`) rather than reusing `failed`/`blocked`, so a caller can never mistake "the fix failed and we restored" for "the fix failed and we could not confirm the project was put back."
- The architectural case is additive-only to the existing escalation payload, so both pre-existing three-strike tests (`TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates`, `TestSwarmThreeStrikeExternalFinalizeReplayKeepsOneEscalation`) pass unmodified.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- LIVE-04 is satisfied: Swarm's automatic repair now carries the same checkpoint/rollback safety net every other repair lane uses, and the three-strike escalation reads as a case an owner can act on.
- `cmd/swarm_strikes.go` remains untouched, so any later plan that builds on the strike-counting/persistence machinery inherits it unchanged.
- Ready for the next Phase 202 wave-4 plan.

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*

## Self-Check: PASSED

- FOUND: cmd/swarm_repair_checkpoint.go
- FOUND: cmd/swarm_repair_checkpoint_test.go
- FOUND: .planning/phases/202-swarm-oracle-and-live-colony/202-07-SUMMARY.md
- FOUND: 8c479148 (Task 1 commit)
- FOUND: 5e49b936 (Task 2 commit)
- FOUND: 6658d345 (Task 3 commit)
