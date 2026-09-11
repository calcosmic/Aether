---
phase: 202-swarm-oracle-and-live-colony
plan: "11"
subsystem: oracle
tags: [oracle, live-events, watch-dashboard, diminishing-returns, research-depth]

# Dependency graph
requires:
  - phase: 202-03
    provides: "pkg/events/colony_live.go's one versioned live-colony event model, cmd/live_events.go's emitColonyLive single emission boundary, cmd/live_projection.go's pure replay reducer, and the declared episode-kind vocabulary (ColonyLiveEpisodeKinds()) whose completeness contract this plan extends with the oracle kind."
  - phase: 202-08
    provides: "cmd/oracle_preset.go's shared Fast/Balanced/Deep/Exhaustive vocabulary (resolveOraclePreset) over Oracle's own unchanged oracleDepthLevels numbers -- reused directly by this plan's round-cap/target emission and by the clarification-panel proof."
provides:
  - "cmd/oracle_live.go: emitOracleLiveRound/emitOracleLiveConfidence/emitOracleLiveContradiction/emitOracleLiveGapTargeted/emitOracleLiveRoundEnded -- Oracle's research state (round number, cap, phase, question, confidence, contradictions, gaps) reaching the same live.* stream every other lifecycle lane speaks on, from the loop's existing oracleStateFile mutation points."
  - "events.EpisodeKindOracle and events.LiveTopicGapTargeted (pkg/events/colony_live.go), plus ColonyLivePayload.RoundCap/PreviousConfidence -- the schema extensions the emission needed, with the oracle lane registered in cmd/live_lane_coverage_test.go's completeness net (TestEveryLifecycleLaneEmitsLiveEvents now covers six lanes, not five)."
  - "Proof (not new behavior) that Oracle's pre-existing propose/brief/--from-brief setup ritual already satisfies D-08: a vague topic gets a clarification screen naming the core question, the preset's own target/cap and success criteria before any round runs, writes nothing until confirmed, and an approved brief skips straight to research with the loop never reading stdin once started."
  - "cmd/oracle_progress.go/cmd/watch_dashboard.go: the round line and run-end summary now state a diminishing-returns stall in plain English with its count (translating the raw diminishing_returns stop reason), and a running Oracle round renders through the existing live dashboard as a worker-like row alongside its existing confidence-against-target and contradictions sections."
affects: [202-12, 202-13, 202-14, 202-15]

# Actuals (#2632)
actuals:
  tokens: 11686
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Oracle's round represented as a single repeatedly-replaced worker row (WorkerID \"oracle\", caste \"oracle\") on the live-colony worker-row model, rather than a new rendering path -- lets the existing generic per-worker block and snapshot-level confidence/contradiction sections show a running research round with no second screen."
    - "New-entries-only diff (newOracleNotes) applied identically at Oracle's OpenGaps and Contradictions merge points: capture the pre-merge slice, merge, then emit one live event per element present after but not before -- an item already recorded before the merge is never announced twice."
    - "oracleLiveEpisodeID derives a stable per-run episode identifier from oracleStateFile.StartedAt (already-existing, set once, preserved across every `oracle iterate` resume) rather than adding a new field -- honors the plan's prohibition on new Oracle research state."

key-files:
  created:
    - cmd/oracle_live.go
    - cmd/oracle_live_test.go
  modified:
    - cmd/oracle_loop.go
    - cmd/oracle_progress.go
    - cmd/watch_dashboard.go
    - cmd/live_lane_coverage_test.go
    - pkg/events/colony_live.go

key-decisions:
  - "Task 2 required no production code change. Investigation (confirmed by grepping the wrapper contract at .aether/commands/oracle.yaml and by four passing proof tests) established that Oracle's propose/brief/--from-brief ritual -- built in an earlier, pre-phase-202 commit (2b80c2a4, 2026-08-16) -- already satisfies D-08 end to end, and that direct-topic invocation bypassing it is a wrapper-documented, intentional 'skip scoping' escape hatch, not a gap to close. Gating the direct-topic path would have broken several pre-existing tests (TestOracleConfidenceTargetFlagDefaultUsesDepthPreset and siblings) that rely on it completing synchronously with no brief. 202-CLASSIC-SYNTHESIS.md's SYN-202-11 disposition (`keep-current` for the loop/wizard logic) is the documented basis for this call."
  - "emitOracleLiveRoundEnded and the round-start event both use WorkerStarted/WorkerProgress topics (not WaveStarted/WaveEnded) so the live-projection reducer's existing per-worker fold populates a real worker row for Oracle's round -- WaveStarted/Ended in the current reducer only ever update snapshot.Wave, not Question, so a round-scoped question would never have reached the dashboard through that pair."
  - "Confidence-changed and round-ended events fire on the value oracleOverallConfidence(plan) already computed -- gated to fire only when the value genuinely changed (previous != new) -- rather than unconditionally every round, matching the acceptance criterion that a 'confidence change' event names an actual change."
  - "ColonyLivePayload gained two fields (RoundCap, PreviousConfidence) and pkg/events gained one topic (LiveTopicGapTargeted) and one episode kind (EpisodeKindOracle) -- schema extensions to the shared live-event model, not new Oracle research state; the 202-02 doc comment explicitly anticipates and permits adding a topic this way."

