---
phase: 202-swarm-oracle-and-live-colony
reviewed: 2026-09-11T00:00:00Z
depth: standard
files_reviewed: 71
files_reviewed_list:
  - .aether/commands/oracle.yaml
  - .aether/commands/swarm.yaml
  - .aether/commands/watch.yaml
  - .claude/commands/ant-oracle.md
  - .claude/commands/ant-swarm.md
  - .claude/commands/ant-watch.md
  - .claude/commands/ant/oracle.md
  - .claude/commands/ant/swarm.md
  - .claude/commands/ant/watch.md
  - .opencode/commands/ant/oracle.md
  - .opencode/commands/ant/swarm.md
  - .opencode/commands/ant/watch.md
  - CLAUDE.md
  - cmd/classic_contract_202_test.go
  - cmd/classic_contract_test.go
  - cmd/claudemd_live_colony_test.go
  - cmd/codex_build_worktree.go
  - cmd/codex_build.go
  - cmd/codex_continue.go
  - cmd/codex_plan.go
  - cmd/compatibility_cmds.go
  - cmd/episode_index_test.go
  - cmd/episode_index.go
  - cmd/history.go
  - cmd/lifecycle_history_199_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/live_events.go
  - cmd/live_lane_coverage_test.go
  - cmd/live_lane_wiring_test.go
  - cmd/live_model_singleton_test.go
  - cmd/live_projection_test.go
  - cmd/live_projection.go
  - cmd/memory_feed_continue_test.go
  - cmd/oracle_autofile_198_2_test.go
  - cmd/oracle_brief.go
  - cmd/oracle_live_test.go
  - cmd/oracle_live.go
  - cmd/oracle_loop.go
  - cmd/oracle_preset_test.go
  - cmd/oracle_preset.go
  - cmd/oracle_progress.go
  - cmd/oracle_research_doc_test.go
  - cmd/oracle_research_doc.go
  - cmd/oracle_synthesis_test.go
  - cmd/oracle_synthesis.go
  - cmd/orientation_agreement_199_test.go
  - cmd/partial_work_label_test.go
  - cmd/partial_work_label.go
  - cmd/recovery_orchestrator.go
  - cmd/status.go
  - cmd/swarm_cmd_test.go
  - cmd/swarm_cmd.go
  - cmd/swarm_episode_test.go
  - cmd/swarm_episode.go
  - cmd/swarm_lens_test.go
  - cmd/swarm_lens.go
  - cmd/swarm_repair_checkpoint_test.go
  - cmd/swarm_repair_checkpoint.go
  - cmd/testdata/classic-contract/v1/cases.json
  - cmd/testdata/classic-contract/v1/mechanisms.json
  - cmd/testdata/classic-contract/v1/schema.json
  - cmd/watch_dashboard_test.go
  - cmd/watch_dashboard.go
  - cmd/watch_idle_199_test.go
  - cmd/watch_live_test.go
  - cmd/watch_live.go
  - cmd/watch_replay_test.go
  - cmd/watch_replay.go
  - pkg/codex/dispatch.go
  - pkg/events/colony_live_test.go
  - pkg/events/colony_live.go
findings:
  critical: 2
  warning: 4
  info: 0
  total: 6
status: issues
---

# Phase 202: Code Review Report

**Reviewed:** 2026-09-11T00:00:00Z
**Depth:** standard
**Files Reviewed:** 71
**Status:** issues_found

## Summary

This phase adds a versioned live-event model (`pkg/events/colony_live.go`), a
single emission boundary and per-lane helpers (`cmd/live_events.go`), a pure
replay reducer (`cmd/live_projection.go`), the `aether watch` live/replay/idle
dashboard (`cmd/watch_live.go`, `cmd/watch_dashboard.go`, `cmd/watch_replay.go`),
Swarm's four-lens diagnosis and checkpointed repair
(`cmd/swarm_lens.go`, `cmd/swarm_episode.go`, `cmd/swarm_repair_checkpoint.go`),
Oracle depth presets and recommendation-first synthesis
(`cmd/oracle_preset.go`, `cmd/oracle_synthesis.go`, `cmd/oracle_live.go`), a
shared partial-work vocabulary (`cmd/partial_work_label.go`), and an episode
index (`cmd/episode_index.go`) surfaced through `history`/`status`.

The individual building blocks (the reducer, the standing vocabulary, the
episode index, Swarm's hypothesis comparison, Oracle's synthesis ordering) are
carefully written and mostly well tested. The two BLOCKER findings below are
both in the wiring between lanes and the `aether watch` live/replay
classification, and both were verified empirically against the real code
(not just read) — see each finding's reproduction. Both directly contradict
promises this same phase makes in `.aether/commands/oracle.yaml` and in
`renderColonyLiveDashboard`'s own doc comments: that a running Oracle round
and a running build/continue are visible live in `aether watch`.

