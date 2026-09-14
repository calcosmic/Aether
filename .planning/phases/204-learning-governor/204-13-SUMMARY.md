---
phase: 204-learning-governor
plan: 13
subsystem: learning-governor
tags: [episode-ledger, live-events, swarm, recovery, intervention-vocabulary, go]

# Dependency graph
requires:
  - phase: 204-04
    provides: episode_ledger.go (recordEpisodeOutcome, the closed episodeLedgerRecordKind vocabulary shape) and live_events.go's episode-boundary helpers
  - phase: 204-10
    provides: improvement_report.go's two-figure report and collectPreventableInterventions
provides:
  - All six lifecycle lanes (build, continue, plan, oracle, swarm, recovery) open and close a durable episode record through emitColonyLiveEpisodeStarted/Ended
  - A closed, source-derived episodeInterventionKind vocabulary with three real production writers
  - collectPreventableInterventions classifies against the declared vocabulary instead of free text
affects: [204-VERIFICATION.md SC3b, half of SC3a, WINDOWS entry 45's owner-intervention half]

# Actuals (#2632)
actuals:
  tokens: 14260
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Episode boundary pattern reused verbatim from cmd/codex_build.go's runCodexBuildWithOptions three-line shape (emitColonyLiveEpisodeStarted, deferred emitColonyLiveEpisodeEnded reading a runStatus variable) for the swarm lane"
    - "Closed-vocabulary + AST-derived completeness test, mirroring episodeLedgerRecordKind's own shape, applied to a new episodeInterventionKind type"

key-files:
  created:
    - cmd/intervention_vocabulary_test.go
  modified:
    - cmd/swarm_cmd.go
    - cmd/recovery_orchestrator.go
    - cmd/live_events.go
    - cmd/episode_ledger.go
    - cmd/improvement_report.go
    - cmd/handoff_decisions_cmd.go
    - cmd/forced_reviewer_waiver.go
    - cmd/rollback.go
    - cmd/episode_ledger_test.go
    - cmd/live_lane_coverage_test.go

key-decisions:
  - "Task 1 (swarm driver route): drove runSwarmDestroy end to end from driveSwarmLiveLane rather than extracting a production helper. newSwarmWorkerInvoker is already a package-level swap seam (used by TestSwarmDestroyRunsWorkerWavesAndReturnsStructuredResult) and runSwarmDestroy's own preflight (swarmInterventionPreflight -> loadLifecycleFacts) degrades gracefully with no colony state, so the real entry point could be called directly without new production surface."
  - "Task 3(c) waiver writer site: emitColonyLiveInterventionRecorded is called inside resolveForcedReviewerWaiverPendingDecision, immediately after its UpdateJSONAtomically transaction reports found=true. This is the single site that fires exactly once per real, genuine waiver -- forcedReviewerWaiver (the read-only checker) and ensureForcedReviewerWaiverPendingDecision (which only creates the pending row, before any decision is made) were both rejected as writer sites."
  - "Intervention-episode resolution reuses currentLiveRecoveryEpisode(phase) verbatim at all three writer call sites (build episode, else check episode, else a phase-derived fallback), rather than inventing a second, parallel resolver. releaseCanaryQuarantine has no phase parameter, so it calls currentLiveRecoveryEpisode(0), producing the fallback id \"recovery-phase-0\" for a standalone owner action."
  - "collectPreventableInterventions now filters to declared kinds only; an episode whose only intervention is an undeclared category falls through to buildImprovementReport's EXISTING UnclassifiedEpisodes list (no new struct field), keeping TestTwoFiguresAreNeverCombined's closed-field assertion green. A new collectUnrecognizedInterventionCategories helper names the excluded category by episode and text, tested directly."

requirements-completed: [LEARN-02]

coverage:
  - id: D1
    description: "The swarm lane opens and closes a durable episode record in production, proven by driving the real runSwarmDestroy entry point, including on its investigation-wave failure path."
    requirement: "LEARN-02"
    verification:
      - kind: unit
        ref: "cmd/episode_ledger_test.go#TestSwarmLaneOpensAndClosesADurableEpisode"
        status: pass
      - kind: unit
        ref: "cmd/episode_ledger_test.go#TestEveryLifecycleLaneWritesADurableOutcome/swarm"
        status: pass
    human_judgment: false
  - id: D2
    description: "The recovery lane opens a durable episode only in the no-open-episode fallback case; the recovery-stays-in-its-run guarantees are unbroken; the t.Skipf branch in TestEveryLifecycleLaneWritesADurableOutcome is now a named t.Fatalf."
    requirement: "LEARN-02"
    verification:
      - kind: unit
        ref: "cmd/episode_ledger_test.go#TestRecoveryLaneOpensADurableEpisodeOnlyWhenItOwnsOne"
        status: pass
      - kind: unit
        ref: "cmd/episode_ledger_test.go#TestEveryLifecycleLaneWritesADurableOutcome"
        status: pass
      - kind: unit
        ref: "cmd/live_recovery_episode_test.go#TestRecoveryDecisionKeepsTheBuildEpisodeLive"
        status: pass
    human_judgment: false
  - id: D3
    description: "A closed episodeInterventionKind vocabulary (answered a worker's question, declined a forced reviewer, released a quarantined candidate), validated at emitColonyLiveInterventionRecorded's one write point, with a real, mutation-proved production writer for each declared member."
    requirement: "LEARN-02"
    verification:
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestInterventionKindVocabularyIsClosed"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestUndeclaredInterventionKindIsRefused"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestEveryInterventionKindHasALiveWriter"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestOwnerAnswerWritesOneInterventionRecord"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestDeclinedReviewerWritesOneInterventionRecord"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestQuarantineReleaseWritesOneInterventionRecord"
        status: pass
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestIdenticalInterventionsCollapseOnlyAtTheSameInstant"
        status: pass
    human_judgment: false
  - id: D4
    description: "collectPreventableInterventions classifies against the declared vocabulary; an undeclared category is excluded from the figure and surfaced by name rather than silently counted, without adding a field to the closed improvementReport struct."
    requirement: "LEARN-02"
    verification:
      - kind: unit
        ref: "cmd/intervention_vocabulary_test.go#TestUnrecognizedInterventionCategoryIsExcludedAndReportedByName"
        status: pass
      - kind: unit
        ref: "cmd/improvement_report_test.go#TestTwoFiguresAreNeverCombined"
        status: pass
    human_judgment: false

# Metrics
duration: 95min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 13: Swarm and Recovery Episode Boundaries, Closed Intervention Vocabulary Summary

**Wired the swarm and recovery lanes onto the durable episode boundary (closing SC3b) and declared a closed, source-derived intervention-kind vocabulary with three real production writers (closing the owner-intervention half of SC3a and D-10).**

## Performance

- **Duration:** 95 min
- **Started:** 2026-09-14T20:10:00Z
- **Completed:** 2026-09-14T21:45:00Z
- **Tasks:** 3
- **Files modified:** 11 (8 production, 2 existing tests, 1 new test file)

## Accomplishments
- `runSwarmDestroy` now opens `emitColonyLiveEpisodeStarted(swarmID, events.EpisodeKindSwarm)` right after `initializeSwarmRun` succeeds and closes it via a deferred `emitColonyLiveEpisodeEnded` reading the same `runStatus` variable `finishRuntimeSpawnRun` already reads — the episode closes on every return path, including the investigation-wave error/timeout paths.
- `orchestrateRecovery` opens and closes a durable episode of its own ONLY when `currentLiveRecoveryEpisode` returns the no-open-episode fallback (`events.EpisodeKindRecovery`); when a build or check episode is already open, recovery emits neither boundary event, preserving `TestRecoveryDecisionKeepsTheBuildEpisodeLive`'s guarantee.
- `TestEveryLifecycleLaneWritesADurableOutcome`'s `t.Skipf` branch is now a named `t.Fatalf` — all six declared lifecycle lanes pass with zero skip lines.
- A new closed vocabulary, `episodeInterventionKind`, declares exactly three members, each with a real, mutation-proved production writer:
  - `episodeInterventionKindAnsweredWorkerQuestion` ("answered a worker's question") — written by `recordDecisionAnswer` (cmd/handoff_decisions_cmd.go), scoped to non-`seal-`-sourced answers only.
  - `episodeInterventionKindDeclinedForcedReviewer` ("declined a forced reviewer") — written by `resolveForcedReviewerWaiverPendingDecision` (cmd/forced_reviewer_waiver.go), the single site that fires exactly once per real waiver resolution.
  - `episodeInterventionKindReleasedQuarantine` ("released a quarantined candidate") — written by `releaseCanaryQuarantine` (cmd/rollback.go), only on a genuine Quarantined-true-to-false transition, never on a no-op release call.
- `emitColonyLiveInterventionRecorded`'s third parameter is now the typed `episodeInterventionKind`; it refuses (warns to stderr, writes nothing) an undeclared value at the one write point.
- `collectPreventableInterventions` filters to declared kinds only; an episode whose only intervention record carries an undeclared category now falls through to the existing `UnclassifiedEpisodes` list rather than inflating the preventable-intervention figure. A new `collectUnrecognizedInterventionCategories` helper names the excluded category by episode and text.

## Task Commits

1. **Task 1: The swarm lane opens and closes a durable episode** - `7bc71be8` (feat)
2. **Task 2: The recovery lane opens a durable episode only when it owns one, and the skip branch becomes a failure** - `ae6c8c33` (feat)
3. **Task 3: A closed intervention-kind vocabulary, with real production writers** - `82c1b485` (feat)

## Files Created/Modified
- `cmd/swarm_cmd.go` - `runSwarmDestroy` gains the episode-started/ended boundary pair
- `cmd/recovery_orchestrator.go` - `orchestrateRecovery` gains the kind-branched episode boundary for the standalone-recovery case only
- `cmd/live_events.go` - `emitColonyLiveInterventionRecorded`'s third argument becomes the typed, validated `episodeInterventionKind`
- `cmd/episode_ledger.go` - the new `episodeInterventionKind` type, const block, vocabulary slice and helpers; a doc comment on `Interventions` recording that no legacy free-form data exists to migrate
- `cmd/improvement_report.go` - `collectPreventableInterventions` filters to declared kinds; new `collectUnrecognizedInterventionCategories` helper
- `cmd/handoff_decisions_cmd.go` - `recordDecisionAnswer` writes an intervention record for a genuine (non-seal) worker-question answer
- `cmd/forced_reviewer_waiver.go` - `resolveForcedReviewerWaiverPendingDecision` writes an intervention record on a genuine waiver resolution
- `cmd/rollback.go` - `releaseCanaryQuarantine` writes an intervention record on a genuine quarantine release
- `cmd/episode_ledger_test.go` - `t.Skipf` → `t.Fatalf`; new `TestSwarmLaneOpensAndClosesADurableEpisode` and `TestRecoveryLaneOpensADurableEpisodeOnlyWhenItOwnsOne`
- `cmd/live_lane_coverage_test.go` - `driveSwarmLiveLane` now drives `runSwarmDestroy` end to end instead of hand-mirroring its wave emission
- `cmd/intervention_vocabulary_test.go` (new) - AST-derived closed-vocabulary test, AST-derived writer-coverage test, refusal test, three per-writer tests, the LEARN-02 adjacency test, and the unrecognized-category classification test

## Decisions Made

See `key-decisions` in frontmatter: the swarm driver route (drive `runSwarmDestroy` end to end), the single waiver writer site (`resolveForcedReviewerWaiverPendingDecision` on `found=true`), reusing `currentLiveRecoveryEpisode` verbatim for all three intervention writers, and routing an unrecognized category through the existing `UnclassifiedEpisodes` mechanism rather than a new struct field.

## Deviations from Plan

None - plan executed exactly as written. The plan's own flagged assumptions section states plan 204-13 has no open probes of its own.

## FAILS-WHEN-UNWIRED proofs (all five performed, observed, and reverted)

**1. Swarm episode boundary** (`cmd/swarm_cmd.go`, comment out `emitColonyLiveEpisodeStarted(swarmID, events.EpisodeKindSwarm)`):
```
--- FAIL: TestSwarmLaneOpensAndClosesADurableEpisode (0.08s)
    --- FAIL: TestSwarmLaneOpensAndClosesADurableEpisode/a_completed_run_opens_and_closes_exactly_one_episode (0.07s)
        episode_ledger_test.go:615: episode-ended id "swarm-1789417524918245000" does not match episode-started id ""
    --- FAIL: TestSwarmLaneOpensAndClosesADurableEpisode/a_run_that_fails_its_investigation_wave_still_closes_its_episode_with_its_own_terminal_status (0.01s)
        episode_ledger_test.go:680: expected exactly 1 LiveTopicEpisodeStarted even on the failing path, got 0
```

**2. Recovery episode boundary** (`cmd/recovery_orchestrator.go`, comment out `emitColonyLiveEpisodeStarted(recoveryEpisodeID, recoveryEpisodeKind)`):
```
--- FAIL: TestEveryLifecycleLaneWritesADurableOutcome (11.83s)
    --- FAIL: TestEveryLifecycleLaneWritesADurableOutcome/recovery (0.01s)
        episode_ledger_test.go:566: lane "recovery" never emits LiveTopicEpisodeStarted via its real entry point -- this lane has gone dark on the episode boundary
```

**3. Owner-answer writer** (`cmd/handoff_decisions_cmd.go`, comment out the `emitColonyLiveInterventionRecorded` call):
```
--- FAIL: TestOwnerAnswerWritesOneInterventionRecord (0.00s)
    intervention_vocabulary_test.go:285: expected exactly 1 durable intervention record for episode "recovery-phase-204", got 0
```

**4. Declined-reviewer writer** (`cmd/forced_reviewer_waiver.go`, comment out the `emitColonyLiveInterventionRecorded` call):
```
--- FAIL: TestDeclinedReviewerWritesOneInterventionRecord (7.29s)
    intervention_vocabulary_test.go:352: expected exactly 1 durable intervention record for episode "recovery-phase-1", got 0
```

**5. Quarantine-release writer** (`cmd/rollback.go`, comment out the `emitColonyLiveInterventionRecorded` call):
```
--- FAIL: TestQuarantineReleaseWritesOneInterventionRecord (0.02s)
    intervention_vocabulary_test.go:391: expected exactly 1 durable intervention record for episode "recovery-phase-0", got 0
```

Every mutation was reverted immediately after observing the failure; `git diff --stat` on each file after revert showed only additions relative to the pre-mutation state (confirmed via `git diff cmd/swarm_cmd.go` etc.).

## Issues Encountered

None that blocked the plan. One debugging detour: the first attempt at `TestDeclinedReviewerWritesOneInterventionRecord` used `commitTestBuildStart`'s default variant (`buildStartDirect`), whose reviewer-window effect is "close" — this closed the forced-reviewer decline window at commit time, before the test ever tried to create a pending waiver row, making `ensureForcedReviewerWaiverPendingDecision` return an empty capability. Fixed by explicitly requesting `Variant: buildStartPlanOnly` (reviewer-window effect "reopen"), matching the real `runCodexBuildPlanOnlyWithOptions` path `TestWaiverControlsBothFinalContinueDispatchLists`'s own fixture already uses. No production code changed as a result of this detour — it was purely a test-fixture correction.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SC3b is closed: all six lifecycle lanes now open and close a durable episode, proven by `TestEveryLifecycleLaneWritesADurableOutcome` with zero skips, and each new production call site is proven load-bearing by a FAILS-WHEN-UNWIRED mutation.
- The owner-intervention half of SC3a and D-10 are closed: a curated, closed vocabulary with three real writers, each mutation-proved.
- The full SC3a gap (9 other episode-record fields with zero production writers, per 204-VERIFICATION.md) is NOT addressed by this plan and remains open for its own gap-closure plan.
- This repository's pre-existing known-red baseline (17 tests, see `.planning/WINDOWS.md`) was not touched by this plan; none of the scoped test runs above hit any of those 17 names.

## Self-Check: PASSED

- `cmd/intervention_vocabulary_test.go` confirmed present on disk.
- Commits `7bc71be8`, `ae6c8c33`, `82c1b485` all confirmed present in `git log --oneline --all`.
- All plan-level `<verification>` commands re-run clean immediately before writing this summary (`TestEveryLifecycleLaneWritesADurableOutcome` with zero SKIP lines; the full Task 1-3 test list; the regression guard list).

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
