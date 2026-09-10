---
phase: 201-queen-led-work-cycle
plan: "09"
subsystem: queen-orchestration
tags: [go, repair, checkpoint, autopilot, continue, closeout, D-09, D-10, D-11]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 02)
    provides: "cmd/codex_verify_advance.go: runContinueAcceptVerifyAdvance, the shared accept/verify/advance decision body the direct continue lane reaches through"
  - phase: 201-queen-led-work-cycle (plan 04)
    provides: "pkg/colony/work_outcome.go: colony.WorkOutcome (six-verdict vocabulary) and the equal-ceremony closeout this plan's failed-repair handback renders through"
provides:
  - "cmd/work_repair.go: runBoundedRepairRound, the one bounded checkpointed repair path (save a checkpoint, one repair wave, one re-verification, exact restore on failure) build and check both use"
  - "cmd/work_repair.go: repairCheckpointIdentity / findAutopilotRepairReceiptByCheckpoint -- the idempotency-key replay mechanism (SYN-201-10) reusing the existing autopilotRepairLedger rather than a second budget tracker"
  - "cmd/work_repair.go: saveRepairCheckpoint / restoreRepairCheckpoint / repairCheckpointDirectoryDigest -- a scoped directory snapshot/restore built on medic's own backupCopyFile/backupCopyDir helpers"
  - "cmd/work_repair.go: applyBoundedCheckFixRepair -- wires the D-09 checkpoint/restore round and D-10 flow announcements around the existing D-02 check-fix attempt, wired into the direct continue check lane (cmd/codex_continue.go)"
  - "cmd/work_repair.go: repairHandback / buildFailedRepairHandback -- D-11's four-element failed-repair handback, rendered through the 201-04 equal-ceremony closeout and reusing the 201-07 recommendedActionForWorkOutcome mechanism"
affects: [201-10]

# Actuals (#2632)
actuals:
  tokens: 9900
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Idempotency-key replay (SYN-201-10): repairCheckpointIdentity(phase, check) mirrors colony.PauseHandoff.HandoffID/resumeTransactionID's own replay pattern -- a checkpoint identity that already has a receipt is a spent round, refused by name, whether the caller is genuinely retrying or replaying the identical call. One mechanism serves both D-09 guarantees (no second attempt, replay returns the existing receipt) rather than two."
    - "Wrap, don't rewrite: applyBoundedCheckFixRepair wraps the existing, already-tested D-02 applyAutomaticCheckFixAttempt with checkpoint/announce/restore, reusing planCheckFixAttempt's own eligibility gate read-only rather than re-deriving it -- minimizes the blast radius on a widely-exercised continue code path."
    - "Reuse over reinvention: checkpoint save/restore reuses medic_repair.go's backupCopyFile/backupCopyDir; the failed-repair diagnosis reuses check_fix_attempt.go's compactFailureExcerpts; the one owner action reuses lifecycle_closeout.go's recommendedActionForWorkOutcome -- no parallel implementation of any of the three."

key-files:
  created:
    - cmd/work_repair.go
    - cmd/work_repair_test.go
  modified:
    - cmd/autopilot_policy.go
    - cmd/codex_continue.go

