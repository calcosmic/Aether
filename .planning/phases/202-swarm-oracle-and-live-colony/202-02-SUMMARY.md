---
phase: 202-swarm-oracle-and-live-colony
plan: "02"
subsystem: live-events
tags: [event-bus, swarm, watch, jsonl, replay, resume, live-colony]

# Dependency graph
requires:
  - phase: 202-01
    provides: "202-CLASSIC-SYNTHESIS.md's SYN-202-01 ruling (extend pkg/events.Bus with typed swarm.*/oracle.*/watch.* topics and a schema_version field, never a second bus) and the widened Classic contract corpus that accepts SYN-202 identifiers."
provides:
  - "pkg/events/colony_live.go: the one versioned typed live-colony event vocabulary (16 live.* topics, ColonyLiveSchemaVersion = live/v1, ColonyLivePayload) every later Phase 202 plan emits and reads through."
  - "cmd/live_events.go: emitColonyLive, the single emission boundary, structurally enforced by an AST guard so a later lane cannot grow its own live-event writer."
  - "cmd/live_projection.go: replayColonyLiveSnapshot / replayColonyLiveSnapshotResume, the pure reducer that turns persisted events into a live snapshot, with schema-version tolerance and resume-without-double-counting."
  - "cmd/watch_live.go: resolveWatchMode / renderLiveWatchVisual, the Go-side resolution of live vs. replay vs. idle watch modes."
  - "The one real emission path wired end to end: a Swarm investigation wave now emits wave-started/worker-started/worker-finished/wave-ended, and aether watch renders that wave live while it runs."
affects: [202-03, 202-04, 202-05, 202-06, 202-07, 202-08, 202-09, 202-10, 202-11, 202-12, 202-13, 202-14, 202-15]

# Actuals (#2632)
actuals:
  tokens: 15554
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Live-colony emission boundary: a single unexported function (emitColonyLive) is the only permitted caller of events.Bus.Publish for live.* topics, enforced by an AST guard (TestEveryLiveEventGoesThroughOneBoundary) that parses the cmd package's own syntax tree and reads its topic set from events.ColonyLiveTopics() rather than a maintained list."
    - "Lock-free read path for watch-mode resolution: cmd/live_projection.go reads event-bus.jsonl directly via os.ReadFile (readColonyLiveEventsRaw), bypassing storage.Store's locking read path (store.ReadJSONL / events.Bus.Query), mirroring cmd/lifecycle_facts.go's established discipline -- the locking path would create a lock-file side effect on every `aether watch` invocation, including the idle path, breaking its locked-in read-only guarantee (TestWatchIdle199ReadOnly)."
    - "Open/closed derived from a start/end boundary balance (episode.started/wave.started increment, episode.ended/wave.ended decrement), never from wall-clock inference -- elapsed time for an open episode is computed from its own recorded start event vs. its own last recorded event, so repeated replays of the same file are always byte-identical."
    - "Resume-without-double-counting: replayColonyLiveSnapshotResume folds forward from a (timestamp, sequence, event ID) cursor using the identical ordering the full-replay sort uses, seeding the fold's worker index and open/closed balance from the previously folded snapshot rather than starting over."
    - "Test-only schema-version override seam (colonyLiveSchemaVersionOverride) lets a test produce a genuine, boundary-emitted event carrying an unrecognized schema version, so the reducer's version-skip path is proven against real published events rather than a hand-typed JSON fixture."

key-files:
  created:
    - pkg/events/colony_live.go
    - pkg/events/colony_live_test.go
    - cmd/live_events.go
    - cmd/live_projection.go
    - cmd/live_projection_test.go
    - cmd/watch_live.go
    - cmd/watch_live_test.go
  modified:
    - cmd/compatibility_cmds.go
    - cmd/swarm_cmd.go
    - cmd/memory_feed_continue_test.go

key-decisions:
  - "executeSwarmWave gained a liveEpisode bool parameter rather than a second implementation for the investigation wave: the fix and verification waves pass false and are behaviorally untouched (no live events emitted), matching the plan's explicit scope fence that plan 202-03 owns wiring the rest of the lifecycle lanes, while the investigation wave call site passes true."
  - "The watch-mode read path (resolveWatchMode / replayColonyLiveSnapshot / latestLiveEpisodeID) deliberately bypasses events.Bus.Query in favor of a direct os.ReadFile, discovered as a Rule 1 bug during Task 1 verification: the naive Bus.Query path created a new per-path lock file on every plain `aether watch` invocation, failing the pre-existing TestWatchIdle199ReadOnly regression test."
  - "Open/closed state is a start/end boundary balance rather than a boolean flag toggled by the most recent boundary event seen, so an episode/wave that is later resumed (folded forward from a checkpoint) carries its balance forward correctly instead of resetting to closed."