## Critical Issues

### CR-01: Oracle's live episode can never satisfy `aether watch`'s "live" classification — an active research round always renders as a stale replay

**File:** `cmd/oracle_live.go` (whole file, esp. lines 50-121), `cmd/watch_live.go:44-73`, `cmd/live_projection.go:352-367,436`

**Issue:** `resolveWatchMode` (cmd/watch_live.go:44) only renders the live
cockpit when the replayed snapshot's `Open` field is true. `Open` is derived
in `foldColonyLiveEvents` (cmd/live_projection.go:436) purely from a
start/end balance across `live.episode.started`/`live.episode.ended` and
`live.wave.started`/`live.wave.ended` events (lines 352-367) — no other topic
touches it.

`cmd/oracle_live.go` never calls `emitColonyLiveEpisodeStarted` or
`emitColonyLiveEpisodeEnded` (grep confirms zero call sites for Oracle
anywhere in the repo), and it never emits `live.wave.started`/`ended` either
— every Oracle live event goes out as `live.worker.started`
(`emitOracleLiveRound`), `live.worker.progress`
(`emitOracleLiveRoundEnded`), `live.confidence.changed`,
`live.contradiction.found`, or `live.gap.targeted`. None of these topics
increment the open/close balance. The practical effect: for every Oracle run
that has ever existed or ever will, `snapshot.Open` is always `false`, so
`resolveWatchMode` always returns `watchModeReplay`, never `watchModeLive` —
even while Oracle is actively iterating.

This directly contradicts the phase's own documentation,
`.aether/commands/oracle.yaml`'s `research_shape.watchable_rounds`: *"A
running Oracle round is visible in the live colony dashboard (`aether
watch`) exactly like any other worker... as they are recorded"* — this
promise cannot be kept by the shipped code.

**Reproduction (verified by running a real test against the actual code,
not a hypothetical):**
```go
state := oracleStateFile{StartedAt: "2026-01-01T00:00:00Z", Iteration: 1,
    MaxIterations: 10, Phase: "investigate", OverallConfidence: 10, TargetConfidence: 95}
emitOracleLiveRound(state)
mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
// mode == watchModeReplay, snapshot.Open == false, snapshot.EpisodeID == "oracle-2026-01-01T00:00:00Z"
```
This reproduces even immediately after the very first live event of a brand
new, actively-running Oracle round.

Note `cmd/oracle_live_test.go`'s `TestLiveDashboardShowsTheResearchRound`
calls `resolveWatchMode` twice but discards the returned mode
(`_, snapshot := resolveWatchMode(...)`), so this gap is untested.

**Fix:** Wrap Oracle's run boundary the same way build/continue/plan do —
call `emitColonyLiveEpisodeStarted(oracleLiveEpisodeID(state),
events.EpisodeKindOracle)` once when a round-based run genuinely begins
(`oracle iterate`'s first round, or wherever the run's `Phase`/`Status`
first becomes active) and `emitColonyLiveEpisodeEnded(...)` when the run
stops (completion, manual stop, or process exit) — mirroring
`cmd/codex_continue.go:787-804`'s pattern. Alternatively, if Oracle's
single-repeated-worker-row model (documented in oracle_live.go) is meant to
stand in for an episode boundary, teach `foldColonyLiveEvents` to also raise
`openBalance` on an unfinished `live.worker.started` row for episode kind
`oracle` and lower it on `live.worker.finished`/a terminal `Status`. Either
way, this needs a test that actually asserts `mode == watchModeLive` while
an Oracle round is in flight (the existing test does not).

### CR-02: A recovery decision fired during an open build/continue episode is tagged with an unrelated synthetic episode ID, silently hijacking `aether watch`'s displayed episode away from the running work

**File:** `cmd/recovery_orchestrator.go:211-216`, `cmd/watch_live.go:168-185` (`latestLiveEpisodeID`)

**Issue:** `orchestrateRecovery`'s new emission funnel
(cmd/recovery_orchestrator.go:215) publishes every recovery decision under:
```go
emitColonyLiveRecoveryChanged(fmt.Sprintf("recovery-phase-%d", ctx.Phase), events.EpisodeKindRecovery, outcome.Action.Type)
```
— a synthetic episode ID unrelated to the real, currently-open build or
continue episode ID (`runHandle.Run.ID`, carried in-process via
`currentLiveBuildEpisode()`/`currentLiveContinueEpisode()`, cmd/live_events.go
lines 275-339). `orchestrateRecovery` is called from
`cmd/codex_build_finalize.go:1700`, `cmd/codex_continue_finalize.go:529`, and
`cmd/queen_wave_lifecycle.go:153` — all inside an active build or continue
run.

