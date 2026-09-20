---
phase: 202-swarm-oracle-and-live-colony
plan: "09"
subsystem: live-events
tags: [watch, live-colony, replay, idle-floor, cockpit]

# Dependency graph
requires:
  - phase: 202-02
    provides: "pkg/events/colony_live.go's one versioned live-colony event model, cmd/live_events.go's emitColonyLive single emission boundary, and cmd/live_projection.go's replayColonyLiveSnapshot pure reducer this plan replays through."
  - phase: 202-06
    provides: "cmd/watch_dashboard.go's renderColonyLiveTicker (reused verbatim for the replay episode's own final events) and the live cockpit this plan sits beside as the second of three watch branches."
provides:
  - "cmd/watch_replay.go: buildReplayWatchResult / renderReplayWatchVisual -- the replay-backed summary of the most recently STARTED persisted live episode (tie-broken by sequence number), naming its kind, worker count, terminal outcome (or that it was interrupted, naming the last recorded transition), elapsed time, cost (read from the existing spend ledger authority), and the lifecycle projection's own next command."
  - "cmd/watch_live.go: resolveWatchMode now demotes an episode left open by a run that has itself already reached a terminal status to the replay branch instead of reporting it live -- the episode-level application of the exact rule applyUnfinishedWorkerInterruption already applies to an individual worker row."
  - "cmd/watch_live.go: writeColonyWatchFrame's replay branch is wired to buildReplayWatchResult/renderReplayWatchVisual, completing the three-way live/replay/idle resolution watchModeReplay was reserved for since 202-02."
  - "cmd/watch_idle_199_test.go: the Phase 199 idle floor's three original tests now assert the idle branch was actually chosen (mode == idle_watch), and a new case proves a fixture with one recorded episode no longer reaches it."
affects: [202-15]

