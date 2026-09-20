---
phase: 202-swarm-oracle-and-live-colony
plan: "17"
subsystem: live-colony
tags: [go, live-events, watch, oracle, gap-closure]

# Dependency graph
requires:
  - phase: 202-swarm-oracle-and-live-colony
    provides: "The typed live-colony event model (pkg/events/colony_live.go), the emission boundary and per-lane helpers (cmd/live_events.go), the pure replay reducer (cmd/live_projection.go), and plan 202-16's shared open/close boundary rule and episode-selection fix (colonyLiveBoundaryDelta, latestStartedLiveEpisodeAmong, currentLiveRecoveryEpisode)"
provides:
  - "openOracleLiveEpisode / oracleLiveTerminalStatus (cmd/oracle_live.go): Oracle's own episode open/close pair, deriving the closing status from the loop's own reported outcome"
  - "runOracleLoop now wraps the round-based run (renamed runOracleLoopRounds) in that episode boundary; stopOracleCompatibility closes it for a controller it just killed"
  - "colonyLiveEpisodeAbandoned (cmd/watch_live.go): episode-kind-aware abandonment dispatch, delegating to oracleLiveEpisodeAbandoned for Oracle episodes and preserving the existing spawn-run rule verbatim for every other lane"
  - "A round genuinely in flight resolves watchModeLive even in a colony where an earlier build has already finished; a dead controller, a superseded run, or a manual/automatic stop all resolve it back to replay"
affects: [watch, aether-watch-cockpit, oracle]

# Actuals (#2632)
actuals:
  tokens: 9843
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Episode-kind-aware dispatch inside a single decision function (colonyLiveEpisodeAbandoned), rather than a lane-specific special case scattered through resolveWatchMode"
    - "An AST-walk wiring guard (TestOracleEpisodeBoundaryIsWiredIntoTheLoop) proves a boundary function is genuinely called from its intended call site, not merely defined"
    - "Break-it-to-prove-it regression discipline carried across a two-task interlocking fix: the first task's own test can only pass once the second task's dispatch fix also lands, and that dependency was verified directly rather than assumed"

key-files:
  created: []
  modified:
    - cmd/oracle_live.go
    - cmd/oracle_loop.go
    - cmd/oracle_live_test.go
    - cmd/watch_live.go
    - cmd/watch_live_test.go
    - CLAUDE.md

key-decisions:
  - "The episode-open guard for a never-started run was applied: runOracleLoop only opens a boundary when the loaded state's StartedAt is non-empty, so a run refused before it starts (no dispatcher, invoker unavailable, agent validation failure) records no phantom episode."
  - "oracleLiveEpisodeAbandoned treats a durable state whose oracleLiveEpisodeID no longer matches the open episode's ID as abandoned -- covering the case where a fresh oracle run overwrote state.json without the prior episode's own close ever being emitted."
  - "colonyLiveEpisodeAbandoned dispatches purely on snapshot.EpisodeKind; every non-Oracle kind delegates to the pre-existing colonyLiveEpisodeRunHasTerminated(s) call, byte-for-byte, so build/continue/plan/Swarm behavior is provably unaffected (TestWatchResolvesThreeBranchesFromEvidenceAlone unchanged)."

requirements-completed: [CEC-05, LIVE-06]

coverage:
  - id: D1
    description: "A round-based Oracle run opens and closes its own live episode boundary, so a genuinely iterating round resolves watchModeLive, and the boundary is provably called by the research loop rather than merely defined."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleRoundIsLiveWhileItRuns"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleManualStopClosesTheLiveEpisode"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleEpisodeBoundaryIsWiredIntoTheLoop"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestLiveDashboardShowsTheResearchRound"
        status: pass
    human_judgment: false
  - id: D2
    description: "An open Oracle episode's abandonment is judged by Oracle's own durable state and controller process, never by another lane's spawn-run record -- so a finished earlier build never wrongly demotes a genuinely live Oracle round, and a dead controller or superseded run is honestly reported as no longer live."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/watch_live_test.go#TestAbandonedOracleRoundIsNotLive"
        status: pass
      - kind: unit
        ref: "cmd/watch_replay_test.go#TestWatchResolvesThreeBranchesFromEvidenceAlone"
        status: pass
      - kind: unit
        ref: "cmd/watch_dashboard_test.go#TestWatchIsReadOnlyInEveryBranch"
        status: pass
    human_judgment: false
  - id: D3
    description: "CLAUDE.md's Live Colony section names the tests proving both this plan's Oracle-liveness fix and plan 202-16's recovery-continuity fix, satisfying the removal-proof CLAUDE.md guard."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/claudemd_live_colony_test.go#TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest"
        status: pass
    human_judgment: false

