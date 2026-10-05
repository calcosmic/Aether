---
phase: 208-never-a-dead-end
plan: 04
subsystem: cli
tags: [go, cobra, continue, recovery, status, planning-authority]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal) -- this plan's own screen line follows the same plain-English discipline, though it does not itself go through refuse()."
  - phase: 208-02
    provides: "Per-task criterion verdicts and cd-prefixed verification commands in cmd/codex_continue.go -- this plan's read_first line ranges assumed that code, not the plan's stale memory of it."
provides:
  - "A blocked `aether continue` writes the unfinished work back onto the phase as pending recovery tasks, idempotently across repeat blocked checks"
  - "continueNextCommandForBlocked never returns the empty string -- the D-08 dead end is closed"
  - "resolvePhaseProgressFromDisk: one ordered five-rank scale (not started / in progress / built but unverified / verified / complete) that believes the LESS finished of the stored phase and the last check's own durable record"
  - "aether status shows the less-finished rank and says so on screen when the two disagree"
affects: [208-06, 208-07, 208-08]

# Actuals (#2632)
actuals:
  tokens: 13868
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Recovery-task identity via a deterministic, plain-English Goal prefix rather than colony.Task.SemanticID -- SemanticID is part of the specification/candidate planning-authority system and a non-empty value there breaks any legacy_unbound plan outright (a real bug this plan's own idempotency test caught before it could ship)"
    - "One package-level ordered rank scale (phaseProgressRankOrder), read by two independent rankers (stored phase, durable check record), with a structural AST test refusing a second such scale anywhere in cmd/*.go"

key-files:
  created:
    - cmd/failed_check_carries_on_test.go
    - cmd/phase_progress_from_disk.go
    - cmd/phase_progress_from_disk_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/codex_visuals.go
    - cmd/status.go
    - cmd/codex_continue_test.go

key-decisions:
  - "Recovery tasks never set colony.Task.SemanticID. assessCodexContinue's Recovery.ReconcileTasks and Recovery.RedispatchTasks currently overlap completely (both are populated from the identical `RecoveryAction == \"redispatch\"` classification -- there is no distinct 'reconcile' outcome today), so recoveryTasksForBlockedContinue de-duplicates a task ID across both lists, reconcile winning first (the lighter recovery, matching continueNextCommandForBlocked's own priority)."
  - "Cross-run idempotency and 'don't re-recover a recovery task' both key off a deterministic Goal prefix (\"Finish task <id> (<verb>)\"), not a dedicated field -- appendRecoveryTasks matches new candidates against existing tasks' Goal prefixes, and recoveryTasksForBlockedContinue refuses to turn a task whose OWN Goal already has that shape into a fresh candidate (otherwise a recovery task, having no dispatch evidence of its own, would be re-classified as needing recovery on the very next blocked check, without limit)."
  - "resolvePhaseProgressFromDisk is wired only into renderDashboard (the function `aether status` actually renders through, per WINDOWS.md rows 22/23), not into buildStatusResult's JSON map -- matching the plan's explicit scope; the JSON envelope's tasks_completed/tasks_total are unchanged by this plan."
  - "The sealed/completed special case (a COMPLETED colony should not show a stale incomplete task count) is checked BEFORE resolvePhaseProgressFromDisk and short-circuits it, so a sealed colony's dashboard is byte-for-byte unchanged by this plan."

requirements-completed: [UED-12]

coverage:
  - id: D1
    description: "A blocked check with N tasks needing recovery writes N pending tasks back onto the phase (once, not twice on repeat), never advances or verifies the phase, and continueNextCommandForBlocked never returns the empty string"
    requirement: "UED-12"
    verification:
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestFailedCheckAddsTheUnfinishedWorkAsTasks"
        status: pass
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestFailedCheckAddsTheSameTasksOnlyOnce"
        status: pass
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestFailedCheckNeverAdvancesOrVerifies"
        status: pass
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestBlockedCheckAlwaysNamesANextCommand"
        status: pass
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestRecoveryTasksCarryNoEvidenceRequirements"
        status: pass
      - kind: unit
        ref: "cmd/failed_check_carries_on_test.go#TestRecoveryTasksNeverSetSemanticID"
        status: pass
    human_judgment: false
  - id: D2
    description: "aether status resolves each phase's progress from the stored phase and the durable check record and shows the less-finished of the two, saying so on screen, when they disagree"
    requirement: "UED-12"
    verification:
      - kind: unit
        ref: "cmd/phase_progress_from_disk_test.go#TestLessFinishedRecordIsBelieved"
        status: pass
      - kind: unit
        ref: "cmd/phase_progress_from_disk_test.go#TestProgressFromDiskAgreesWhenRecordsAgree"
        status: pass
      - kind: unit
        ref: "cmd/phase_progress_from_disk_test.go#TestResolvePhaseProgressFromDiskWritesNothing"
        status: pass
      - kind: unit
        ref: "cmd/phase_progress_from_disk_test.go#TestPhaseProgressRankOrderHasOneSource"
        status: pass
    human_judgment: false

duration: 95min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 04: Never a Dead End -- Recovery Tasks and Believed Progress Summary

**A failed check no longer ends with nothing to run next: the unfinished work is written back onto the phase as tasks (once, never twice on a repeat check), and `aether status` now believes whichever of two on-disk records -- the saved phase or the last check's own record -- says the work is LESS finished.**

For the owner, in plain English: before this, when a check on your project failed, Aether would sometimes just stop talking -- the screen showed no command to type next. Now it writes down exactly what's still unfinished as a short new to-do list on that phase, and always tells you the one thing to run. It never pretends the phase is done just because it wrote that list. Separately, the main status screen used to trust whatever was last saved about a phase, even if a more recent check had actually found less was really finished (or the reverse). Now it compares the two records and always shows you the more cautious, less-finished answer -- and tells you plainly when they disagreed, instead of picking one silently.

## Performance

- **Duration:** 95 min
- **Started:** 2026-09-22T17:53:00Z (approx, first file read)
- **Completed:** 2026-09-22T19:28:00Z
- **Tasks:** 2
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments

- `recoveryTasksForBlockedContinue` / `appendRecoveryTasks` / `recordBlockedContinueWorkerFlow` (`cmd/codex_continue.go`): a blocked `aether continue` now writes the unfinished work (from `assessment.Recovery.ReconcileTasks`/`RedispatchTasks`) back onto the phase as pending tasks, inside the SAME atomic state write that already records the blocked event -- idempotent across repeated blocked checks, and a recovery task never itself becomes a candidate for further recovery (which would otherwise grow without limit).
- `continueNextCommandForBlocked` (`cmd/codex_continue.go`): the branch that used to return the empty string (D-08's "don't loop back to `aether continue`") now names the reconcile command, else the targeted redispatch command, else `aether status` -- never empty. The blocked result map gains `recovery_tasks_added`, and `renderContinueBlockedVisual` (`cmd/codex_visuals.go`) names the count in plain English.
- `phaseProgressRankOrder` / `resolvePhaseProgressFromDisk` (new `cmd/phase_progress_from_disk.go`): one ordered five-rank scale (not started / in progress / built but unverified / verified / complete), read by two independent rankers -- the stored phase's own `Status` + `Tasks`, and the durable check record at `build/phase-<id>/continue.json` -- always returning the LESS finished of the two when they disagree.
- `cmd/status.go`'s `renderDashboard` (the function `aether status` actually renders through) now calls `resolvePhaseProgressFromDisk` and, on disagreement, overrides the shown task counts/status and adds one plain-English line naming both records.

## Task Commits

Each task was committed atomically:

1. **Task 1: A failed check writes the unfinished work back as tasks and always names the way forward** - `718f7bb3` (feat)
2. **Task 2: Work the status out from what is on disk, and believe the less-finished record** - `70a24c4e` (feat) -- includes the necessary update to a pre-existing test (`TestContinueBlocksWhenContinueWatcherRejectsPhase`) that asserted Task 1's now-removed empty-string dead end.

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/failed_check_carries_on_test.go` - Task 1's tests: end-to-end blocked-path proofs (drives real `aether continue` via the cobra command), the `continueNextCommandForBlocked` branch table, the no-evidence-requirements proof, and the `SemanticID` regression guard
- `cmd/phase_progress_from_disk.go` - `phaseProgressRankOrder`, `phaseProgressFromDisk`, `resolvePhaseProgressFromDisk`, and the two per-record rankers
- `cmd/phase_progress_from_disk_test.go` - Task 2's tests, including the AST structural guard for a second ranking scale
- `cmd/codex_continue.go` - `recoveryTaskSemanticID`, `recoveryTaskVerb`, `recoveryTaskGoalPrefix`, `recoveryTaskGoalGeneratedPrefix`, `taskGoalLooksLikeRecoveryTask`, `recoveryTasksForBlockedContinue`, `appendRecoveryTasks`; `recordBlockedContinueWorkerFlow` signature extended (`phaseID`, `assessment`, returns the added count); both blocked-branch call sites updated; `continueNextCommandForBlocked`'s dead-end branch rewritten
- `cmd/codex_visuals.go` - `renderContinueBlockedVisual` names `recovery_tasks_added`
- `cmd/status.go` - `renderDashboard`'s task-progress block calls `resolvePhaseProgressFromDisk` and renders the disagreement line
- `cmd/codex_continue_test.go` - `TestContinueBlocksWhenContinueWatcherRejectsPhase` updated for the new never-empty guarantee

## Decisions Made

See `key-decisions` in the frontmatter above. In short: recovery-task identity lives in the Goal text, never `SemanticID` (a real collision this plan's own idempotency test caught -- see Deviations); the two overlapping recovery-task lists are deduplicated reconcile-first; and `resolvePhaseProgressFromDisk` is wired only into the visual renderer, matching the plan's declared scope.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `colony.Task.SemanticID` is not a safe field for recovery-task identity**

- **Found during:** Task 1, while writing `TestFailedCheckAddsTheSameTasksOnlyOnce` (running the same blocked check twice in one test)
- **Issue:** The plan's action text called for setting a new recovery task's `SemanticID` to a derived identity string, matching on it later for idempotency. Doing so broke `aether continue` outright on the second call in the test, with `"legacy_unbound plan cannot contain current candidate or acceptance bindings"`. Root cause: `colony.Task.SemanticID` belongs to the specification/candidate planning-authority system (`pkg/colony`'s `planHasCurrentBindings`, `cmd/planning_state.go`'s `validatePlanningState`) -- ANY task carrying a non-empty `SemanticID` trips that refusal once `acceptance_policy` is `legacy_unbound`, the common historical shape. This is a real, previously-latent collision this plan's own test surfaced, not a hypothetical.
- **Fix:** Recovery tasks never set `SemanticID`. Identity and cross-run idempotency instead read a deterministic, plain-English Goal prefix (`recoveryTaskGoalPrefix`/`recoveryTaskGoalGeneratedPrefix`), which is also what stops a recovery task from being re-classified as needing recovery on the next blocked check.
- **Files modified:** cmd/codex_continue.go, cmd/failed_check_carries_on_test.go (added `TestRecoveryTasksNeverSetSemanticID` as the regression guard)
- **Verification:** `TestFailedCheckAddsTheSameTasksOnlyOnce` and `TestRecoveryTasksNeverSetSemanticID` pass; the latter directly reproduces the exact `legacy_unbound` scenario and asserts `validatePlanningState` accepts a phase carrying recovery tasks.
- **Committed in:** 718f7bb3

**2. [Rule 1 - Bug] `TestContinueBlocksWhenContinueWatcherRejectsPhase` asserted the exact behavior this plan removes**

- **Found during:** Task 2's full-suite verification pass
- **Issue:** This pre-existing test explicitly asserted `next == ""` ("want empty guidance so blocked watcher output does not suggest an identical retry") -- the literal D-08 dead end 208-04 exists to close.
- **Fix:** Updated the assertion to require a non-empty `next` that is not an identical `aether continue` retry (the concern the original test's comment actually cared about).
- **Files modified:** cmd/codex_continue_test.go
- **Verification:** `TestContinueBlocksWhenContinueWatcherRejectsPhase` passes.
- **Committed in:** 70a24c4e

---

**Total deviations:** 2 auto-fixed (1 blocking correctness bug caught by this plan's own test, 1 necessary update to a test asserting the removed behavior). **Impact on plan:** Neither changed architecture or scope; both were required for Task 1's change to be correct and consistent.

## Issues Encountered

**Pre-existing, unrelated known-red confirmed during verification** (each independently re-verified via `git stash` against the commit immediately before this plan's work, with this plan's own new test files moved aside so the stash comparison is exact):

- `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestNoSecondAutomaticFixAttempt`, `TestFixAttemptIsCountedSeparately` (`cmd/floor_fix_attempt_test.go`) -- fail identically on the pre-existing commit; matches the known-red list 208-02-SUMMARY.md already recorded for this same file.
- `TestContinueCreditsTasksProvenInAnEarlierAttempt` (`cmd/continue_attempt_credit_test.go`) -- fails identically pre-existing; unrelated to any file this plan touches.
- `TestStatusNamesAnArchivedProject` (`cmd/status_test.go`) -- fails identically pre-existing, at the `aether seal` step with `"revisions[0].plan_hash does not match its phase snapshot"`, entirely before this plan's `renderDashboard` change is ever reached.

None of these touch a file this plan modified.

**`gsd-tools windows append` unavailable in this environment.** The installed `gsd-tools` binary here is a different (`gsd-sdk`) build whose command list has no `windows` subcommand, so the cross-phase defect ledger could not be populated programmatically. Per the ledger's own best-effort contract, this is recorded here instead rather than risking a hand-edited corruption of `.planning/WINDOWS.md`'s dual table/JSON representation: **the wrapper-mediated `continue-finalize` lane's blocked path** (`cmd/codex_continue_finalize.go`, `advanceExternalContinue`'s sibling blocked branch, around the comment referencing `recordBlockedContinueWorkerFlow`) writes `COLONY_STATE.json` through its own separate `UpdateJSONAtomically` block and does **not** call `recordBlockedContinueWorkerFlow` -- so a blocked check on that lane does not yet write recovery tasks back onto the phase the way the default `aether continue` lane (this plan's Task 1) now does. Out of this plan's declared `files_modified` (`cmd/codex_continue.go`, `cmd/phase_progress_from_disk.go`, `cmd/status.go` only) -- a future plan should route that sibling blocked path through the same mechanism.

## Known Stubs

- **`continue-finalize`'s blocked path does not yet write recovery tasks** (see above) -- the "never a dead end" guarantee this plan builds currently covers the default `aether continue` lane only, not the wrapper-mediated finalize lane. Tracked above in lieu of a WINDOWS.md entry (tool unavailable this session).

## Mutation Proofs

- **Task 1 (D-08 fallback):** with `continueNextCommandForBlocked`'s dead-end branch temporarily reverted to `return ""`, `TestBlockedCheckAlwaysNamesANextCommand` fails on all three "previously-empty fall-through" sub-cases with `"branch ... returned an empty next command"`. Reverted; test passes again.
- **Task 2 (believe the less-finished record):** with `resolvePhaseProgressFromDisk`'s disagreement branch temporarily forced to always keep the stored (more-finished, in the mutated case) record, `TestLessFinishedRecordIsBelieved`'s "stored says complete, check says nothing verified" sub-case fails with `Status = "complete", want "not started"`. Reverted; test passes again.
- **Task 2 (one ranking rule):** a second hand-rolled `[]string{"not started", "in progress", "built but unverified"}` literal assigned to a differently-named variable, temporarily appended to `cmd/phase_progress_from_disk.go`, makes `TestPhaseProgressRankOrderHasOneSource` fail by name (`"found a second ordered progress-rank scale assigned to \"mutationTestSecondRankScale\""`). Reverted; test passes again.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plans 208-06, 208-07, and 208-08 (per this plan's `affects` list) can rely on the recovery-task mechanism and the believed-progress resolver without re-deriving either. The one open item is the `continue-finalize` lane gap documented above under Known Stubs -- not a gate on this plan or on 208-06/07/08, but worth a line item if a later plan touches that file.

No blockers.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- All 3 created files verified present on disk (`[ -f ]`): `cmd/failed_check_carries_on_test.go`, `cmd/phase_progress_from_disk.go`, `cmd/phase_progress_from_disk_test.go`.
- Both task commit hashes (718f7bb3, 70a24c4e) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: all named tests in the Coverage block above pass.
- Re-ran plan-level `<verification>`: `go build ./cmd/aether` and `go vet ./cmd` clean; `go test ./cmd -run 'TestFailedCheck|TestBlockedCheckAlwaysNamesANextCommand|TestRecoveryTasks|TestLessFinishedRecordIsBelieved|TestProgressFromDisk|TestStatus|TestCriterionEvidence|TestContinue' -count=1 -timeout 8m` passes except the three confirmed pre-existing known-red failures documented above under Issues Encountered.
