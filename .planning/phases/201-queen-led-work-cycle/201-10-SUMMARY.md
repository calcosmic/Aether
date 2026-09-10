---
phase: 201-queen-led-work-cycle
plan: "10"
subsystem: queen-orchestration
tags: [go, midden, blocker-truth, pheromone-signals, repair, D-12, CAP-003, CAP-004, CAP-024, CAP-051]

# Dependency graph
requires:
  - phase: 201-queen-led-work-cycle (plan 04)
    provides: "pkg/colony/work_outcome.go: colony.WorkOutcome and its equal-ceremony closeout -- outcome vocabulary context this plan's failure/blocker evidence flows into"
  - phase: 201-queen-led-work-cycle (plan 08)
    provides: "cmd/coherent_jobs.go / build_attempt.go: durable per-attempt buildAttemptRecord.Dispatches (Wave/JobName/AttemptID), the identity source this plan reads via loadLatestBuildAttempt rather than threading a new field through codex.WorkerDispatch"
  - phase: 201-queen-led-work-cycle (plan 09)
    provides: "cmd/work_repair.go: runBoundedRepairRound / applyBoundedCheckFixRepair, the one checkpointed repair path this plan extends with flags/recurring-failure inputs and same-run signal delivery"
provides:
  - "cmd/memory_feed.go: recordDispatchWorkerOutcome resolves the phase's own latest durable build attempt to bind a failure record and a worker-reported blocker to the exact attempt and (when grouped) job; MiddenEntry carries that identity via Tags (attempt:/job: prefixes); a worker-reported blocker becomes a durable FlagEntry in pending-decisions.json with Source: escalation and AttemptID set -- the same store advancement/status/closure already read"
  - "pkg/colony/flags.go: FlagEntry.AttemptID, the attempt-binding field for a worker-reported blocker"
  - "cmd/work_repair.go: repairEligibilityEvaluation wraps classifyAutopilotRepairFailure with unresolved blocker flags and a recurring failure class as inputs, naming which one drove the recorded reason; wired into runBoundedRepairRound"
  - "cmd/work_repair.go: failureBornRepairSignal / deliverFailureBornRepairSignal deliver this same-run attempt's own failure evidence into the repair wave's brief as an active REDIRECT pheromone signal -- the existing steering-signal channel both resolveCodexWorkerContext and composeBuildManifestBrief already read -- wired into applyBoundedCheckFixRepair before the repair wave dispatches"
affects: [201-11, 201-12, 201-13, 201-14, 201-15]

# Actuals (#2632)
actuals:
  tokens: 10500
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Identity resolved by re-reading the durable record, never threaded through the dispatch struct: recordDispatchWorkerOutcome calls loadLatestBuildAttempt(dispatch.Phase) to recover the attempt ID and (by matching worker name) the job name, rather than adding an AttemptID field to codex.WorkerDispatch and re-plumbing both build lanes' dispatch-construction call sites -- both lanes already call recordDispatchWorkerOutcome only after commitBuildStart made the attempt durable, so 'the latest build attempt for this phase' IS the attempt this outcome belongs to."
    - "Tags as the extension point, not a signature change: MiddenEntry carries attempt/job identity via Tags (attempt:/job: prefixes) rather than new struct fields plus a new appendMiddenEntry parameter -- appendMiddenEntry's (category, source, message, tags) signature and its other callers (autopilot_retry_record.go, midden_cmds.go) are untouched."
    - "One store, not a new one: a worker-reported blocker becomes a colony.FlagEntry in the exact pending-decisions.json file advancement (checkUnresolvedBlockerFlags), status (readBlockerSnapshot), and closure (LifecycleFacts.Blockers) already read from -- Source: escalation (the same value swarm-strike and checkpoint escalations already write) means the existing single escalated-blocker counting function needs zero changes to count it."
    - "Consume the existing signal, never recount the threshold: recurringFailureClassSignal reads the active REDIRECT signal emitMiddenThresholdRedirect already writes at three unacknowledged failures, rather than re-scanning midden.json and re-deriving the threshold a second time."
    - "Reuse the existing steering channel instead of adding a new brief section or touching check_fix_attempt.go: deliverFailureBornRepairSignal writes an active pheromone REDIRECT signal, which plannedCheckFixBuilderDispatch's ContextCapsule (resolveCodexWorkerContext) and composeBuildManifestBrief's own \"## Pheromone Signals\" section both already surface -- zero changes needed to the real check-fix dispatch construction."

key-files:
  created:
    - cmd/failure_evidence_test.go
  modified:
    - cmd/memory_feed.go
    - cmd/work_repair.go
    - pkg/colony/flags.go