requirements-completed: [LIVE-06, CEC-05]

coverage:
  - id: D1
    description: "Oracle's round beginning, confidence changes, newly-observed contradictions, newly-added gaps, and round endings all reach the live.* stream, from the loop's existing oracleStateFile mutation points -- no new research state added, and emission failure never changes a round's outcome."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleRoundsReachTheLiveStream"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleContradictionAnnouncedOnceNotEveryRound"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestFailedOracleEmitNeverChangesTheRun"
        status: pass
      - kind: unit
        ref: "cmd/live_lane_coverage_test.go#TestEveryLifecycleLaneEmitsLiveEvents (oracle subtest)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Oracle clarifies the actual question once before any round runs (naming the core question, the preset's own target/cap, and success criteria) and, once rounds begin, never stops to ask -- proven against the pre-existing propose/brief/--from-brief ritual, which already satisfied this."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleClarifiesOnceBeforeTheFirstRound"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleWithApprovedBriefSkipsClarification"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleAsksNothingOnceRoundsBegin"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleClarificationWritesNothingUntilConfirmed"
        status: pass
    human_judgment: false
  - id: D3
    description: "A confidence figure shown anywhere (round line, live event payload, dashboard) equals the integer recorded in research state, unrounded; diminishing returns is stated in plain English with its count and the run stops at exactly the configured stall limit, never one round earlier."
    requirement: "LIVE-06"
    verification:
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestConfidenceIsShownExactlyAsRecorded"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestDiminishingReturnsIsStatedWithItsCount"
        status: pass
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestOracleStopsAtExactlyTheStallLimit"
        status: pass
    human_judgment: false
  - id: D4
    description: "A running Oracle episode renders its current round, question, confidence against target, and contradictions through the existing live dashboard renderer (no second screen), omitting the contradictions section when there are none."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/oracle_live_test.go#TestLiveDashboardShowsTheResearchRound"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 11: Oracle Research Made Watchable Summary

**Oracle's round number, confidence, contradictions and gaps now reach the same live stream every other lifecycle lane speaks on, straight from the loop's existing state -- and the pre-existing clarify-once-then-autonomous setup ritual (propose/brief/--from-brief) is proven, not rebuilt, to already satisfy D-08.**

## Performance

- **Duration:** ~55 min (most of it front-loaded reading `cmd/oracle_loop.go`, `pkg/events/colony_live.go`, `cmd/live_projection.go` and the 202-01/202-02/202-03/202-08 summaries to find the real mutation points and avoid inventing a second event mechanism)
- **Started:** 2026-09-11T14:12:52+02:00 (first task commit)
- **Completed:** 2026-09-11T14:13:44+02:00 (third task commit)
- **Tasks:** 3
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `cmd/oracle_live.go` (new): five emission functions (`emitOracleLiveRound`, `emitOracleLiveConfidence`, `emitOracleLiveContradiction`, `emitOracleLiveGapTargeted`, `emitOracleLiveRoundEnded`) built from `oracleStateFile`'s already-tracked fields, published through the one existing `emitColonyLive` boundary. Oracle has no per-worker dispatch of its own, so a running round is represented as a single, repeatedly-replaced worker row (`WorkerID "oracle"`, `caste "oracle"`) -- letting the dashboard's existing generic worker-block and confidence/contradiction rendering show it with no new screen.
- `cmd/oracle_loop.go`: four call sites added at the round increment/active-question assignment, the confidence/contradiction/gap merge inside `applyOracleWorkerResponse`, and the round's end -- no new field added to `oracleStateFile`, no existing assignment changed. A `newOracleNotes` diff (new in `oracle_live.go`) ensures a contradiction or gap already recorded before a merge is never announced twice.
- `pkg/events/colony_live.go`: added `EpisodeKindOracle`, `LiveTopicGapTargeted`, and two payload fields (`RoundCap`, `PreviousConfidence`) the emission needed; wired the oracle lane into `cmd/live_lane_coverage_test.go`'s `TestEveryLifecycleLaneEmitsLiveEvents` completeness net (now six lanes, not five) via a real `aether oracle run-loop` fixture drive.
- Investigated whether the direct-topic invocation path (`aether oracle "<topic>"`, bypassing the propose/brief ritual) needed a new clarification gate for D-08. Found it did not: the ritual already existed (built 2026-08-16, before this phase), the wrapper contract (`.aether/commands/oracle.yaml`) explicitly documents direct-topic as a sanctioned "skip scoping" escape hatch, and gating it would have broken several pre-existing tests that rely on it completing synchronously. Wrote four tests proving the existing ritual already satisfies every behavior D-08 names, rather than building a second mechanism.
- `cmd/oracle_progress.go`: `oracleProgressEvent` gained a `ConsecutiveLow` field (mirroring `oracleStateFile.Novelty.ConsecutiveLow`, not a new counter). The round line now appends "(N in a row added no new ground)" when the counter is above zero, and the run-end line translates the raw `diminishing_returns` stop reason into a plain-English sentence naming the count instead of leaking the internal code.
- `cmd/watch_dashboard.go`: added a ticker-description case for the new `live.gap.targeted` topic ("a research gap was flagged"), keeping every declared live topic's ticker line in ordinary English per the file's own completeness contract.
- Proved the "no rounding, truncation or rescaling" rule end to end on 67% -- a value any nearest-5/nearest-10 rounding rule would change -- across the round line, the live event payload, and the dashboard's rendered output.