key-decisions:
  - "The checkpoint identity mechanism follows SYN-201-10's ruling in spirit rather than literally embedding colony.PauseHandoff: PauseHandoff is tied to /ant-pause's full lifecycle-transaction machinery (repository evidence, worktree evidence, a committed transaction with declared writes across COLONY_STATE.json/session.json/CONTEXT.md/HANDOFF.md) built for an owner-invoked, whole-colony pause -- fitting that machinery around a verification-triggered, per-check repair round would have meant either forcing an inline repair through the full pause/resume transaction coordinator (a large, risky rewrite of a proven, unrelated subsystem) or building a second, parallel PauseHandoff-shaped record, which SYN-201-10 explicitly rules out (\"rather than inventing a distinct primitive\"). The idempotency-key REPLAY PATTERN -- a durable identity, a lookup before any new effect, a replay that returns the existing result instead of re-running -- is what SYN-201-10 actually asks to be reused, and that is what repairCheckpointIdentity/findAutopilotRepairReceiptByCheckpoint implement, on top of the ALREADY-GENERALIZED autopilotRepairLedger/autopilotRepairReceipt types (the plan's own explicit instruction) rather than a second ledger."
  - "Checkpoint identity is keyed on (phase, check) only, not the attempt ID that discovered the failure -- matching checkFixAttemptRecord's own established 'one per phase per check, ever' discipline (maxAutomaticCheckFixAttempts). This is what makes 'a second automatic attempt for the same failing verification is refused by name' and 'replaying the same checkpoint identity returns the existing receipt' the same mechanism rather than two: both are 'this (phase, check) pair already has a receipt.'"
  - "Checkpoint save/restore snapshots the caller's declared permitted scope (the files a check-fix attempt's implicated tasks already claimed, via repairScopePathsForCheckFix -- never a fresh guess), not the whole repository, for both correctness (byte-identical restore, proven by directory digest in TestFailedRepairRestoresTheCheckpointExactly) and production safety (no full-repo copy on every continue run)."
  - "applyBoundedCheckFixRepair falls back to the pre-201-09 behavior (call applyAutomaticCheckFixAttempt directly, no checkpoint, no restore) if saving the checkpoint itself fails, rather than refusing the fix attempt outright -- a checkpoint that cannot be taken must never block the existing, already-proven D-02 mechanism."
  - "The failed-repair handback (Task 3) uses colony.WorkOutcomeBlocker and reuses recommendedActionForWorkOutcome's existing blocker branch (fed the diagnosis via a fixture-shaped buildAttemptRecord.Error) rather than adding a second recommendation path, per the plan's explicit prohibition."

requirements-completed: []
# WORK-06 (this plan's declared requirement) is also declared by sibling
# plan 201-10, which has not yet produced a SUMMARY.md -- per the shared-ID
# gate (#2388) WORK-06 stays open in REQUIREMENTS.md until 201-10 finishes
# too. This is expected, not a gap in this plan's own work.

coverage:
  - id: D1
    description: "A failing verification with an eligible repair produces exactly one checkpoint, one repair wave and one re-verification; a second automatic attempt for the same failing verification is refused by name; a failed repair restores the working tree byte-identically; a passed repair leaves the repaired files in place; replaying the same checkpoint identity returns the existing receipt; the original failing attempt record is unchanged and the repair record names it as parent"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/work_repair_test.go#TestRepairRunsAtMostOnceAutomatically"
        status: pass
      - kind: unit
        ref: "cmd/work_repair_test.go#TestFailedRepairRestoresTheCheckpointExactly"
        status: pass
      - kind: unit
        ref: "cmd/work_repair_test.go#TestRepairReplayReturnsTheExistingReceipt"
        status: pass
      - kind: unit
        ref: "cmd/work_repair_test.go#TestRepairNeverOverwritesTheOriginalResult"
        status: pass
    human_judgment: false
  - id: D2
    description: "The checkpoint save announcement appears in the rendered flow before the repair dispatch, and the restore announcement appears after a failed re-verification and not after a passed one, in plain English with no repository-invented vocabulary"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/work_repair_test.go#TestCheckpointSaveAndRestoreAreAnnouncedOnBothLanes"
        status: pass
      - kind: unit
        ref: "cmd/work_repair_test.go#TestPassedRepairAnnouncesNoRestore"
        status: pass
    human_judgment: true
    rationale: "The two named tests prove the announcement mechanism (runBoundedRepairRound + emitRepairCheckpointSaved/Restored) is correct, ordered, and lane-agnostic, and that production wiring reaches the direct check lane (applyBoundedCheckFixRepair, cmd/codex_continue.go). They do not exercise the plan-only/finalize snapshot lane's own real dispatch pipeline end-to-end -- that lane never called the underlying D-02 check-fix mechanism before this plan either (a pre-existing gap), and wiring real dispatch-capable repair coverage into it is out of this plan's declared file scope. See Deviations below."
  - id: D3
    description: "A failed-repair handback carries all four D-11 elements as named fields (diagnosis derived from the failing check's own captured output, what was attempted and why it did not take, the restored safe position, exactly one owner action), and renders the identical full closeout ceremony a success card renders"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/work_repair_test.go#TestFailedRepairHandbackCarriesAllFourElements"
        status: pass
      - kind: unit
        ref: "cmd/work_repair_test.go#TestHandbackOffersExactlyOneOwnerAction"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 09: One Checkpointed Repair Round Summary

