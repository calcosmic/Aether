---
phase: 202-swarm-oracle-and-live-colony
plan: "06"
subsystem: live-events
tags: [live-colony, cockpit, watch, ticker, refresh-loop, spawn-runs]

# Dependency graph
requires:
  - phase: 202-02
    provides: "pkg/events/colony_live.go's one versioned live-colony event model, cmd/live_events.go's emitColonyLive single emission boundary, and cmd/live_projection.go's replayColonyLiveSnapshot pure reducer this plan renders from."
  - phase: 202-03
    provides: "Real live-event emission wired at every lifecycle lane (build, continue, plan, swarm, recovery), so the dashboard has real events to render, not just the Swarm tracer."
provides:
  - "cmd/watch_dashboard.go: renderColonyLiveDashboard, the D-01 live cockpit -- one compact colony header line, the current wave's workers rendered in depth, every other wave compressed to a counted line, confidence/contradictions/signals/recovery state omitted when absent, and the reported-cost block (read verbatim from renderSpendCostLineFromLedgers) last. Worker-identity and wave drill-down via colonyLiveDrillSelector."
  - "cmd/watch_dashboard.go: renderColonyLiveTicker, the bounded (colonyLiveTickerLimit) event ticker strip, one plain-English line per entry with a caste glyph when the event names one, built from the same replay the dashboard already has -- no second query."
  - "cmd/watch_live.go: runColonyLiveRefreshLoop / runWatchCommand -- watchCmd now redraws in place on --interval when connected to a real terminal, renders exactly one frame with --once or when not a TTY, hides/restores the cursor, and stops cleanly on interrupt via signal.NotifyContext."
  - "cmd/watch_live.go: applyUnfinishedWorkerInterruption -- a worker with no worker.finished event of its own is reclassified as interrupted (never completed) once the spawn run that dispatched it reaches a terminal status, read lock-free from spawn-runs.json."
  - "colonyLiveWorkerRow gains StartedAt/Finished/InterruptedReason; colonyLiveSnapshot gains Signals folded from live.signal.consulted; colonyLiveTickerEntry gains Caste."
affects: [202-09, 202-15]

