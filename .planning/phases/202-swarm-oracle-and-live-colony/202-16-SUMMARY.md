---
phase: 202-swarm-oracle-and-live-colony
plan: "16"
subsystem: live-colony
tags: [go, live-events, watch, recovery, swarm, cockpit]

# Dependency graph
requires:
  - phase: 202-swarm-oracle-and-live-colony
    provides: "The typed live-colony event model (pkg/events/colony_live.go), the emission boundary and per-lane helpers (cmd/live_events.go), and the pure replay reducer (cmd/live_projection.go) that plans 202-02/03/06/09 built"
provides:
  - "A single shared open/close rule (colonyLiveBoundaryDelta) that both the replay reducer and the live-episode selector derive their boundary balance from"
  - "latestLiveEpisodeID that follows the most recently started STILL-OPEN episode, falling back to the same 'latest started' episode the replay summary names when nothing is open"
  - "currentLiveRecoveryEpisode: recovery decisions route onto the real open build/continue episode instead of a synthetic episode of their own"
  - "Swarm's six live-event payload literals draw their episode kind from the shared events.EpisodeKindSwarm constant"
  - "ParentWorkerID's doc comment states plainly that no production dispatch sets it today"
affects: [202-17, watch, aether-watch-cockpit]

# Actuals (#2632)
actuals:
  tokens: 7593
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single source-of-truth boundary-delta function shared between a reducer and a selector, rather than two independently-maintained balance implementations"
    - "Break-it-to-prove-it regression discipline: temporarily reverting a fix, confirming the named test fails, then restoring it and confirming the test passes again"

key-files:
  created:
    - cmd/live_recovery_episode_test.go
  modified:
    - cmd/live_projection.go
    - cmd/watch_replay.go
    - cmd/watch_live.go
    - cmd/watch_live_test.go
    - cmd/live_events.go
    - cmd/recovery_orchestrator.go
    - cmd/swarm_cmd.go
    - cmd/swarm_lens.go
    - pkg/codex/dispatch.go

key-decisions:
  - "Recovery is demoted from an episode-producing lane to a detail layered onto the episode that owns it (currentLiveRecoveryEpisode); the synthetic recovery-phase-N identifier survives only as the no-open-episode fallback."
  - "The open/close boundary rule (colonyLiveBoundaryDelta) is extracted into its own function rather than kept as inline ++/-- statements, so the reducer and the new selector cannot silently drift apart on what 'open' means."
  - "latestStartedLiveEpisodeAmong is the one implementation of 'latest started episode' in the package -- both the replay branch (mostRecentlyStartedLiveEpisode, over all episodes) and the live branch (latestLiveEpisodeID, over the open subset, falling back to all episodes) call it."

requirements-completed: [LIVE-02, CEC-05]

coverage:
  - id: D1
    description: "The live view follows the most recently started still-open episode, not whichever episode owns the chronologically newest single event; with nothing open, live and replay name the same episode."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_live_test.go#TestWatchFollowsTheMostRecentlyStartedOpenEpisode"
        status: pass
      - kind: unit
        ref: "cmd/watch_live_test.go#TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames"
        status: pass
      - kind: unit
        ref: "cmd/watch_live_test.go#TestOneOpenBalanceRule"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchResolvesThreeBranchesFromEvidenceAlone"
        status: pass
    human_judgment: false
  - id: D2
    description: "A recovery decision fired while a build or check episode is open is recorded on that episode (not a synthetic one of its own); a recovery decision with nothing open is still recorded under its own fallback identifier."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/live_recovery_episode_test.go#TestRecoveryDecisionKeepsTheBuildEpisodeLive"
        status: pass
      - kind: unit
        ref: "cmd/live_lane_coverage_test.go#TestEveryLifecycleLaneEmitsLiveEvents"
        status: pass
      - kind: unit
        ref: "cmd/live_projection_test.go#TestEveryLiveEventGoesThroughOneBoundary"
        status: pass
    human_judgment: false
  - id: D3
    description: "Swarm's six live-event payload literals draw the episode kind from the shared events.EpisodeKindSwarm constant instead of a hand-typed string; ParentWorkerID's doc comment states no production dispatch sets it."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/watch_live_test.go#TestSwarmInvestigationWaveReachesTheLiveWatchScreen"
        status: pass
      - kind: unit
        ref: "cmd/swarm_lens_test.go#TestFourSwarmLensesProduceDistinctEvidence"
        status: pass
      - kind: unit
        ref: "cmd/swarm_episode_test.go#TestSwarmRunProducesOneReplaySafeEpisode"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 16: Gap Closure -- Recovery No Longer Hijacks the Live Colony View Summary