requirements-completed: [LIVE-01, CEC-05]

coverage:
  - id: D1
    description: "One versioned typed live-event model (pkg/events/colony_live.go) exists, carries worker identity, caste, lineage, wave, workspace, question, confidence, contradictions, findings and recovery state as declared fields, and is the only model any renderer reads."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "pkg/events/colony_live_test.go#TestColonyLiveTopicsAreComplete"
        status: pass
      - kind: unit
        ref: "pkg/events/colony_live_test.go#TestColonyLivePayloadRoundTrip"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real Swarm investigation wave emits live events at its actual dispatch and completion sites, and aether watch renders that wave from those events while it is still running."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/watch_live_test.go#TestSwarmInvestigationWaveReachesTheLiveWatchScreen"
        status: pass
    human_judgment: false
  - id: D3
    description: "The dashboard is a pure projection of replayed events: driving the projection from the persisted event file alone, with no live process, reproduces the same snapshot field for field; an absent or empty event file yields an empty projection; equal-timestamp events replay in one stable, sequence-ordered position."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/watch_live_test.go#TestLiveProjectionIsPureReplay"
        status: pass
      - kind: unit
        ref: "cmd/watch_live_test.go#TestSwarmInvestigationWaveReachesTheLiveWatchScreen"
        status: pass
    human_judgment: false
  - id: D4
    description: "A failed live-event publish never aborts, fails or delays the work it describes, and every live event goes through exactly one emission boundary -- structurally enforced, not by review."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/live_projection_test.go#TestEveryLiveEventGoesThroughOneBoundary"
        status: pass
      - kind: unit
        ref: "cmd/watch_live_test.go#TestLiveProjectionIsPureReplay/nil_store_never_writes"
        status: pass
    human_judgment: false
  - id: D5
    description: "Sequence numbers assigned by the emission boundary are strictly increasing within one episode and independent across concurrent episodes."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/live_projection_test.go#TestLiveSequenceNumbersAreMonotonicPerEpisode"
        status: pass
    human_judgment: false
  - id: D6
    description: "The one live model carries a version; an unrecognized schema version is skipped with a named note and never aborts the surrounding replay; the model can be replayed and resumed across a simulated process restart without double-counting any event."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/live_projection_test.go#TestLiveModelVersionSkipsUnknownWithoutAborting"
        status: pass
      - kind: unit
        ref: "cmd/live_projection_test.go#TestLiveProjectionResumesWithoutDoubleCounting"
        status: pass
    human_judgment: false
  - id: D7
    description: "No live event payload carries a currency amount; the cockpit reads reported cost from the existing spend ledger authority. The honest idle watch floor (buildIdleWatchResult / renderIdleWatchVisual, including its read-only guarantee) is preserved unchanged as the third branch."
    requirement: "LIVE-01"
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
    human_judgment: false

duration: 9min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 02: One Live Event, Emitted, Replayed, Watched Summary

**Built the one versioned typed live-colony event model (`live/v1`), its single emission boundary, its pure replay/resume reducer, and wired a real Swarm investigation wave into `aether watch`'s live screen — proven end to end against a persisted-file-only replay, with the idle floor's read-only guarantee preserved.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-09-11T10:40:55+02:00
- **Completed:** 2026-09-11T10:49:43+02:00
- **Tasks:** 3
- **Files modified:** 10 (7 created, 3 modified)

## Accomplishments