# Actuals (#2632)
actuals:
  tokens: 8466
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Episode-start selection reuses the total order already defined for resume: mostRecentlyStartedLiveEpisode picks, per episode, the earliest boundary-start event (live.episode.started, falling back to the earliest live.wave.started for a lane like Swarm that never opens an episode-level boundary, and further falling back to the episode's own earliest event of any kind for a lane like Oracle with no boundary at all), then compares those start events across episodes using the exact (timestamp, sequence, event ID) ordering colonyLiveEntryAfterCursor already defines for resuming a replay -- no second ordering rule."
    - "Pre-rendered text blocks over structured map values: buildReplayWatchResult calls the shared renderers (renderColonyLiveTicker, renderSpendCostLineFromLedgers) itself and stores the finished strings in its result map, rather than storing structured slices/ledgers for the render function to re-render -- sidesteps any JSON-round-trip type-assertion fragility and keeps renderReplayWatchVisual a straight assembly of already-correct text."
    - "Open is demoted, never re-derived: resolveWatchMode still computes snapshot.Open exactly as before (start/end boundary balance); a terminated owning run only changes which MODE that Open value routes to (live vs. replay), so the replayed snapshot fact and the mode decision stay separately inspectable and never disagree with the dashboard's own worker-level interruption rule."

key-files:
  created:
    - cmd/watch_replay.go
    - cmd/watch_replay_test.go
  modified:
    - cmd/watch_live.go
    - cmd/compatibility_cmds.go
    - cmd/watch_idle_199_test.go

key-decisions:
  - "An episode's 'start event' for recency ranking is its boundary-start event (episode.started, or the earliest wave.started when no episode-level boundary was ever emitted), not literally its first-ever recorded event -- because a per-episode sequence counter always starts fresh at 1 for a brand-new episode, so two different episodes' genuinely-first events can never carry a meaningfully comparable sequence number. Treating the boundary-start event as the comparison point (with a non-boundary event optionally consumed first) is what makes the plan's own 'equal timestamp, higher sequence wins' acceptance criterion achievable through real emission rather than a hand-typed fixture."
  - "colonyLiveEpisodeRunHasTerminated reads the durable spawn-run record and demotes an Open episode to the replay branch only when a run record EXISTS and is terminal -- an absent run record is never treated as terminated, so a genuinely still-running episode with no spawn-run bookkeeping yet (the common case in every existing live test) is unaffected and stays live."
  - "buildReplayWatchResult performs its own independent episode selection (mostRecentlyStartedLiveEpisode) rather than reusing resolveWatchMode's returned snapshot -- the two can legitimately name different episodes in principle (resolveWatchMode's selection is anchored to the latest EVENT overall; the replay summary's selection is anchored to the latest episode START), so the replay branch always replays whichever episode is actually the most recently started one, not merely whichever one resolveWatchMode happened to inspect first."

requirements-completed: [LIVE-02]

coverage:
  - id: D1
    description: "With nothing running but at least one episode recorded, the live view shows a replay-backed summary (what ran, how many workers, how it ended, how long, what it cost, the one next command) built purely by replaying real persisted events -- never inventing activity or presenting a recorded row as current."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchReplaysTheMostRecentEpisode"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestReplaySummaryCostComesFromTheLedger"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestReplaySummaryNextCommandComesFromTheProjection"
        status: pass
    human_judgment: false
  - id: D2
    description: "An episode whose end event is missing is summarised as interrupted, naming the last recorded transition and its time -- never as completed, whether closed cleanly or left open by a run that has since terminated."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestReplaySummaryOfInterruptedEpisodeSaysInterrupted"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchResolvesThreeBranchesFromEvidenceAlone (open episode whose owning run has terminated resolves replay, not live)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The three-way live/replay/idle choice is resolved in Go alone from persisted evidence -- no flag, environment variable, or wrapper input can select a branch the recorded evidence does not support -- and resolving/rendering any of the three branches writes nothing."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchResolvesThreeBranchesFromEvidenceAlone"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchIsReadOnlyInEveryBranch"
        status: pass
    human_judgment: false
  - id: D4
    description: "The Phase 199 honest idle card survives unchanged for the genuinely-no-history case, and its scope is now explicit: a fixture with one recorded episode no longer reaches it."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_idle_199_test.go#TestWatchIdle199StatusFallback"
        status: pass
      - kind: unit
        ref: "cmd/watch_idle_199_test.go#TestWatchIdle199NoFakeLiveness"
        status: pass
      - kind: unit
        ref: "cmd/watch_idle_199_test.go#TestWatchIdle199ReadOnly"
        status: pass
      - kind: unit
        ref: "cmd/watch_idle_199_test.go#TestWatchIdle199NarrowedByRecordedEpisode"
        status: pass
    human_judgment: false

duration: 18min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 09: The Replay-Backed Middle Branch of Watch Summary

**`aether watch` now fills the branch Phase 199 deliberately left idle: between runs it replays the most recently started episode's real persisted events into a truthful "what just happened" summary, while the honest no-history idle card survives unchanged and the whole three-way choice is resolved in Go, read-only, with no branch ever claiming a recorded row is current liveness.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-09-11T14:16:50+02:00
- **Completed:** 2026-09-11T14:33:54+02:00
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `cmd/watch_replay.go`'s `buildReplayWatchResult` selects the most recently STARTED persisted episode (`mostRecentlyStartedLiveEpisode`, comparing each episode's own boundary-start event by the same (timestamp, sequence, event ID) total order `replayColonyLiveSnapshotResume` already uses to resume), replays it, and reports its kind, identifier, worker count, terminal outcome (or, for an episode with a start and no matching end, that it was interrupted and its own last recorded transition), elapsed time, reported cost (`loadSpendLedgersForPhase` / `renderSpendCostLineFromLedgers` -- the one existing spend authority, never recomputed here), and the lifecycle projection's own next command (exactly as `buildIdleWatchResult` already takes it).
- `renderReplayWatchVisual` renders that result: an opening line stating plainly this is the last thing that ran and nothing is running now, the episode's identity and outcome, its own recent-events ticker (reusing `renderColonyLiveTicker` verbatim -- no second ticker renderer), and the reported-cost block last.
- `resolveWatchMode` (`cmd/watch_live.go`) now derives "open" from the episode's own start-without-end evidence PLUS the owning run's terminal status: a start boundary with no end is still genuinely open while its dispatching run is alive, but once that durable run record (`latestSpawnRunRaw`) reaches a terminal status, the episode is demoted to the replay branch (as interrupted) rather than reported live -- the episode-level twin of the exact rule the dashboard already applies to an individual unfinished worker row, so the two surfaces can never disagree.
- `writeColonyWatchFrame`'s `watchModeReplay` case, dormant since 202-02 ("falls through to the honest idle floor... until then"), now calls `buildReplayWatchResult`/`renderReplayWatchVisual` directly.
- `cmd/watch_idle_199_test.go`'s three Phase 199 tests now assert `mode == "idle_watch"` explicitly (their original content assertions are untouched), and a new `TestWatchIdle199NarrowedByRecordedEpisode` proves a fixture that is otherwise identical but carries one closed episode no longer reaches the idle branch.
- `watchCmd`'s `Short` description updated to name all three modes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Summarise the most recent episode from replayed events** - `da51626d` (feat, TDD)
2. **Task 2: Resolve the three branches in Go and keep the honest floor intact** - `5102df45` (feat, TDD)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/watch_replay.go` - `buildReplayWatchResult`, `renderReplayWatchVisual`, `mostRecentlyStartedLiveEpisode`, `colonyLiveEpisodeStartEntry`, `colonyReplayOutcomeStatus`, `colonyLiveLastTransition`.
- `cmd/watch_replay_test.go` - The four Task 1 tests plus `TestWatchResolvesThreeBranchesFromEvidenceAlone` and `TestWatchIsReadOnlyInEveryBranch` (Task 2).
- `cmd/watch_live.go` - `resolveWatchMode`'s terminated-run demotion, new `colonyLiveEpisodeRunHasTerminated`, `writeColonyWatchFrame`'s wired `watchModeReplay` case.
- `cmd/compatibility_cmds.go` - `watchCmd.Short` now names all three modes.
- `cmd/watch_idle_199_test.go` - Explicit `mode == "idle_watch"` assertions on the three original tests, plus the new narrowing case.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A broader regression sweep (`go test ./cmd -run 'Watch|Live|Spawn|Ticker|Dashboard' -count=1`, beyond the plan's own scoped `<verify>` commands, run deliberately to catch any cross-file effect of changing `resolveWatchMode`'s open/live decision) surfaced one failure: `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount` in `cmd/codex_build_test.go`. Confirmed pre-existing and unrelated by stashing every file this plan touched and re-running the same test alone against the unmodified tree -- it fails identically. Out of scope (this plan never touches spawn-budget/caste-allocation code); not auto-fixed, left as-is.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three `aether watch` branches (live, replay, idle) are now real; `watchModeReplay`'s "plan 202-09 owns this" placeholder from 202-02 is resolved.
- 202-15 (final acceptance/discoverability) depends on this plan plus five others (202-07, 202-10 through 202-14) and is not yet unblocked.
- No blockers.

## Self-Check: PASSED

- `cmd/watch_replay.go` — FOUND
- `cmd/watch_replay_test.go` — FOUND
- `cmd/watch_live.go`, `cmd/compatibility_cmds.go`, `cmd/watch_idle_199_test.go` — modified, confirmed via `git show --stat`
- Commit `da51626d` — FOUND in `git log --oneline`
- Commit `5102df45` — FOUND in `git log --oneline`
- Task 1 `<verify>` regex (`TestWatchReplaysTheMostRecentEpisode|TestReplaySummaryCostComesFromTheLedger|TestReplaySummaryOfInterruptedEpisodeSaysInterrupted|TestReplaySummaryNextCommandComesFromTheProjection`) — PASS
- Task 2 `<verify>` regex (`TestWatchResolvesThreeBranchesFromEvidenceAlone|TestWatchIsReadOnlyInEveryBranch|TestWatchIdle199StatusFallback|TestWatchIdle199NoFakeLiveness|TestWatchIdle199ReadOnly`) plus `go vet ./cmd` — PASS
- `go build ./...` — PASS
- Broader `go test ./cmd -run 'Watch|Live|Spawn|Ticker|Dashboard' -count=1` — PASS except the confirmed pre-existing, unrelated `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