# Actuals (#2632)
actuals:
  tokens: 13520
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Lock-free durable-record read for a watch-path decision: latestSpawnRunRaw reads spawn-runs.json via os.ReadFile directly (mirroring readColonyLiveEventsRaw's own established discipline from 202-02), never storage.Store.ReadFile/agent.SpawnTree.CurrentRun(), because the latter takes an RLock and creates a lock file on every `aether watch` invocation -- breaking the command's read-only guarantee."
    - "Loop-vs-single-frame gated on isTerminalWriter, not shouldRenderVisualOutput: the latter is also true for a forced/piped/test-forced 'pretty' render that wants exactly one frame back. Gating the redraw loop on genuine TTY-ness is what lets `AETHER_OUTPUT_MODE=visual` scripts and tests keep returning immediately while a real terminal still gets the live redraw."
    - "Open/unfinished-worker distinction carried as a dedicated Finished bool on the worker row, set only by the worker's own worker.finished event -- Status alone is not enough to know whether a row is done, since worker.progress/question/finding updates also write Status without ever finishing the worker."

key-files:
  created:
    - cmd/watch_dashboard.go
    - cmd/watch_dashboard_test.go
  modified:
    - cmd/live_projection.go
    - cmd/watch_live.go
    - cmd/compatibility_cmds.go

key-decisions:
  - "The unfinished-worker rule reads the durable spawn-runs.json record (agent.SpawnRun.Status via agent.IsTerminalSpawnStatus), not a live event -- there is no live.* topic for 'the run itself ended', so this is the one place the cockpit legitimately reads a second durable record alongside the replayed event stream, and it does so lock-free to preserve watch's read-only guarantee."
  - "The redraw loop is gated on isTerminalWriter(stdout), not shouldRenderVisualOutput(stdout) -- discovered as a Rule 1 bug during verification: gating on the visual-mode flag alone made a pre-existing test (TestWatchVisualOutputShowsHonestIdleFallback, which forces AETHER_OUTPUT_MODE=visual and calls `watch` with no --once) hang until the test binary's timeout killed it, because the command would loop forever waiting for a signal that never comes in a test process."
  - "watchCmd's Args changed from cobra.NoArgs to cobra.MaximumNArgs(1) to carry the drill-down selector (a worker identity or a bare wave number); Cobra derives the command name from the first word of Use, so every name-based reference to the `watch` command (audit catalogs, parity manifests, documented-command-name checks) is unaffected."

patterns-established:
  - "Dashboard rendering functions (renderColonyLiveDashboard, renderColonyLiveTicker) take only already-replayed/already-loaded values (colonyLiveSnapshot, colony.ColonyState, []spendLedger) -- never a *storage.Store -- so a render call is structurally unable to perform a second read, which is what TestLiveTickerAndDashboardShareOneReplay proves via colonyLiveRawReadCalls rather than assuming."

requirements-completed: [LIVE-02, CEC-05]

coverage:
  - id: D1
    description: "renderColonyLiveDashboard renders the current wave's workers in depth (identity, caste, lineage, workspace, lens/question, findings), compresses every other wave to a counted line, and the header stays exactly one line naming project/phase/standing/episode."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardShowsCurrentWaveInDepth"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardHeaderIsOneCompactLine"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardOmitsFieldsTheSnapshotLacks"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardRendersDeterministically"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardDrillDown"
        status: pass
    human_judgment: false
  - id: D2
    description: "Reported cost is the last block on the dashboard and equals renderSpendCostLineFromLedgers for the exact ledgers passed in -- never recomputed on screen."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardCostComesFromTheLedger"
        status: pass
    human_judgment: false
  - id: D3
    description: "The bounded event ticker strip renders newest-last with no filler, shares the dashboard's one replay (no second read), and every declared live.* topic renders as a plain-English line."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveTickerIsBoundedAndNewestLast"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveTickerAndDashboardShareOneReplay"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveTickerLinesArePlainEnglish"
        status: pass
    human_judgment: false
  - id: D4
    description: "`aether watch --once` renders exactly one frame; on a real terminal without --once it redraws in place on the configured interval, never mutates the colony data directory, and stops cleanly (cursor restored) on interrupt."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestWatchSingleSnapshotRendersOnce"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestWatchRefreshesInPlaceWithoutMutating"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestWatchRefreshStopsCleanlyOnInterrupt"
        status: pass
    human_judgment: false
  - id: D5
    description: "A worker with a start event and no terminal event of its own renders as running with elapsed time from its own start event; once the spawn run that owned it reaches a terminal status it renders as interrupted, naming that status, and is never shown as completed."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestUnfinishedWorkerShowsAsRunningThenInterrupted"
        status: pass
    human_judgment: false
  - id: D6
    description: "The Classic colony character (caste emoji/ant glyphs, banner/stage-marker house style) carries through the dashboard, read from the shared caste map rather than a local copy."
    requirement: "LIVE-02"
    verification:
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestLiveDashboardShowsCurrentWaveInDepth (caste-emoji-map-mutation subtest)"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 06: The Live Colony Cockpit Summary

**One in-place-refreshing `aether watch` dashboard built purely from the replayed live-event snapshot plus the reported-cost ledger, with a bounded plain-English event ticker and a durable-record-backed rule that never calls an unfinished worker "completed."**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-09-11 (Task 1 reconnaissance)
- **Completed:** 2026-09-11T13:23:46+02:00
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `renderColonyLiveDashboard` (`cmd/watch_dashboard.go`): a one-line colony header (project/phase/standing/episode); a stage marker naming the focused wave or worker; the current wave's workers rendered in full (caste identity via the shared `casteIdentity`/`casteEmoji` functions, lineage, workspace, lens/question, findings, and -- for a still-open worker -- elapsed time computed from its own start event against the snapshot's own last replayed timestamp); every other wave compressed to a counted line; confidence/contradictions/signals/recovery state each omitted entirely when the snapshot doesn't carry them; and the reported-cost block, read verbatim from `renderSpendCostLineFromLedgers`, always last.
- Worker-identity and wave drill-down (`colonyLiveDrillSelector`, parsed from `watch`'s own optional positional argument) that renders one selection in full while the header line never changes.
- `renderColonyLiveTicker`: the bounded (`colonyLiveTickerLimit`) most-recent-events strip, newest last, one plain-English line per entry (timestamp, caste glyph when the event names a caste, and a description of the transition -- never a raw `live.*` topic string), built from the exact same replay the rest of the dashboard already has.
- `runColonyLiveRefreshLoop` / `runWatchCommand` (`cmd/watch_live.go`): `aether watch` now actually honours its long-accepted `--once`/`--interval` flags. On a genuine terminal without `--once` it redraws in place (hide cursor, clear-and-redraw each tick, restore cursor on any exit path including interrupt); everywhere else -- `--once`, a pipe, a forced-visual test/script -- it renders exactly one frame and returns, which is also what fixed a real hang this change first introduced (see Deviations).
- `applyUnfinishedWorkerInterruption`: a worker with a `worker.started` event and no `worker.finished` event of its own stays "active" (with elapsed time) until the spawn run that dispatched it (read lock-free from `spawn-runs.json`, mirroring the 202-02 event-bus read discipline) reaches a terminal status, at which point it renders "interrupted" naming that run's own status -- never "completed," which only the worker's own terminal event can say.
- `colonyLiveWorkerRow` gained `StartedAt`/`Finished`/`InterruptedReason`; `colonyLiveSnapshot` gained `Signals` (folded from `live.signal.consulted`); `colonyLiveTickerEntry` gained `Caste`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Render the dashboard from the replayed snapshot, focused on the current wave** - `1734c54e` (feat)
2. **Task 2: Add the event ticker strip from the same stream** - `8358c387` (feat)
3. **Task 3: Refresh in place, and never call an unfinished worker finished** - `84651052` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/watch_dashboard.go` - `renderColonyLiveDashboard`, its header/worker-block/drill-down helpers, and `renderColonyLiveTicker`.
- `cmd/watch_dashboard_test.go` - 13 tests covering the dashboard, ticker, refresh loop, and unfinished-worker rule.
- `cmd/live_projection.go` - Worker-row and snapshot fields the dashboard/ticker read (`StartedAt`, `Finished`, `InterruptedReason`, `Signals`, ticker `Caste`), plus a test-only read-call counter.
- `cmd/watch_live.go` - `applyUnfinishedWorkerInterruption`, the lock-free `spawn-runs.json` reader, the refresh loop, and `runWatchCommand`.
- `cmd/compatibility_cmds.go` - `watchCmd` now accepts one optional positional argument and delegates to `runWatchCommand`.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Gated the redraw loop on real TTY-ness, not visual-output mode**
- **Found during:** Task 3 verification (a pre-existing regression test, `TestWatchVisualOutputShowsHonestIdleFallback`, which forces `AETHER_OUTPUT_MODE=visual` and calls `watch` with no `--once`)
- **Issue:** The initial implementation entered the redraw loop whenever `shouldRenderVisualOutput(stdout)` was true and `--once` was absent. `shouldRenderVisualOutput` is also true for a forced/piped "pretty" render that wants exactly one frame back (any script or test setting `AETHER_OUTPUT_MODE=visual`/`AETHER_FORCE_VISUAL`), so that pre-existing test hung until the test binary's own timeout killed the whole package's test run -- caught by running the broader `Watch|Live|Spawn|Ticker|Dashboard` regex, not the plan's own narrower `<verify>` commands, which never exercised this path.
- **Fix:** Gated the loop on `isTerminalWriter(stdout)` (a genuine `*os.File` connected to a character device) instead. A forced-visual test/script now gets one frame and returns immediately, exactly as before this plan; only a real interactive terminal enters the redraw loop.
- **Files modified:** `cmd/watch_live.go`
- **Verification:** `go test ./cmd -run '^TestWatchVisualOutputShowsHonestIdleFallback$' -v -count=1 -timeout 60s` passes; the plan's own `<verify>` regexes for all three tasks re-ran clean afterward; `go build ./...` and `go vet ./cmd` clean.
- **Committed in:** `84651052` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - a real hang bug caught during verification, not by the plan's own narrower `<verify>` commands).
**Impact on plan:** Necessary for correctness -- without this fix, `aether watch` (and any script or CI check that pipes/forces its visual output) would hang indefinitely instead of returning. No scope creep; the fix is contained to the loop-entry gate.

## Issues Encountered

- Ran the broader regex `go test ./cmd -run 'Watch|Live|Spawn|Ticker|Dashboard' -count=1` (beyond the plan's own scoped `<verify>` commands) specifically to catch cross-file regressions from changing `watchCmd`'s long-dormant `--once`/`--interval` flags from accepted-but-ignored to actually driving behavior; this is what surfaced the hang above. Also discovered, and confirmed pre-existing via a throwaway worktree at the phase's base commit (`3ad08ade`), an unrelated failure in `TestDocumentedCommandNamesResolve` (three `.md`/`.yaml` sources reference `aether help $ARGUMENTS`, which is not a registered subcommand) -- out of scope for this plan (never touched `help` command registration), not auto-fixed, left as-is.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 202-09 (the replay-backed idle/closed-episode view) can now reuse `renderColonyLiveDashboard`'s header/worker-block machinery for a closed episode's own detail, and the projection already carries everything it needs (`StartedAt`/`Finished` distinguish "was still running when the episode closed" from "finished cleanly").
- 202-15 depends on this plan plus five others (202-07, 202-09, 202-11, 202-12, 202-14) and is not yet unblocked.
- No blockers.

## Self-Check: PASSED

- `cmd/watch_dashboard.go` — FOUND
- `cmd/watch_dashboard_test.go` — FOUND
- `cmd/live_projection.go`, `cmd/watch_live.go`, `cmd/compatibility_cmds.go` — modified, confirmed via `git show --stat`
- Commit `1734c54e` — FOUND in `git log --oneline`
- Commit `8358c387` — FOUND in `git log --oneline`
- Commit `84651052` — FOUND in `git log --oneline`
- `go build ./...` — PASS
- `go vet ./cmd` — PASS
- All three plan `<verify>` regex commands — PASS
- `go test ./cmd -run 'Watch|Live|Spawn|Ticker|Dashboard' -count=1` (full, unscoped) — PASS (after the Task 3 fix; initially caught the hang documented above)

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