# Metrics
duration: 40min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 17: Gap Closure -- Oracle's Research Round Now Registers as Live Summary

**A round-based Oracle run opens and closes its own live-colony episode boundary, and that episode's abandonment is judged by Oracle's own durable state and controller process instead of another lane's spawn-run record -- closing CR-01 end to end.**

## Performance

- **Duration:** 40 min (approximate)
- **Completed:** 2026-09-11T22:20:55+02:00
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- `openOracleLiveEpisode` / `oracleLiveTerminalStatus` (`cmd/oracle_live.go`) give Oracle the same emit-started / defer-emit-ended boundary pair `cmd/codex_build.go` and `cmd/codex_continue.go` already use on their own lanes. The episode ID reuses `oracleLiveEpisodeID(state)` (derived from `state.StartedAt`), so a resumed `oracle iterate` rejoins the same episode instead of minting a new one.
- `runOracleLoop` (`cmd/oracle_loop.go`) now wraps the renamed `runOracleLoopRounds` in that boundary -- opened only when the loaded state already has a non-empty `StartedAt`, so a run that never starts (no dispatcher, invoker unavailable, agent validation failure) records no phantom episode. `stopOracleCompatibility` captures the episode ID *before* its existing `StartedAt` backfill and closes the episode for a controller it just killed, so a manually stopped run never stays classified live.
- `colonyLiveEpisodeAbandoned` (`cmd/watch_live.go`) replaces the direct `colonyLiveEpisodeRunHasTerminated(s)` call inside `resolveWatchMode` with an episode-kind-aware dispatch: every non-Oracle kind keeps the exact same rule (verified unchanged by `TestWatchResolvesThreeBranchesFromEvidenceAlone`); an Oracle episode is judged by `oracleLiveEpisodeAbandoned`, which reads Oracle's own state file and checks its controller PID via `oracleStateHasStaleController` -- so a finished earlier build's spawn run in the same colony can no longer wrongly demote a genuinely live Oracle round.
- CLAUDE.md's Live Colony section now names the tests proving both this plan's Oracle-liveness fix and 202-16's recovery-continuity fix (`TestOracleRoundIsLiveWhileItRuns`, `TestOracleEpisodeBoundaryIsWiredIntoTheLoop`, `TestAbandonedOracleRoundIsNotLive`, `TestRecoveryDecisionKeepsTheBuildEpisodeLive`, `TestWatchFollowsTheMostRecentlyStartedOpenEpisode`), passing `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest`.

## Task Commits

Each task was committed atomically:

1. **Task 1: A round-based Oracle run opens and closes its own live episode** - `019403ef` (fix)
2. **Task 2: An abandoned Oracle round is judged by Oracle's own evidence, never by another lane's run record** - `7c871dbb` (fix)
3. **Task 3: The shipped claims about a live Oracle and layered recovery name the tests that prove them** - `40579408` (docs)

## Files Created/Modified