- `pkg/events/colony_live.go`: 16 `live.*` topic constants (episode/wave/worker lifecycle, question, confidence, contradiction, finding, signal, check, recovery), `ColonyLiveSchemaVersion = "live/v1"`, and `ColonyLivePayload` — a single typed struct (never a bare map) with every declared field the plan required, and no field that could ever carry a currency amount.
- `cmd/live_events.go`: `emitColonyLive`, the one emission boundary. A nil store, empty topic, marshal failure, or bus publish failure all return silently — emission can never change the outcome of the work it describes. Per-episode monotonic sequence numbers are assigned via a mutex-guarded map of atomic counters, proven independent across concurrent episodes under `-race`.
- `cmd/live_projection.go`: `replayColonyLiveSnapshot`, a pure reducer folding persisted events into a `colonyLiveSnapshot` — sorted strictly by timestamp, then sequence, then event ID, so equal (second-precision) timestamps always resolve to one stable order. Extended in Task 3 with schema-version tolerance (an unrecognized version is skipped with a named note, never fatal) and `replayColonyLiveSnapshotResume`, which folds forward from a previously folded snapshot's cursor without double-counting — proven equal, field for field, to a full replay from the beginning.
- `cmd/watch_live.go`: `resolveWatchMode` (live/replay/idle, resolved in Go from persisted evidence) and `renderLiveWatchVisual`. The idle branch (`buildIdleWatchResult`/`renderIdleWatchVisual`) is untouched; the replay branch falls through to it for now, as plan 202-09 owns rendering a closed episode.
- Wired the one real emission path: `cmd/swarm_cmd.go`'s `executeSwarmWave` gained a `liveEpisode bool` parameter — the investigation wave call site passes `true` (wave-started/worker-started/worker-finished/wave-ended all emit through `emitColonyLive`); the fix and verification wave call sites pass `false` and are behaviorally unaffected, matching the plan's scope fence.
- Proved the whole path end to end in `cmd/watch_live_test.go#TestSwarmInvestigationWaveReachesTheLiveWatchScreen`: a real Swarm investigation wave (`buildSwarmInvestigationPlans` + `executeSwarmWave`) against a stub invoker reaches the live watch screen — naming the dispatched worker(s), caste, and wave — while the wave is still open, and a completely fresh `storage.Store` handle, reading nothing but the persisted event file, reproduces the identical snapshot after the wave closes.
- `cmd/live_projection_test.go#TestEveryLiveEventGoesThroughOneBoundary`: an AST guard parsing the `cmd` package's own syntax tree, reading its live-topic set from `events.ColonyLiveTopics()` (never a hardcoded list), that fails by function name and file position when a synthetic fixture function publishes a live topic directly — proven able to fail, not just pass vacuously.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "one live event, emitted, replayed, watched"** — `18b3e595` (feat, tracer)
2. **Task 2: Make the emission boundary singular and refuse a second one** — `881e48b8` (test)
3. **Task 3: Version the live model and prove replay and resume across a restart** — `2bccc3d5` (feat)

**Plan metadata:** (this commit)

_Note: Task 2 carried `tdd="true"`, but the emission boundary was already correct from Task 1's implementation, so this landed as a locked-in structural regression test rather than a RED-then-GREEN implementation change — the same pattern 202-01's Task 2 documented for a static-fixture contract change._

## Files Created/Modified

- `pkg/events/colony_live.go` — The one versioned live-colony event vocabulary and typed payload (`ColonyLivePayload`, `ColonyLiveTopics()`).
- `pkg/events/colony_live_test.go` — Topic-completeness and payload round-trip tests.
- `cmd/live_events.go` — `emitColonyLive`, the single emission boundary; per-episode monotonic sequence counters; the `colonyLiveSchemaVersionOverride` test-only seam.
- `cmd/live_projection.go` — `colonyLiveSnapshot`, `replayColonyLiveSnapshot`, `replayColonyLiveSnapshotResume`, the lock-free `readColonyLiveEventsRaw` reader, and the fold/sort reducer internals.
- `cmd/live_projection_test.go` — The AST emission-boundary guard, the sequence-monotonicity race test, the schema-version-skip test, and the resume-without-double-counting test.
- `cmd/watch_live.go` — `resolveWatchMode`, `renderLiveWatchVisual`, `liveWatchResult`.
- `cmd/watch_live_test.go` — The Swarm-investigation-wave end-to-end test and the reducer purity test.
- `cmd/compatibility_cmds.go` — `watchCmd`'s `RunE` now resolves live/replay/idle via `resolveWatchMode` before falling back to the idle floor.
- `cmd/swarm_cmd.go` — `executeSwarmWave` gained a `liveEpisode bool` parameter; `runSwarmDestroy`'s investigation-wave call site wraps it in wave-started/wave-ended emission; added `swarmPlansWaveNumber`.
- `cmd/memory_feed_continue_test.go` — Updated two pre-existing `executeSwarmWave` call sites for the new parameter (mechanical, no behavior change).

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Watch-mode resolution bypassed `events.Bus.Query` in favor of a lock-free direct file read**
- **Found during:** Task 1, self-check verification run
- **Issue:** The initial implementation of `resolveWatchMode`/`replayColonyLiveSnapshot`/`latestLiveEpisodeID` read persisted events via `events.Bus.Query`, which acquires `storage.Store`'s advisory read lock — creating a new digest-named `.lock` file under `.aether/data/locks/` the first time `event-bus.jsonl` was ever read. Since `resolveWatchMode` now runs at the top of every `aether watch` invocation, including the plain idle path, this broke the pre-existing, locked-in `TestWatchIdle199ReadOnly` regression test (`idle watch mutated the workspace`).
- **Fix:** Added `readColonyLiveEventsRaw`, reading `event-bus.jsonl` directly via `os.ReadFile` and parsing JSONL manually — bypassing `storage.Store`'s locking read path entirely, mirroring the established discipline `cmd/lifecycle_facts.go`'s `readLifecycleJSON`/`readLifecycleActors` already use for exactly the same reason. `replayColonyLiveSnapshot` and `latestLiveEpisodeID` were updated to use it.
- **Files modified:** `cmd/live_projection.go`, `cmd/watch_live.go`
- **Verification:** `go test ./cmd -run '^(TestWatchIdle199StatusFallback|TestWatchIdle199NoFakeLiveness|TestWatchIdle199ReadOnly)$' -count=1` passes; the full Task 1 `<verify>` command passes.
- **Committed in:** `18b3e595` (Task 1 commit — found and fixed before the task was committed, so the commit contains the corrected implementation directly)

