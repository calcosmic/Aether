---
phase: 202-swarm-oracle-and-live-colony
plan: "10"
subsystem: swarm
tags: [swarm, episode, retention, cleanup, learning-proposal, live-colony]

# Dependency graph
requires:
  - phase: 202-05
    provides: "cmd/swarm_lens.go's swarmComparison/swarmHypothesis/compareSwarmHypotheses -- the structured comparison this plan durably records onto the episode."
  - phase: 202-07
    provides: "cmd/swarm_repair_checkpoint.go's checkpoint save/restore -- the episode's Checkpoint.Saved/Restored fields mirror this outcome rather than re-deriving it."
provides:
  - "cmd/swarm_episode.go: swarmEpisodeRecord -- the one durable, replay-safe episode per Swarm run (lenses, hypotheses, comparison, checkpoint, verification outcome, strike standing, a cost reference, and an optional learning proposal), bound to the existing swarmResultRecord by SwarmID rather than replacing it."
  - "persistSwarmEpisode/loadSwarmEpisode/buildSwarmEpisodeRecord for completed runs; persistInterruptedSwarmEpisode for a run that stops mid-flight, naming the stage it stopped at."
  - "planSwarmEpisodeRetention/removeSwarmEpisodes: retention that acts on an exact episode identifier plus the digest it was previewed at -- never a directory sweep -- and never removes the latest episode for a target or one belonging to an unresolved strike sequence."
  - "proposeSwarmLearningFromEpisode: at most one focus or avoid-this note from a passed, non-rolled-back repair, attached to the episode as a proposal only, sanitized, and never written as an active signal."
affects: [202-11, 202-12, 202-13, 202-14, 202-15]