**A failing verification now gets exactly one saved-checkpoint, one-repair-wave, one-re-verification round, with an exact restore on continued failure, an announced save/restore in the flow, and a four-element handback when the repair does not take -- all built on the existing budgeted repair ledger, never a second checkpoint concept.**

## Performance

- **Duration:** 35 min
- **Tasks:** 3
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `runBoundedRepairRound` (`cmd/work_repair.go`) is the one bounded checkpointed repair path: save a checkpoint, run one repair wave, verify once more, and on continued failure restore the checkpoint exactly and return a paused result. It reuses the existing `autopilotRepairLedger`/`autopilotRepairReceipt` types (generalized with a new `CheckpointID` field) rather than a second budget tracker.
- `repairCheckpointIdentity`/`findAutopilotRepairReceiptByCheckpoint` implement SYN-201-10's idempotency-key replay pattern: a checkpoint identity keyed on (phase, check) that already has a receipt is a spent round, refused by name — the same mechanism that also makes a literal replay return the existing receipt.
- `saveRepairCheckpoint`/`restoreRepairCheckpoint`/`repairCheckpointDirectoryDigest` give a scoped directory snapshot/restore, reusing `medic_repair.go`'s own `backupCopyFile`/`backupCopyDir` helpers — proven byte-identical via directory digest comparison.
- `applyBoundedCheckFixRepair` wires the checkpoint/restore round and the D-10 flow announcements (`emitRepairCheckpointSaved`/`emitRepairCheckpointRestored`, rendered through the same visual-mode-gated path `emitContinueVerificationStart` already uses) around the existing D-02 `applyAutomaticCheckFixAttempt`, replacing its call site in the direct continue check lane (`cmd/codex_continue.go`) without altering the fix attempt's own eligibility or dispatch logic.
- `repairHandback`/`buildFailedRepairHandback` assemble D-11's four-element failed-repair handback (diagnosis, what was attempted and why it did not take, the restored position, one owner action) and render it through the identical full closeout ceremony every other work verdict uses (`colony.WorkOutcomeBlocker`, plan 201-04), reusing `recommendedActionForWorkOutcome` (plan 201-07) rather than adding a second recommendation path.

## Task Commits

1. **Task 1: Run one checkpointed repair round and restore on continued failure** - `22604d39` (feat)
2. **Task 2: Announce the checkpoint save and the restore in the flow** - `633f9d62` (feat)
3. **Task 3: Hand back a failed repair with everything the owner needs** - `a04e8e0a` (feat)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `cmd/work_repair.go` - `runBoundedRepairRound`, checkpoint identity/replay, checkpoint save/restore/digest, flow announcements, `applyBoundedCheckFixRepair`, `repairScopePathsForCheckFix`, `repairHandback`/`buildFailedRepairHandback`
- `cmd/work_repair_test.go` - all eight plan-required tests plus fixture helpers
- `cmd/autopilot_policy.go` - `autopilotRepairReceipt.CheckpointID` (the idempotency-key generalization)
- `cmd/codex_continue.go` - direct check lane now calls `applyBoundedCheckFixRepair` instead of `applyAutomaticCheckFixAttempt` directly

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: the checkpoint identity reuses SYN-201-10's idempotency-key REPLAY PATTERN rather than literally embedding `colony.PauseHandoff` (a heavier, differently-shaped primitive built for an owner-invoked whole-colony pause); identity is keyed on (phase, check) so "no second attempt" and "replay returns the existing receipt" are one mechanism; checkpoint scope is the repair's own claimed files, never the whole repository; a checkpoint save failure falls back to the pre-201-09 behavior rather than blocking the existing fix attempt; the handback reuses the existing recommendation mechanism rather than adding a second one.

## Deviations from Plan

### Auto-fixed Issues

None - no bugs, missing-critical-functionality, or blocking issues were found requiring Rule 1-3 auto-fixes.

### Documented Scope Note (not a Rule 1-4 deviation)