**2. [Rule 3 - Blocking] Updated two pre-existing `executeSwarmWave` call sites for the new `liveEpisode` parameter**
- **Found during:** Task 1, after adding the `liveEpisode bool` parameter to `executeSwarmWave`
- **Issue:** `cmd/memory_feed_continue_test.go` contains two call sites (`TestSwarmWorkerFailureReachesTheFailureLogOnBothLanes`'s native lane, and a second self-reporting-invoker table test) that call `executeSwarmWave` directly with the pre-Task-1 five-argument signature. The new sixth parameter was not named in the plan's action text but is required for the package to compile.
- **Fix:** Added `false` as the sixth argument at both call sites — mechanical, no behavior change (neither test exercises live emission).
- **Files modified:** `cmd/memory_feed_continue_test.go`
- **Verification:** `go build ./cmd` and `go test ./cmd -run 'Swarm'` both pass with no regressions.
- **Committed in:** `18b3e595` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 bug, 1 Rule 3 blocking compile fix).
**Impact on plan:** The Rule 1 fix was necessary to preserve an existing locked-in read-only guarantee that the plan's own acceptance criteria requires to keep passing (`TestWatchIdle199ReadOnly` is not named in the plan's `<verify>` line, but breaking it would be a real regression to a Phase 199 invariant). The Rule 3 fix was a required compile fix with zero behavioral effect. No scope creep.

## Issues Encountered

None beyond the deviation documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The one versioned live model, its single emission boundary, and its pure replay/resume reducer are in place for every remaining Phase 202 plan to emit and read through — no later plan needs to (and per `cmd/live_projection_test.go`'s AST guard, cannot silently) build a second one.
- `resolveWatchMode`'s `replay` branch currently falls through to the honest idle floor; plan 202-09 owns rendering a closed episode from its persisted events.
- The fix and verification Swarm waves, and every other lifecycle lane (build, continue, plan, oracle, recovery), still emit no live events; plan 202-03 owns wiring the rest.
- No blockers.

## Self-Check: PASSED

- `pkg/events/colony_live.go` — FOUND
- `pkg/events/colony_live_test.go` — FOUND
- `cmd/live_events.go` — FOUND
- `cmd/live_projection.go` — FOUND
- `cmd/live_projection_test.go` — FOUND
- `cmd/watch_live.go` — FOUND
- `cmd/watch_live_test.go` — FOUND
- Commit `18b3e595` — FOUND in `git log --oneline --all`
- Commit `881e48b8` — FOUND in `git log --oneline --all`
- Commit `2bccc3d5` — FOUND in `git log --oneline --all`
- `go test ./pkg/events ./cmd -run '^(TestColonyLiveTopicsAreComplete|TestColonyLivePayloadRoundTrip|TestSwarmInvestigationWaveReachesTheLiveWatchScreen|TestLiveProjectionIsPureReplay|TestEveryLiveEventGoesThroughOneBoundary|TestLiveSequenceNumbersAreMonotonicPerEpisode|TestLiveModelVersionSkipsUnknownWithoutAborting|TestLiveProjectionResumesWithoutDoubleCounting|TestWatchIdle199StatusFallback|TestWatchIdle199NoFakeLiveness|TestWatchIdle199ReadOnly)$' -count=1 -race` — PASS
- `go vet ./cmd ./pkg/events` — clean

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