key-decisions:
  - "Attempt/job identity is resolved by loadLatestBuildAttempt(dispatch.Phase) inside recordDispatchWorkerOutcome rather than threading a new AttemptID field through codex.WorkerDispatch and both build lanes' dispatch-construction call sites (buildCodexWorkerDispatches in cmd/codex_build.go, the external-lane construction in cmd/codex_build_finalize.go). This kept the change confined to Task 1's declared file (cmd/memory_feed.go) and avoided touching two widely-exercised dispatch-construction paths for identity that was already durably recorded one call away."
  - "A worker-reported blocker's durable record is a colony.FlagEntry (Type: blocker, Source: escalation, AttemptID set) appended into the SAME pending-decisions.json file every advancement/status/closure reader already resolves blockers from -- not a new attempt-bound store. This makes CAP-004's 'visible to advancement, status and closure reads' true by construction rather than by wiring three separate readers."
  - "The same-run failure-born signal (D-12) is delivered as a genuine active pheromone REDIRECT signal (writePheromoneSignal) rather than by extending composeBuildManifestBrief's parameter list and wiring a new parameter through cmd/check_fix_attempt.go's plannedCheckFixBuilderDispatch. plannedCheckFixBuilderDispatch already sets its worker's ContextCapsule to resolveCodexWorkerContext(), which already surfaces active pheromone signals -- so writing the signal reaches the real production check-fix dispatch with zero changes to check_fix_attempt.go, and composeBuildManifestBrief's own Pheromone Signals section (used by the build lane) picks up the identical signal for free."
  - "repairEligibilityEvaluation enriches classifyAutopilotRepairFailure's Reason string rather than replacing its Eligible/Pause decision -- there is still exactly one eligibility decision; the two new inputs (unresolved flags, a recurring failure class) only make the recorded reason name which input drove it, satisfying CAP-024 without introducing a second decision path."
  - "REQUIREMENTS.md's WORK-06 checkbox was flipped by hand after `gsd-tools requirements mark-complete WORK-06` reported not_found -- the same pre-existing tool/format mismatch documented in 201-08-SUMMARY.md (the checkbox regex expects `**REQ-ID**` closed immediately after the ID; this file's actual style is `**REQ-ID — Title:**`). Confirmed safe to flip by hand only after `requirements ready-ids` independently reported WORK-06 unblocked (both plans declaring it, 201-09 and this one, now have summaries). WORK-05 remains open -- still blocked by sibling plans 201-06/07/15."

requirements-completed: [WORK-06]
# WORK-05 (this plan's other declared requirement) is also declared by
# sibling plans 201-06, 201-07, and 201-15 per requirements.ready-ids' own
# scan, none of which have a SUMMARY.md yet -- it stays open per the
# shared-ID gate (#2388) until every declaring plan finishes, expected and
# correct, not a gap in this plan's own work.

coverage:
  - id: D1
    description: "A failed worker on either build lane writes one failure record through the single memory-feed boundary, carrying the attempt identifier and job name -- and a worker-reported blocker becomes a durable, attempt-bound record resolved identically by advancement, status, and closure reads, with the escalated-blocker count computed by exactly one existing function"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestFailureEvidenceCarriesTheAttemptIdentity"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestBlockerTruthIsOneStore"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestEscalatedCountHasOneCountingPath"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestEvidenceStorageFailureNeverFailsTheRun"
        status: pass
      - kind: unit
        ref: "cmd/memory_feed_test.go#TestEveryBuildLaneFeedsMemoryThroughOneBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Bounded recovery's eligibility evaluation reads unresolved flags and a recurring failure class alongside the failing check, names which input drove its recorded reason, never reports a phase with unresolved flags as having nothing to repair, and there is exactly one recovery-ledger entry point in the package"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestRepairEvaluationNamesItsDrivingInput"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestUnresolvedFlagsNeverEvaluateAsNothingToRepair"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestOneRecoveryModelConsumesFlagsAndFailures"
        status: pass
    human_judgment: false
  - id: D3
    description: "This attempt's own failure-born signal reaches the repair wave's worker brief (both resolveCodexWorkerContext and composeBuildManifestBrief's Pheromone Signals section) in the same run, naming the specific failure, staying within its content budget with a non-silent trim, through the existing steering-signal section only -- and is delivered after the failure record is written and before the repair dispatch"
    requirement: WORK-06
    verification:
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestFailureBornSignalReachesTheRepairBrief"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestRepairBriefRespectsItsContentBudget"
        status: pass
      - kind: unit
        ref: "cmd/failure_evidence_test.go#TestSignalDeliveredBeforeRepairDispatch"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-10
status: complete
---