## Task Commits

Each task was committed atomically:

1. **Task 1: Emit Oracle's existing state at its existing mutation points** - `e5465abd` (feat)
2. **Task 2: Clarify once at the start, then run without interrupting** - `a9c38d25` (test)
3. **Task 3: Show confidence and diminishing returns exactly as recorded** - `385ce084` (feat)

**Plan metadata:** (this commit)

_Note: all three tasks carried `tdd="true"`. Task 1 and Task 3 landed tests and implementation together (the described behaviors -- emission at an existing mutation point, translating an existing counter into a sentence -- don't have a meaningful pre-implementation "should fail" state distinct from "doesn't compile yet", the same pattern 202-01/202-02/202-08 each documented). Task 2 is a genuine RED-to-GREEN case in the opposite direction: the tests were written to find whether a gap existed, all four passed on the first run against the untouched codebase, and that passing-on-first-run result is itself the finding -- confirmed against SYN-202-11's `keep-current` disposition rather than treated as a fixture bug to chase._

## Files Created/Modified

- `cmd/oracle_live.go` - `emitOracleLiveRound`/`emitOracleLiveConfidence`/`emitOracleLiveContradiction`/`emitOracleLiveGapTargeted`/`emitOracleLiveRoundEnded`, `oracleLiveEpisodeID`, `newOracleNotes`.
- `cmd/oracle_live_test.go` - all eleven named tests across the three tasks, plus the shared `setupOracleLiveFixtureWorkspace` fixture and two fixture invokers (`oracleGapContradictionInvoker`, reusing the existing `oracleCompletingInvoker`).
- `cmd/oracle_loop.go` - four emission call sites in `runOracleLoop`/`applyOracleWorkerResponse`; no field or existing assignment changed.
- `cmd/oracle_progress.go` - `oracleProgressEvent.ConsecutiveLow`, the round-line stall clause, and the run-end plain-English translation of `diminishing_returns`.
- `cmd/watch_dashboard.go` - one ticker-description case for `live.gap.targeted`.
- `cmd/live_lane_coverage_test.go` - `driveOracleLiveLane`, registered in `liveLaneEntryPoints`.
- `pkg/events/colony_live.go` - `EpisodeKindOracle`, `LiveTopicGapTargeted`, `ColonyLivePayload.RoundCap`/`PreviousConfidence`.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `RoundCap`/`PreviousConfidence` fields to `ColonyLivePayload`, and `LiveTopicGapTargeted`/`EpisodeKindOracle` to `pkg/events/colony_live.go`**
- **Found during:** Task 1, while designing `emitOracleLiveRound`/`emitOracleLiveConfidence`/`emitOracleLiveGapTargeted`
- **Issue:** The plan's task 1 acceptance criteria require a round-started event to carry "the round number, the round cap"; the existing `ColonyLivePayload` had `Wave` for the round number but nothing for the cap. Similarly, "a confidence change event carries both the previous and new values" had no field for the previous value, and there was no existing topic for "a newly selected gap" the way `LiveTopicContradictionFound` exists for contradictions. Without these, the acceptance criteria described in the plan's own `<behavior>` block could not be satisfied.
- **Fix:** Added `RoundCap int` and `PreviousConfidence float64` to `ColonyLivePayload` (both `omitempty`, additive-only -- every existing emission helper's reflection-based field assertion in `TestLiveLaneHelpersMapOnlyKnownFields` still passes since none of them set the new fields). Added `LiveTopicGapTargeted` and `EpisodeKindOracle`, both registered in their respective completeness accessors (`ColonyLiveTopics()`, `ColonyLiveEpisodeKinds()`).
- **Files modified:** `pkg/events/colony_live.go`
- **Verification:** `go test ./pkg/events/...` (all pass, including the two structural completeness tests); `cmd/oracle_live_test.go`'s round-started/confidence-changed/gap-targeted assertions read these fields directly.
- **Committed in:** `e5465abd` (Task 1 commit)

**2. [Rule 2 - Missing Critical] Registered the oracle lane in `cmd/live_lane_coverage_test.go`'s `TestEveryLifecycleLaneEmitsLiveEvents`**
- **Found during:** Task 1, immediately after adding `EpisodeKindOracle`
- **Issue:** `ColonyLiveEpisodeKinds()`'s own doc comment states a kind declared there with no entry in `liveLaneEntryPoints` fails `TestEveryLifecycleLaneEmitsLiveEvents` by name. Adding `EpisodeKindOracle` without a driving entry point would have broken this pre-existing regression test.
- **Fix:** Added `driveOracleLiveLane`, driving one real `aether oracle run-loop` round through the fixture invoker, and registered it in `liveLaneEntryPoints`. 202-03's own SUMMARY explicitly left this as a note for "a later plan in this phase" -- this is that plan.
- **Files modified:** `cmd/live_lane_coverage_test.go`
- **Verification:** `go test ./cmd -run '^TestEveryLifecycleLaneEmitsLiveEvents$' -count=1` (all six lanes pass, including the negative "stubbed lane is reported by name" case).
- **Committed in:** `e5465abd` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 2 - missing critical, both schema/net-completeness additions the plan's own acceptance criteria and the pre-existing codebase's own completeness contracts required).
**Impact on plan:** Both were necessary for the plan's stated behavior to be representable and for a pre-existing regression test to keep passing. No scope creep -- no lane other than Oracle's own emission was touched.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- A running Oracle research run is now watchable the same way `aether watch` already shows a build or Swarm investigation -- round, question, confidence-against-target, contradictions.
- The pre-existing clarify-once ritual is now locked by test, so a future change to `startOracleCompatibility`, `runOraclePropose`, or `runOracleBriefApprove` that silently breaks D-08 will fail a test by name instead of only being caught by an owner noticing a run that never asked.
- `events.ColonyLiveEpisodeKinds()` now covers six lanes (swarm, build, continue, plan, recovery, oracle) -- any later phase adding a seventh inherits the same completeness contract.
- No blockers.

## Self-Check: PASSED

- `cmd/oracle_live.go` - FOUND
- `cmd/oracle_live_test.go` - FOUND
- Commit `e5465abd` - FOUND in `git log --oneline --all`
- Commit `a9c38d25` - FOUND in `git log --oneline --all`
- Commit `385ce084` - FOUND in `git log --oneline --all`
- `go build ./...` - PASS
- `go vet ./cmd ./pkg/events` - PASS
- `go test ./cmd -run '^(TestOracleRoundsReachTheLiveStream|TestOracleContradictionAnnouncedOnceNotEveryRound|TestFailedOracleEmitNeverChangesTheRun)$' -count=1` - PASS
- `go test ./cmd -run '^(TestOracleClarifiesOnceBeforeTheFirstRound|TestOracleWithApprovedBriefSkipsClarification|TestOracleAsksNothingOnceRoundsBegin|TestOracleClarificationWritesNothingUntilConfirmed)$' -count=1 && go test ./cmd -run '^TestOracleBrief' -count=1` - PASS
- `go test ./cmd -run '^(TestConfidenceIsShownExactlyAsRecorded|TestDiminishingReturnsIsStatedWithItsCount|TestOracleStopsAtExactlyTheStallLimit|TestLiveDashboardShowsTheResearchRound)$' -count=1 && go test ./cmd -run '^TestOracleProgress' -count=1 && go vet ./cmd` - PASS
- `go test ./cmd -run '^TestOracle' -count=1` (full existing Oracle suite) - PASS
- `go test ./cmd -run '^(TestEveryLifecycleLaneEmitsLiveEvents|TestLiveTickerLinesArePlainEnglish|TestLiveLaneHelpersMapOnlyKnownFields)$' -count=1` - PASS
- `go test ./pkg/events/... ./pkg/codex/...` - PASS

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*