# Actuals (#2632)
actuals:
  tokens: 16900
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive durable record bound by shared identity: swarmEpisodeRecord is written alongside every existing persistSwarmResultOutcome call site (including runSwarmFinalize's external-dispatch lane), sharing swarmID as the join key. swarmResultRecord, saveSwarmResultRecord, and strike evaluation (evaluateSwarmStrikeHistory/ensureSwarmEscalationForHistory) are never modified -- proven by TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates passing unmodified."
    - "Identity fail-closed by reuse, not reinvention: persistSwarmEpisode validates SwarmID through validateDurableSwarmID (cmd/swarm_issuance.go) -- the exact function runSwarmFinalize already uses on manifest.SwarmID -- rather than a parallel identity check that could drift from it over time."
    - "Retention computed fresh, never cached: swarmEpisodeRetentionMeta stores only a class label. Age and eligibility are recomputed on every planSwarmEpisodeRetention call from the CURRENT full episode set (latest-for-target, unresolved-strike-sequence), so a value frozen at write time can never go stale the moment a newer episode for the same target appears."
    - "Manifest-and-digest deletion discipline, reused: removeSwarmEpisodes requires the exact identifier (validateDurableSwarmID again) AND the digest the caller previewed (swarmEpisodeFileDigest, sha256 of the raw episode.json bytes). Any mismatch refuses the WHOLE batch before deleting anything, naming the first mismatched episode -- the same exact-path/owner/digest authority this repository already applies elsewhere to maintenance deletion."
    - "Deterministic mid-run interruption test seam: swarmMidRunInterruptFunc (mirroring newSwarmWorkerInvoker/swarmRestoreRepairCheckpointFunc) is a no-op in production, called once right after the investigation wave ends. A test sets it to cancel the run's own context, producing a genuine ctx.Err() from the real public Swarm path instead of racing a sleep against a timeout."

key-files:
  created:
    - cmd/swarm_episode.go
    - cmd/swarm_episode_test.go
  modified:
    - cmd/swarm_cmd.go

key-decisions:
  - "Interrupted-episode stage naming: InterruptedStage names the wave DURING which the run stopped (\"investigation\"/\"fix\"/\"verification\"), not the last wave that fully completed. A stop between investigation and fix is recorded as stage \"fix\" -- the wave that was attempted and cut off -- because that is strictly more informative than naming the prior completed boundary, while the episode's Lenses field already carries which investigation lenses did report."
  - "Cost reference is the spawn-tree run identifier (runHandle.Run.ID via beginRuntimeSpawnRun/agent.SpawnTree), not a token or currency figure. Swarm has no per-run spend ledger today (spendLedger is keyed by phase+workflow, and no writeSpendRowsForRun call site exists for swarm) -- the spawn run ID is the one genuine ledger key the runtime already issues for a Swarm run, satisfying \"reference the ledger keys, never copy a figure\" without inventing a new money-adjacent record."
  - "The learning proposal derives a focus note from Comparison.SharedCauses when two or more lenses independently agreed on a cause, or an avoid-this note from the highest-ranked non-selected candidate when a synthesized repair had a runner-up; a single, uncorroborated hypothesis with no runner-up proposes nothing. Both branches are gated on VerificationStatus == \"completed\" AND Checkpoint.Restored == false AND a selected repair existing, so a failed, rolled-back, or no-evidence run can never reach either branch."
  - "runSwarmFinalize (the external-task dispatch lane) also persists an episode, built from the merged worker results, with Checkpoint left {false,false} since external dispatch has no local checkpoint mechanism -- keeping LIVE-05's 'one replay-safe episode per run' true for both dispatch lanes, not only the in-process one the plan's read_first list centered on."

patterns-established:
  - "Episode fields mirror decision-bearing structs (swarmComparison, swarmStrikeHistory) by value rather than re-deriving them at read time, so a later replay of the episode reflects exactly what the run itself computed, not a fresh recomputation that could disagree with it."

requirements-completed: [LIVE-05]

coverage:
  - id: D1
    description: "One Swarm run produces exactly one durable, issuance-bound, replay-safe episode carrying its lenses, hypotheses, comparison, checkpoint, verification result, strike standing and cost reference, bound to the existing result record without replacing it."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRunProducesOneReplaySafeEpisode"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmEpisodeRereadNeverChangesStrikeTruth"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmEpisodeIdentityComesFromIssuance"
        status: pass
      - kind: unit
        ref: "cmd/swarm_cmd_test.go#TestSwarmThreeStrikeRecoveryExactTargetRetryReEscalates"
        status: pass
    human_judgment: false
  - id: D2
    description: "A run interrupted after the investigation wave persists one episode marked interrupted, naming the stage it stopped at and carrying the lenses that did report."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestInterruptedSwarmRunPersistsOneInterruptedEpisode"
        status: pass
    human_judgment: false
  - id: D3
    description: "Retention and cleanup act on a named episode's exact identifier and previewed digest, never a directory sweep; the latest episode for a target and any episode in an unresolved strike sequence are never eligible; a stale digest refuses removal before anything is deleted; removal never alters strike history."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRetentionNeverRemovesTheLatestOrAStrikeEpisode"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRemovalRequiresIdentifierAndDigest"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRemovalRefusesChangedDigest"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRetentionPreviewIsReadOnly"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRemovalLeavesStrikeHistoryUnchanged"
        status: pass
    human_judgment: false
  - id: D4
    description: "A run whose repair passed verification and was never rolled back may propose one scoped focus/avoid-this note with its evidence, sanitized, carrying no field asserting a measured effect and never written as an active signal; a failed, rolled-back, or no-evidence run proposes nothing."
    requirement: "LIVE-05"
    verification:
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSuccessfulSwarmProposesOneScopedNote"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestFailedOrRolledBackSwarmProposesNothing"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmLearningProposalIsSanitized"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmLearningProposalClaimsNoEffect"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 10: Swarm Episode, Retention, and Learning Proposal Summary

**Every Swarm run -- in-process or externally dispatched -- now leaves exactly one durable, replay-safe episode (`cmd/swarm_episode.go`) carrying its lenses, hypotheses, comparison, checkpoint, verification outcome, strike standing and a spend-ledger cost reference; retention and cleanup act only on a named episode's exact identifier plus its previewed digest, and a passing repair may leave one sanitized focus/avoid-this learning proposal that claims no measured effect.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-11T13:40:00Z (approximate)
- **Completed:** 2026-09-11T14:56:00Z
- **Tasks:** 3
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- `cmd/swarm_episode.go`: `swarmEpisodeRecord` -- the one durable episode per run, bound to the existing `swarmResultRecord` by `SwarmID` rather than replacing it. Its identity is validated through the same `validateDurableSwarmID` the external finalizer already uses, so a caller-supplied identifier that isn't the runtime-issued convention is refused before anything is written.
- `runSwarmDestroy` (in-process dispatch) and `runSwarmFinalize` (external-task dispatch) both call `persistSwarmEpisode` alongside their existing `persistSwarmResultOutcome` call, reusing the strike history that call already returns. Every wave-error branch (investigation, fix, verification) now persists one `interrupted` episode naming the wave it was attempting when it stopped, proven deterministically via a new `swarmMidRunInterruptFunc` test seam that cancels the run's own context rather than racing a sleep against a timeout.
- `planSwarmEpisodeRetention`/`removeSwarmEpisodes`: a pure-read preview that marks an episode ineligible only when it is the most recent for its target or belongs to a target's currently unresolved strike sequence (re-derived from `evaluateSwarmStrikeHistory`, never a second source of truth); removal requires the exact identifier and the digest the caller previewed, refuses a prefix/glob or a stale digest, refuses the whole batch on the first mismatch, and only ever deletes `episode.json` -- `result.json` and strike history are untouched.
- `proposeSwarmLearningFromEpisode`: derives at most one focus note (from a shared corroborated cause) or avoid-this note (from a non-selected runner-up candidate) when a repair passed verification and was never rolled back; the text is sanitized through `colony.SanitizeSignalContent`, the proposal is attached to the episode only, and nothing about it asserts the note was applied or effective.

## Task Commits

Each task was committed atomically:

1. **Task 1: One issuance-bound, replay-safe episode per run** - `e739a14b` (feat)
2. **Task 2: Retention and cleanup that act on a named episode** - `dd60aac8` (feat)
3. **Task 3: Propose a scoped note from successful evidence, without claiming it worked** - `d503c990` (feat)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `cmd/swarm_episode.go` - `swarmEpisodeRecord` and every function that builds, persists, reads, retains, removes, and proposes learning for it
- `cmd/swarm_episode_test.go` - 13 tests, each driving either the real public Swarm path (`aether swarm <target>`, `aether swarm-finalize`) or the pure `proposeSwarmLearningFromEpisode`/`planSwarmEpisodeRetention`/`removeSwarmEpisodes` functions directly
- `cmd/swarm_cmd.go` - wired episode persistence into `runSwarmDestroy`'s four completion points and three wave-error branches, and into `runSwarmFinalize`'s external completion; added the `swarmMidRunInterruptFunc` test seam and a `checkpointRestored` tracking variable

## Decisions Made

See `key-decisions` in the frontmatter above: interrupted-stage naming (the wave it was attempting, not the last completed boundary), the cost reference (spawn-tree run ID, since Swarm has no per-run spend ledger today), the learning-proposal gating (shared cause -> focus, non-selected runner-up -> avoid, single hypothesis -> nothing), and extending episode persistence to the external-finalize lane as well as the in-process one.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `swarmRunTimeoutDuration` seam was unnecessary; replaced by a direct cancel-hook seam**
- **Found during:** Task 1, while designing `TestInterruptedSwarmRunPersistsOneInterruptedEpisode`
- **Issue:** The plan's acceptance criteria require proving the interrupted path against the real public Swarm path. A timeout-based approach (shrinking `defaultSwarmRunTimeout` and sleeping past it in a test invoker) would have been timing-dependent and flaky.
- **Fix:** Added `swarmMidRunInterruptFunc`, a no-op-in-production package variable called once after the investigation wave ends, which a test can set to call the run's own `cancel()` directly -- deterministic, zero-flake, and still exercises the real `ctx.Err()` branch in `runSwarmDestroy`.
- **Files modified:** cmd/swarm_episode.go, cmd/swarm_cmd.go, cmd/swarm_episode_test.go
- **Verification:** `TestInterruptedSwarmRunPersistsOneInterruptedEpisode` passes consistently across repeated runs
- **Committed in:** e739a14b (Task 1 commit)

**2. [Rule 2 - Missing Critical] Extended episode persistence to `runSwarmFinalize` (external-task dispatch), not only `runSwarmDestroy`**
- **Found during:** Task 1, after wiring the in-process path
- **Issue:** The plan's must-haves state "One Swarm run produces exactly one durable... episode" without scoping that to only the in-process dispatch lane. Leaving the external-finalize lane (`aether swarm-finalize`, used by the host-driven/agent-delegate path) without an episode would mean roughly half of real Swarm runs never got one, silently violating LIVE-05.
- **Fix:** Added the same `persistSwarmEpisode` call, built from the merged external worker results, right before `completeExternalSwarmFinalization` in `runSwarmFinalize`.
- **Files modified:** cmd/swarm_cmd.go
- **Verification:** Full `Swarm`-matching test suite (including `TestSwarmThreeStrikeExternalFinalizeReplayKeepsOneEscalation`, which exercises the exact-replay early-return path) passes unchanged
- **Committed in:** e739a14b (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both were necessary for correctness/completeness of LIVE-05 as stated. No scope creep beyond what the plan's must-haves already required.

## Issues Encountered

None. All three tasks' `<verify>` commands passed on first implementation attempt (Task 1 and Task 2 tests passed immediately; Task 3's full-dispatch test needed one iteration to fix the error-envelope assertion reading `stdout` instead of `stderr`, and the interrupted test needed the same fix -- both are test-fixture corrections, not production-code deviations).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- LIVE-05 satisfied: every Swarm run (in-process or external) now leaves one durable, replay-safe episode with retention governed by exact identifier and digest, never a directory sweep.
- `cmd/swarm_episode.go`'s `swarmEpisodeRecord`, `loadSwarmEpisode`, `planSwarmEpisodeRetention`, and `proposeSwarmLearningFromEpisode` are available for later plans in this phase (202-11 through 202-15) that surface Swarm activity in status/history or consume the learning proposal.
- No blockers. `go build ./cmd/...`, `go vet ./cmd`, and the full `Swarm`-matching test suite (including every pre-existing three-strike/checkpoint/lens test) are clean.

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