Separately, `resolveWatchMode` picks which episode to replay via
`latestLiveEpisodeID` (cmd/watch_live.go:172-185), which is deliberately
naive: it takes whichever episode ID the chronologically **last persisted
event** belongs to, with no concept of "the episode that is actually still
open." (Contrast this with `cmd/watch_replay.go`'s
`mostRecentlyStartedLiveEpisode`, which this same phase built specifically
to solve "an older episode can still be receiving events after a newer one
has already begun" for the *replay* path — the identical flaw was left
unfixed in the *live-selection* path.)

The combination: the moment a recovery decision fires mid-build/continue,
`aether watch` stops showing the real, still-running build/continue episode
(with its active workers) and instead shows the synthetic `recovery-phase-N`
episode — which carries zero workers and, per CR-01's same open/close-balance
logic, is never itself "open" either, so the dashboard falls straight into
replay mode showing almost nothing, while real workers are actively running
underneath.

**Reproduction (verified against real code):**
```go
emitColonyLiveEpisodeStarted("build-episode-123", events.EpisodeKindBuild)
emitColonyLiveWaveStarted("build-episode-123", events.EpisodeKindBuild, 1)
// ... 1+ second later, a worker fails and recovery runs ...
emitColonyLiveRecoveryChanged("recovery-phase-5", events.EpisodeKindRecovery, "retry")

mode, snapshot := resolveWatchMode(ctx, s, time.Now().UTC())
// mode == watchModeReplay, snapshot.EpisodeID == "recovery-phase-5",
// snapshot.EpisodeKind == "recovery", len(snapshot.Workers) == 0
// -- the still-open build episode ("build-episode-123") is no longer what `aether watch` shows.
```

**Fix:** Route recovery's live event onto the actual open episode instead of
inventing a new one — reuse `currentLiveBuildEpisode()` /
`currentLiveContinueEpisode()` (falling back to the `recovery-phase-N` form
only when neither is set, e.g. a code path with no live episode open at
all) so `RecoveryState` folds into the real running snapshot. Independently,
`resolveWatchMode`/`latestLiveEpisodeID` should select "the most recently
**started, still-open**" episode the same way
`mostRecentlyStartedLiveEpisode` already does for replay, not "whichever
episode owns the newest single event" — two genuinely concurrent episodes
(a real scenario this codebase already reasons about, per
`watch_replay.go`'s own comments) must not let a short-lived one steal the
live view from a long-running one.

## Warnings

### WR-01: Swarm's fix wave and verification wave (two of its three waves) never emit any live events — `aether watch` goes dark for the back two-thirds of every Swarm run

**File:** `cmd/swarm_cmd.go:452,468,1506-1512`

**Issue:** `executeSwarmWave`'s `liveEpisode bool` parameter is passed
`true` only for the investigation wave (`cmd/swarm_cmd.go:334`) and `false`
for the fix wave (line 452) and the verification wave (line 468). The
function's own doc comment (lines 1507-1511) states this is a known,
partial state: *"currently wired for the investigation wave only (202-02);
the fix and verification waves pass false and are unaffected (202-03 owns
wiring the rest of the lifecycle lanes)"* — but 202-03 is this same phase's
own live-lane-wiring task, and the fix/verification waves are still
unwired. Once the investigation wave's `WaveEnded` event fires, the
open/close balance returns to zero and stays there for the rest of the run
(no worker-started/finished events for the fix or verification waves at
all), so `aether watch` has nothing to show for roughly two-thirds of every
Swarm run's wall-clock duration. (`.aether/commands/swarm.yaml`'s own
promise is scoped correctly to wave 1's four lenses only, so this is not a
doc contradiction — just a materially incomplete feature.)

**Fix:** Pass `liveEpisode: true` for the fix and verification wave calls
too (`cmd/swarm_cmd.go:452,468`), or explicitly scope
`events.EpisodeKindSwarm`'s wave-level open/close events around each of the
three waves the way the investigation wave already does, so the dashboard
stays live for the whole run rather than just its first third.

### WR-02: Check name and Oracle gap text are recorded in the event payload but never surface anywhere in the live dashboard

**File:** `cmd/live_projection.go:352-417` (`foldColonyLiveEvents` switch), `cmd/watch_dashboard.go:70-76` (`colonyLiveTickerEntry`)