**Plan-only/finalize snapshot lane does not yet call the check-fix/repair mechanism at all**
- **Context:** Task 2's acceptance criteria describe announcements appearing "on both lanes" (the direct check lane and the plan-only-plus-finalize lane, i.e. `runCodexContinueVerificationSnapshot`, `cmd/codex_continue_plan.go`, shared by both the plan-only and external finalize continue paths).
- **What was found:** `runCodexContinueVerificationSnapshot` never called `applyAutomaticCheckFixAttempt` (D-02's existing check-fix mechanism) even before this plan — that lane has never run any automatic check-fix attempt, checkpointed or not. Wiring genuine dispatch-capable repair coverage into it would mean either changing that function's signature (it currently has no `colony.ColonyState` or worker-invoker access, both required by `applyAutomaticCheckFixAttempt`) with downstream effects on its other call sites in `cmd/codex_continue_plan.go` and `cmd/codex_continue_finalize.go`, or duplicating the dispatch pipeline — both well outside this plan's declared file scope (`cmd/work_repair.go`, `cmd/autopilot_policy.go`, `cmd/codex_continue.go`) and a materially larger, riskier change than this plan's three tasks describe.
- **What was done instead:** `runBoundedRepairRound` — the one bounded repair path both build and check are meant to use — and its announcement callbacks are lane-agnostic by construction (proven by `TestCheckpointSaveAndRestoreAreAnnouncedOnBothLanes`, which exercises the identical mechanism with two representative caller shapes). Real production wiring landed on the direct check lane, which is the one lane that already ran D-02's check-fix mechanism. The plan-only/finalize snapshot lane's own gap in calling that mechanism at all predates this plan and is unchanged by it.
- **Recommendation for follow-up:** if a future plan (201-10 or later) extends check-fix/repair coverage to the plan-only/finalize snapshot lane, `applyBoundedCheckFixRepair` and `runBoundedRepairRound` are ready to be called from there without further changes to `cmd/work_repair.go`.

---

**Total deviations:** 0 Rule 1-4 auto-fixes. One documented scope note above (a pre-existing, out-of-declared-scope gap that Task 2's acceptance criteria implicitly assumed was already closed). **Impact:** the repair-round mechanism itself is complete, tested, and correctly generalized for both build and check to use; only the plan-only/finalize lane's own (pre-existing, unrelated) lack of any check-fix dispatch limits how far real production wiring could reach within this plan's declared scope.

## Issues Encountered

- `TestGoldenContinueVisualOutput` fails identically on a clean checkout of this plan's parent commit (`e7975f29`, confirmed via `git stash` + re-run) with `got: "Elapsed: 69h..."` vs `want: "Cost: not known..."` — a pre-existing, environment-dependent golden-file mismatch already documented in `201-08-SUMMARY.md`'s own Issues Encountered. Not caused by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `runBoundedRepairRound` is ready for a build-side caller (the plan's own doc comment: "the one bounded repair path build and check both use") — no build-lane wiring was added in this plan (out of declared scope: `cmd/work_repair.go`, `cmd/autopilot_policy.go`, `cmd/codex_continue.go` only).
- `go build ./...` and `go vet ./cmd` are clean. Every plan `<verify>` command passes. Targeted regression sweeps (`CheckFix|Autopilot`, `Continue`, `Closeout`, `TestBuild`) all pass, with the one pre-existing, unrelated golden-file mismatch documented above.
- WORK-06 remains open in `REQUIREMENTS.md` (correctly — shared with 201-10, not yet summarized) and will close once 201-10 finishes too.
- Ready for `201-10-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/work_repair.go` — FOUND
- `cmd/work_repair_test.go` — FOUND
- Commit `22604d39` (Task 1) — FOUND in git log
- Commit `633f9d62` (Task 2) — FOUND in git log
- Commit `a04e8e0a` (Task 3) — FOUND in git log
- `go build ./...` — clean
- `go vet ./cmd` — clean
- All plan `<verify>` commands re-run and passing: `TestRepairRunsAtMostOnceAutomatically`, `TestFailedRepairRestoresTheCheckpointExactly`, `TestRepairReplayReturnsTheExistingReceipt`, `TestRepairNeverOverwritesTheOriginalResult`, `TestCheckpointSaveAndRestoreAreAnnouncedOnBothLanes`, `TestPassedRepairAnnouncesNoRestore`, `TestFailedRepairHandbackCarriesAllFourElements`, `TestHandbackOffersExactlyOneOwnerAction`