# Phase 201 Plan 10: Failure Evidence, Blocker Truth, and Same-Run Repair Signals Summary

**Worker failures and blockers now carry the exact build attempt and job identity that produced them, a worker-reported blocker is a durable record in the same store every advancement/status/closure surface already reads, and a failing attempt's own failure now reaches the repair wave's brief as an active pheromone signal in the same run -- with no second memory-feed path, no second blocker store, no second counting function, and no second recovery model.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-10T15:57:00Z (approx., per STATE.md at session start)
- **Completed:** 2026-09-10T16:26:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `recordDispatchWorkerOutcome` (`cmd/memory_feed.go`) now resolves the phase's own latest durable build attempt (`loadLatestBuildAttempt`) to bind a failed worker's midden entry and a worker-reported blocker's flag entry to the exact attempt ID and (by matching worker name against the attempt's own recorded dispatches) job name -- no new field on `codex.WorkerDispatch`, no change to either build lane's dispatch-construction code.
- `colony.MiddenEntry` carries that identity via `Tags` (`attempt:`/`job:` prefixes, read back by `middenEntryAttemptID`/`middenEntryJobName`) -- the one existing extension point `appendMiddenEntry` already exposed, so its long-standing four-argument signature and its other callers are unchanged.
- A worker-reported blocker becomes a durable `colony.FlagEntry` (new `AttemptID` field, `Type: blocker`, `Source: escalation`) appended into `pending-decisions.json` -- the exact file `checkUnresolvedBlockerFlags` (advancement), `readBlockerSnapshot` (status), and `LifecycleFacts.Blockers` (closure) already read from, so all three surfaces see it without any change to those readers. `Source: escalation` means the existing single escalated-blocker counting function (`readBlockerSnapshotEvidence`) counts it with zero changes (CAP-051).
- `repairEligibilityEvaluation` (`cmd/work_repair.go`) wraps `classifyAutopilotRepairFailure` with two further inputs -- unresolved blocker flags and a recurring failure class (consumed from the existing REDIRECT signal `emitMiddenThresholdRedirect` already writes at the three-unacknowledged-failures threshold, never recounted) -- so the recorded reason names which of the three inputs drove the decision, and a phase with unresolved flags is never reported as having nothing to repair. Wired into `runBoundedRepairRound`, the one repair-ledger entry point (proven from the parsed syntax tree).
- `deliverFailureBornRepairSignal` (`cmd/work_repair.go`) reads this same-run attempt's own failure evidence (`failureBornRepairSignal`, reading Task 1's midden tag) and writes it as an active REDIRECT pheromone signal -- the existing steering-signal channel both `resolveCodexWorkerContext` (the real check-fix repair worker's `ContextCapsule`) and `composeBuildManifestBrief`'s own "## Pheromone Signals" section already surface, so no new brief section and no change to `cmd/check_fix_attempt.go` were needed. Content is truncated (never silently dropped) to `signalContentSafeLimit`, matching the established budget discipline. Wired into `applyBoundedCheckFixRepair` right after the checkpoint is saved and before the repair wave dispatches.

## Task Commits

1. **Task 1: Bind failure evidence and blocker truth to the exact attempt** - `bc8b2d56` (feat)
2. **Task 2: Let bounded recovery consume flags and recurring failure classes** - `9873a53d` (feat)
3. **Task 3: Deliver this phase's failure-born signals into the repair brief** - `a8ee899f` (feat)

**Plan metadata:** committed alongside this summary.

## Files Created/Modified

- `pkg/colony/flags.go` - `FlagEntry.AttemptID`, binding a blocker to the exact build attempt that produced it
- `cmd/memory_feed.go` - `workerOutcomeFacts.AttemptID`/`.JobName`, attempt/job resolution in `recordDispatchWorkerOutcome`, `sanitizedWorkerSentence` (consolidated sanitisation), `middenTagsForFacts`/`middenEntryAttemptID`/`middenEntryJobName`, `recordWorkerBlockerFlag`
- `cmd/work_repair.go` - `repairEligibilityEvaluation`, `recurringFailureClassSignal`, `failureBornRepairSignal`, `deliverFailureBornRepairSignal`, wiring into `runBoundedRepairRound` and `applyBoundedCheckFixRepair`
- `cmd/failure_evidence_test.go` (new) - all eleven plan-required tests plus fixture helpers (`seedMinimalBuildAttempt`, `seedUnresolvedBlockerFlag`, `seedRecurringFailureRedirectSignal`, `seedFailureBornMiddenEntry`)

## Decisions Made

See `key-decisions` in the frontmatter for full reasoning; in short: attempt/job identity is re-read from the durable build attempt record rather than threaded through the dispatch struct; a worker-reported blocker is a `FlagEntry` in the existing `pending-decisions.json` store, never a new one; the same-run repair signal rides the existing pheromone-signal channel (reaching the real check-fix dispatch through `resolveCodexWorkerContext` with zero changes to `check_fix_attempt.go`) rather than a new brief parameter; `repairEligibilityEvaluation` enriches the existing decision's reason rather than adding a second decision path; and `REQUIREMENTS.md`'s `WORK-06` checkbox was hand-flipped after root-causing the same bold-label styling gap `gsd-tools` hit on `WORK-01` in 201-08.

## Deviations from Plan

None - plan executed exactly as written. The identity-resolution and signal-delivery design choices above were implementation decisions made within each task's own declared file scope to satisfy the plan's stated acceptance criteria (e.g., "never add a second path" / "no new brief section") -- not deviations from what the plan specified.

## Issues Encountered

- `TestGoldenContinueVisualOutput` fails identically on a clean checkout of this plan's parent commit (confirmed via a stash-and-rerun), with the same pre-existing, environment-dependent golden-file mismatch documented in both `201-08-SUMMARY.md` and `201-09-SUMMARY.md`'s own Issues Encountered sections. Not caused by this plan.
- `TestPhase199GateReceipt` also fails identically on a clean checkout (a pre-existing repo-state-sentinel check against the working tree's `.planning/config.json`/`.gsd` baseline, unrelated to this plan's own files). Not caused by this plan.
- `gsd-tools requirements mark-complete WORK-06` reported `not_found` -- the same bold-label styling gap (`**REQ-ID — Title:**` vs. the tool's expected `**REQ-ID**`) documented as a tooling/format mismatch in `201-08-SUMMARY.md`. Flipped the checkbox by hand after confirming via `requirements ready-ids` that WORK-06 was genuinely unblocked.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `middenEntryAttemptID`/`middenEntryJobName` and `recordWorkerBlockerFlag`'s `AttemptID`-bound `FlagEntry` are the one place a later plan should read/write attempt-bound failure and blocker identity -- never a second store or a second tag scheme.
- `repairEligibilityEvaluation` is wired into `runBoundedRepairRound`; a later plan extending bounded recovery with further inputs should extend this one function's reason-enrichment rather than adding a parallel evaluation.
- `deliverFailureBornRepairSignal` is production-wired into the direct check-fix lane (`applyBoundedCheckFixRepair`) only, matching this phase's own repeated precedent (201-08, 201-09) of real wiring reaching the lane that already ran the underlying mechanism; the build-lane repair path (`runBoundedRepairRound` called from a build-side caller) remains unwired to any check-fix-style mechanism, as `201-09-SUMMARY.md` already documented -- `failureBornRepairSignal`/`deliverFailureBornRepairSignal` are ready to be called from there without further changes to `cmd/work_repair.go`.
- `go build ./...` and `go vet ./cmd ./pkg/colony` are clean. Every task-level `<verify>` command from `201-10-PLAN.md` passes. Targeted regression sweeps (`WorkRepair|Repair|Blocker|MemoryFeed|Flag|Gate|Midden|Checkpoint`; `CheckFix|Continue|TestBuild$|BuildFinalize|LifecycleClose|LifecycleFacts|LifecycleProjection|Autopilot`) both pass, with the two pre-existing, unrelated failures documented above.
- WORK-06 is now complete. WORK-05 remains open -- blocked by sibling plans 201-06/07/15, expected per the shared-ID gate.
- Ready for `201-11-PLAN.md`.

---
*Phase: 201-queen-led-work-cycle*
*Completed: 2026-09-10*

## Self-Check: PASSED

- `cmd/failure_evidence_test.go` — FOUND
- `pkg/colony/flags.go`, `cmd/memory_feed.go`, `cmd/work_repair.go` — all modified, present
- Commit `bc8b2d56` (Task 1) — FOUND in git log
- Commit `9873a53d` (Task 2) — FOUND in git log
- Commit `a8ee899f` (Task 3) — FOUND in git log
- `go build ./...` — clean
- `go vet ./cmd` — clean
- All plan `<verify>` commands re-run and passing: `TestFailureEvidenceCarriesTheAttemptIdentity`, `TestBlockerTruthIsOneStore`, `TestEscalatedCountHasOneCountingPath`, `TestEvidenceStorageFailureNeverFailsTheRun`, `TestEveryBuildLaneFeedsMemoryThroughOneBoundary`, `TestRepairEvaluationNamesItsDrivingInput`, `TestOneRecoveryModelConsumesFlagsAndFailures`, `TestFailureBornSignalReachesTheRepairBrief`, `TestRepairBriefRespectsItsContentBudget`