**Issue:** `emitColonyLiveCheckStarted`/`CheckPassed`/`CheckFailed`
(cmd/live_events.go:222-251) carry a `CheckName` and, for failures, a
`Findings` summary. `emitOracleLiveGapTargeted` (cmd/oracle_live.go:100-106)
carries the gap text in `Findings`. None of `LiveTopicCheckStarted`,
`LiveTopicCheckPassed`, `LiveTopicCheckFailed`, or `LiveTopicGapTargeted`
has a `case` in `foldColonyLiveEvents`'s switch (cmd/live_projection.go:352),
so `CheckName` and `Findings` are silently discarded — they never populate
any field on `colonyLiveSnapshot`. `colonyLiveTickerEntry`
(cmd/watch_dashboard.go:70-76) doesn't even have a `CheckName` or `Findings`
field to carry them into the ticker. The net result, confirmed against
`colonyLiveTickerDescription` (cmd/watch_dashboard.go:297-337): the live
dashboard's ticker literally prints "a check started", "a check passed", "a
check failed", and "a research gap was flagged" with no name and no detail
— for exactly the events whose entire purpose is telling the owner what is
happening. `grep -rn "CheckName" cmd/*.go` (excluding tests) shows the field
is set at three call sites and read at zero.

**Fix:** Add `CheckName` (and, for a failed check, its findings) to
`colonyLiveTickerEntry`, fold them in the ticker-append step, and give
`colonyLiveTickerDescription` real cases for `LiveTopicCheckStarted/Passed/Failed`
and `LiveTopicGapTargeted` that name the check/gap rather than falling back
to a generic sentence. If a persistent "current check" or "open gaps" field
on the snapshot itself is wanted (mirroring how worker rows persist), add
that too.

### WR-03: `codex.WorkerDispatch.ParentWorkerID` — the field this phase added specifically to carry worker lineage into the live dashboard — is never set by any producer

**File:** `pkg/codex/dispatch.go:49-57`, `cmd/live_events.go:151-166,202-218`, `cmd/watch_dashboard.go:198-200`

**Issue:** `ParentWorkerID` was added to `codex.WorkerDispatch` by this
phase (dispatch.go:49-57) explicitly so `emitColonyLiveWorkerStarted`/
`Finished` could read genuine worker lineage "rather than deriving lineage
from a rendered string." `watch_dashboard.go:198-200` renders a `lineage:
parent %s` line whenever a worker row carries a non-empty
`ParentWorkerID`. However, `grep -rn "ParentWorkerID:" cmd/*.go pkg/**/*.go`
(excluding tests and the field/consumer definitions themselves) shows
exactly zero production call sites construct a `codex.WorkerDispatch` with
`ParentWorkerID` set — build, continue, plan, and swarm all leave it at its
zero value. The only other `ParentWorkerID` write site in the repo
(`cmd/session_flow_cmds.go:1247`) is a field of the same name on a
different, pre-existing type (`colony.LifecycleWorkerLineage`), unrelated to
this one. The lineage-rendering machinery this phase built is therefore
unreachable dead capability: it will never fire in production because
nothing ever populates the field it reads.

**Fix:** Either wire a real producer (e.g. a worker spawned by another
worker, if that scenario exists anywhere in this codebase) to actually set
`dispatch.ParentWorkerID` when constructing the dispatch, or — if no such
producer exists yet — note in the doc comment that this is forward-looking
plumbing with no current caller, so a future reader doesn't assume it is
already exercised.

### WR-04: Swarm bypasses the per-lane live-event helpers and hand-constructs payloads with a bare `"swarm"` string literal at six call sites

**File:** `cmd/swarm_cmd.go:328-333,345-350,1534-1545,1619-1630`, `cmd/swarm_lens.go:642-652,661-670`

**Issue:** `cmd/live_events.go`'s own header comment states the contract
for this phase's live-event system: *"Every lifecycle lane... speaks on the
live stream through one of these thin, typed wrappers, never by
constructing an events.ColonyLivePayload and calling emitColonyLive directly
at its own call site."* Swarm does exactly the thing this sentence
prohibits, at six locations, constructing `events.ColonyLivePayload{...}`
literals directly and calling `emitColonyLive` itself rather than using (or
adding) per-lane helpers the way build/continue/plan/recovery/Oracle all do
elsewhere. Each of these literals also spells the episode kind as the bare
string `"swarm"` rather than the declared `events.EpisodeKindSwarm`
constant — harmless today only because the two happen to have the same
value, but a latent drift hazard the constant exists specifically to
prevent. `TestEveryLiveEventGoesThroughOneBoundary`
(cmd/live_projection_test.go) only verifies `emitColonyLive` is the sole
low-level `bus.Publish` boundary; it does not check that call sites route
through the documented per-lane helpers, so this inconsistency is currently
untested and unenforced.

**Fix:** Add `emitColonyLiveWaveStarted`/`WaveEnded`/`WorkerStarted`/
`FindingRecorded`/`ContradictionFound`-shaped helpers for Swarm's own
call sites (or reuse the existing generic ones, which already accept an
`episodeKind` parameter), and replace the six hand-built payload literals
with them, using `events.EpisodeKindSwarm` instead of the string literal.

---

_Reviewed: 2026-09-11T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