**A recovery decision fired mid-build now speaks on the real running episode instead of an orphaned synthetic one, and the live-view selector prefers the most recently started still-open episode over whichever episode owns the newest single event.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-11T19:54:53Z (approx, first commit)
- **Completed:** 2026-09-11T19:59:32Z
- **Tasks:** 3
- **Files modified:** 9 modified, 1 created

## Accomplishments

- `colonyLiveBoundaryDelta` is now the single shared open/close rule; `foldColonyLiveEvents` (the reducer) and the new `openColonyLiveEpisodeIDs` (the selector) both derive their balance from it, so they can never silently disagree about what "still open" means.
- `latestLiveEpisodeID` (`cmd/watch_live.go`) now selects the most recently started STILL-OPEN episode (via the shared `latestStartedLiveEpisodeAmong`, extracted from `cmd/watch_replay.go`'s `mostRecentlyStartedLiveEpisode`), falling back to the same "latest started" episode the replay summary names when nothing is open. Closes CR-02's episode-selection half.
- `currentLiveRecoveryEpisode` (`cmd/live_events.go`) resolves recovery's owning episode: the active build carrier, then the active continue carrier, then a phase-derived fallback only when neither is set. `orchestrateRecovery`'s single emission call site now routes through it instead of always minting a synthetic `recovery-phase-N` identifier. Closes CR-02's recovery-routing half.
- Swarm's six hand-typed `EpisodeKind: "swarm"` literals across `cmd/swarm_cmd.go` and `cmd/swarm_lens.go` now read `events.EpisodeKindSwarm` (WR-04). `pkg/codex/dispatch.go`'s `ParentWorkerID` doc comment now states plainly that no production dispatch assigns it today (WR-03).

## Task Commits

Each task was committed atomically:

1. **Task 1: The live view follows the most recently started still-open episode** - `1e1662f0` (fix)
2. **Task 2: Recovery speaks on the episode it happened inside** - `e9e162f2` (fix)
3. **Task 3: Swarm's episode kind comes from the shared constant; lineage plumbing is labelled** - `14f22aed` (docs)

## Files Created/Modified

- `cmd/live_projection.go` - Added `colonyLiveBoundaryDelta` (the shared open/close rule) and `openColonyLiveEpisodeIDs` (the per-episode open set); `foldColonyLiveEvents`'s balance arithmetic now derives its delta from the shared function instead of literal `++`/`--`
- `cmd/watch_replay.go` - Extracted `latestStartedLiveEpisodeAmong` from `mostRecentlyStartedLiveEpisode`'s own selection loop; `mostRecentlyStartedLiveEpisode` now calls the extracted function
- `cmd/watch_live.go` - Rewrote `latestLiveEpisodeID` to prefer the open subset (via `openColonyLiveEpisodeIDs` + `latestStartedLiveEpisodeAmong`), falling back to every episode ID when none is open
- `cmd/watch_live_test.go` - Added `TestWatchFollowsTheMostRecentlyStartedOpenEpisode`, `TestClosedEpisodesPickTheSameEpisodeTheReplaySummaryNames`, `TestOneOpenBalanceRule`
- `cmd/live_events.go` - Added `currentLiveRecoveryEpisode(phase int) (string, string)`, resolving build carrier -> continue carrier -> phase-derived fallback
- `cmd/recovery_orchestrator.go` - `orchestrateRecovery`'s single emission call site now resolves through `currentLiveRecoveryEpisode` instead of always building `recovery-phase-N`; removed the now-unused `events` import
- `cmd/live_recovery_episode_test.go` (new) - `TestRecoveryDecisionKeepsTheBuildEpisodeLive` with 4 subtests (build-open, continue-open, no-episode-open fallback, single-episode-ID proof)
- `cmd/swarm_cmd.go`, `cmd/swarm_lens.go` - Six `EpisodeKind: "swarm"` literals changed to `EpisodeKind: events.EpisodeKindSwarm`
- `pkg/codex/dispatch.go` - `ParentWorkerID`'s doc comment extended to state no production dispatch currently sets it

## Decisions Made

- Recovery is a **detail** layered onto the episode that owns it, never a screen of its own; the synthetic `recovery-phase-N` identifier survives solely as the no-live-episode fallback (matches the plan's `<assumption_delta_decision>`).
- Build carrier is checked before continue in `currentLiveRecoveryEpisode` -- documented as unable to matter in the current runtime, since each carrier is cleared by its own deferred restore before the other lane's call chain can run.
- The balance-delta extraction in `foldColonyLiveEvents` was implemented by computing the delta once before the topic switch (rather than duplicating `colonyLiveBoundaryDelta(evt.Topic)` inside two separate switch cases), preserving the floor-at-zero behavior and leaving every other per-topic field assignment (StartedAt, Wave, ElapsedSeconds, worker rows, etc.) untouched.

## Deviations from Plan

None - plan executed exactly as written. All three tasks matched their `<action>` specifications; both break-it-to-prove-it observations (Task 1 and Task 2) confirmed the named tests fail when the fix is reverted and pass when restored.

### Break-it-to-prove-it observations

1. **Task 1 (episode selection):** Temporarily reverted `latestLiveEpisodeID` to the old "whichever episode owns the newest single event" implementation. `TestWatchFollowsTheMostRecentlyStartedOpenEpisode` failed with `resolveWatchMode mode = "replay", want "live" -- the older episode is still open`. Restored the fix; test passed again, along with the full scoped suite.
2. **Task 2 (recovery routing):** Temporarily reverted `orchestrateRecovery`'s emission call site to always build `recovery-phase-N` with a bare `"recovery"` kind. 3 of 4 `TestRecoveryDecisionKeepsTheBuildEpisodeLive` subtests failed: the build-open and continue-open subtests reported `snapshot.RecoveryState = ""` (recovery state landed on a disconnected episode the live snapshot never saw), and the single-episode-ID subtest found 2 distinct episode IDs instead of 1 (`recovery-phase-202` and the real build episode). Restored the fix; all 4 subtests passed again.

---

**Total deviations:** 0
**Impact on plan:** None -- plan executed exactly as specified, both regressions were proven and then fixed as designed.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 202-17 (Oracle liveness, CR-01) is independent of this plan's changes but shares the `CEC-05` requirement ID; `requirements.ready-ids` will hold `CEC-05` at "not yet complete" until 202-17 also finishes, per the shared-ID gate (#2388).
- Episode-identity insight for 202-17: Oracle currently never opens/closes a `colonyLiveBoundaryDelta`-tracked boundary at all (per 202-VERIFICATION.md's CR-01 finding), so `openColonyLiveEpisodeIDs` will never include an Oracle episode ID until 202-17 wraps Oracle's run start/stop in `emitColonyLiveEpisodeStarted`/`emitColonyLiveEpisodeEnded` (or an equivalent worker-started/finished pair). The shared selector this plan built (`latestStartedLiveEpisodeAmong`) will pick up an Oracle episode automatically once 202-17 makes it open a boundary -- no further selector-side work should be needed.
- No blockers. `go build ./...` and `go vet ./cmd ./pkg/codex` are clean; all named Phase 202 tests plus the two 202-16-specific regression tests pass. The two pre-existing failures (`TestPhase199GateReceipt`, `TestCurrentVocabulary199`) were re-confirmed as unrelated and unaffected by this plan's changes.

## Self-Check: PASSED

- `cmd/live_recovery_episode_test.go` exists on disk: confirmed.
- All three commit hashes (`1e1662f0`, `e9e162f2`, `14f22aed`) found in `git log --oneline --all`.
- `go build ./...` and `go vet ./cmd ./pkg/codex` exit 0.
- Full named scoped test list from the plan's `<verification>` block re-run and passed (see Accomplishments and Task Commits above).

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