- `cmd/oracle_live.go` - Added `openOracleLiveEpisode` (episode open/close pair) and `oracleLiveTerminalStatus` (closing status derived from the loop's own outcome, never invented)
- `cmd/oracle_loop.go` - Renamed the round loop body to `runOracleLoopRounds`; added a new `runOracleLoop` wrapper opening/closing the live episode around it; `stopOracleCompatibility` now captures the episode ID before the `StartedAt` backfill and emits the closing event for a controller it just killed
- `cmd/oracle_live_test.go` - Added `TestOracleRoundIsLiveWhileItRuns`, `TestOracleManualStopClosesTheLiveEpisode`, `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` (AST wiring guard); amended `TestLiveDashboardShowsTheResearchRound` to open the production episode boundary and assert the resolved mode instead of discarding it
- `cmd/watch_live.go` - Added `colonyLiveEpisodeAbandoned` (episode-kind-aware dispatch) and `oracleLiveEpisodeAbandoned` (Oracle's own state/controller evidence); `resolveWatchMode` now calls the dispatcher instead of the spawn-run rule directly
- `cmd/watch_live_test.go` - Added `TestAbandonedOracleRoundIsNotLive` (6 subtests) and the `oracleDeadControllerPID` helper, which derives a definitely-dead controller PID from a real started-and-reaped process rather than a typed-in literal
- `CLAUDE.md` - Added two claims to the Live Colony section (recovery continuity, live research rounds), each naming the tests that prove it; extended the "for dummies" closing paragraph with one sentence covering both

## Decisions Made

- **Episode-open guard for never-started runs: applied.** `runOracleLoop` only opens the boundary when the independently-loaded state already carries a non-empty `StartedAt`. A run refused before it starts (dispatcher unavailable, agent validation failure, unreadable state) opens and closes nothing.
- `oracleLiveEpisodeAbandoned` treats a durable state whose `oracleLiveEpisodeID` no longer matches the open episode's own ID as abandoned -- this covers a fresh `oracle "new topic"` overwriting `state.json` without the prior episode's own close ever having been emitted (verified by the "durable state naming a different run" subtest).
- `colonyLiveEpisodeAbandoned` dispatches purely on `snapshot.EpisodeKind`, with every non-Oracle kind delegating to the pre-existing `colonyLiveEpisodeRunHasTerminated(s)` call verbatim -- chosen specifically so build/continue/plan/Swarm behavior is provably unaffected (the plan's own acceptance criterion: `TestWatchResolvesThreeBranchesFromEvidenceAlone`'s five subtests pass with zero edits to that test file).

## Deviations from Plan

None -- all three tasks executed as written, including the episode-open guard the plan left as an explicit either/or choice (applied).

### Break-it-to-prove-it observations

1. **Task 1 (episode wiring):** Temporarily removed the `openOracleLiveEpisode` call from `runOracleLoop`. `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` failed by name (`"runOracleLoop does not call openOracleLiveEpisode -- the live episode boundary is not wired into the loop"`), exactly as the plan's own `<behavior>` text specifies ("Deleting the boundary call from the loop fails the wiring guard by name"). `TestOracleRoundIsLiveWhileItRuns` itself stayed green under this same removal, because -- per the plan's own action text -- that test calls the production `openOracleLiveEpisode` and `emitOracleLiveRound` directly rather than driving them through `runOracleLoop`; it proves the semantic behavior of the boundary and the abandonment check together, while `TestOracleEpisodeBoundaryIsWiredIntoTheLoop` is the test that specifically proves the loop itself calls it. Restored the call; both tests passed again.
2. **Task 1/2 interlock (the "half of the gap a clean fixture would hide"):** After implementing Task 1 alone and before Task 2, `TestOracleRoundIsLiveWhileItRuns` genuinely FAILED (`mode = "replay", want "live"`) because the test's own fixture places a finished earlier build's terminated spawn run in the store, and `resolveWatchMode` at that point still called `colonyLiveEpisodeRunHasTerminated(s)` unconditionally -- demoting the live Oracle episode. This is precisely the second half of CR-01 the plan's objective describes ("Fixing only the boundary would produce a test that passes on an empty fixture while the real case still fails"). Implementing Task 2's `colonyLiveEpisodeAbandoned` dispatch turned the same test green with no change to the test itself.
3. **Task 2 (abandonment dispatch):** Temporarily reverted `resolveWatchMode` to call `colonyLiveEpisodeRunHasTerminated(s)` directly. 3 of 6 `TestAbandonedOracleRoundIsNotLive` subtests failed as expected: the terminated-spawn-run subtest (`mode = "replay", want "live"`), the dead-controller subtest (`mode = "live", want "replay"`), and the superseded-run subtest (`mode = "live", want "replay"`) -- the other three subtests (no readable state, non-Oracle kind, read-only) were unaffected by this specific revert, as expected since they do not depend on the dispatch. Restored the fix; all 6 subtests passed again.

---

**Total deviations:** 0
**Impact on plan:** None -- plan executed exactly as written; the observations above are the plan's own required break-it-to-prove-it verifications, all performed and recorded.

## Issues Encountered

None. `requirements.mark-complete CEC-05 LIVE-06` was expected to hit the same tool/format mismatch 202-16 recorded (this project's `**REQ-ID — Title:**` bold format vs. the tool's `**REQ-ID**`-only regex) -- confirmed below and worked around the same way.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CR-01 is fully closed: an actively iterating Oracle round now registers as live in `aether watch` in every colony state exercised, including one with a finished earlier build's spawn run present.
- `CEC-05` is now satisfied by both plan 202-16 (recovery/episode-selection half) and this plan (Oracle-liveness half); `LIVE-06` is satisfied by this plan alone.
- No blockers. `go build ./...` and `go vet ./cmd` are clean; the full named test list from this plan's `<verification>` block passes together (13.0s). The two pre-existing failures (`TestPhase199GateReceipt`, `TestCurrentVocabulary199`) were re-confirmed as unrelated and unaffected by this plan's changes.
- Ready for `/gsd-verify-work 202` -- both gap-closure plans (202-16, 202-17) for phase 202's verification report are now complete.

## Self-Check: PASSED

- All three commit hashes (`019403ef`, `7c871dbb`, `40579408`) found in `git log --oneline --all`.
- `go build ./...` and `go vet ./cmd` exit 0.
- Full named scoped test list from the plan's `<verification>` block re-run together and passed (13.0s), including `TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest`.
- All three break-it-to-prove-it observations performed and recorded above, with restoration confirmed by a subsequent green run.

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
