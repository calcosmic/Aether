---
phase: 202-swarm-oracle-and-live-colony
plan: "03"
subsystem: live-events
tags: [event-bus, build, continue, plan, recovery, live-colony, cockpit]

# Dependency graph
requires:
  - phase: 202-02
    provides: "pkg/events/colony_live.go's one versioned live-colony event model, cmd/live_events.go's emitColonyLive single emission boundary, cmd/live_projection.go's pure replay reducer, and the Swarm investigation wave's already-wired emission as the pattern to extend."
provides:
  - "cmd/live_events.go: 12 per-lane emission helpers (emitColonyLiveEpisodeStarted/Ended, WaveStarted/Ended, WorkerStarted/Progress/Finished, CheckStarted/Passed/Failed, RecoveryChanged, SignalConsulted) over the one emission boundary, plus two package-level active-episode carriers (build, continue) mirroring ceremony_emitter.go's activeBuildCeremony pattern."
  - "pkg/events/colony_live.go: a declared episode-kind vocabulary (EpisodeKindSwarm/Build/Continue/Plan/Recovery) and ColonyLiveEpisodeKinds(), read by the coverage test so a lane cannot silently be excluded from the lane inventory."
  - "pkg/codex/dispatch.go: WorkerDispatch.ParentWorkerID, letting worker-started lineage be read from the dispatch's own field rather than derived from a rendered string."
  - "Real emission wiring at the build (episode + wave + worker, both in-repo and worktree dispatch loops), continue (check episode + one terminal event per deterministic-floor check), planning (episode + wave + worker), and recovery (recovery-changed from the one funnel point orchestrateRecovery now returns through) transition sites."
  - "cmd/live_lane_coverage_test.go: TestEveryLifecycleLaneEmitsLiveEvents, deriving its lane inventory from events.ColonyLiveEpisodeKinds() and driving each lane through a real public entry point, plus a negative case proving the guard can fail and names the lane."
affects: [202-06, 202-09, 202-10, 202-11]

# Actuals (#2632)
actuals:
  tokens: 17024
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Active-episode carrier: a package-level RWMutex-guarded string (setActiveLiveBuildEpisode/currentLiveBuildEpisode, and its continue counterpart) threads an episode ID from the top of a lane's call chain down into a deeply nested dispatch loop without adding a parameter to every intermediate function signature -- the exact same problem cmd/ceremony_emitter.go's activeBuildCeremony/setActiveBuildCeremony/currentBuildCeremony already solved for ceremony emission, solved the same way."
    - "Recovery emission funnels through one return point: orchestrateRecovery's four classification branches (Blocking, RequiresAttempt, Recoverable, default) were refactored to assign to one `outcome` variable rather than returning directly, so a single emitColonyLiveRecoveryChanged call before the function's one return covers every branch without duplicating the call four times."
    - "Test-only emission-failure seam (colonyLiveEmissionFailureOverride), mirroring the existing colonyLiveSchemaVersionOverride seam from 202-02: forces emitColonyLive to no-op at the exact point a real bus.Publish failure would occur, so TestFailedLiveEmitNeverChangesLaneOutcome proves non-blocking emission against the real code path rather than a mocked one."

key-files:
  created:
    - cmd/live_lane_wiring_test.go
  modified:
    - cmd/live_events.go
    - cmd/live_lane_coverage_test.go
    - cmd/codex_build.go
    - cmd/codex_build_worktree.go
    - cmd/codex_continue.go
    - cmd/codex_plan.go
    - cmd/recovery_orchestrator.go
    - pkg/codex/dispatch.go
    - pkg/events/colony_live.go

key-decisions:
  - "Real per-worker build wiring lives in cmd/codex_build_worktree.go's dispatchCodexBuildWorkersInRepo and dispatchCodexBuildWorkersWithReconciliation, not cmd/codex_build.go -- that file's own dispatch loop only wraps the wave-lifecycle call and holds the existing ceremony worker-start/finish emission the plan said to sit 'immediately adjacent to'. codex_build.go itself owns only the build's episode boundary (start/end)."
  - "Continue's check-started/passed/failed loop reads floor.Steps from the ORIGINAL runDeterministicFloor call (before applyBoundedCheckFixRepair may re-run it), so a bounded auto-fix repair attempt does not double-emit terminal events for the same check names."
  - "Recovery's episode ID is derived as fmt.Sprintf(\"recovery-phase-%d\", ctx.Phase) from RecoveryContext's own Phase field -- RecoveryContext carries no independent run/attempt identifier to reuse, and orchestrateRecovery is documented as a pure classify/decide/log/return function, so no new field was added to it."
  - "WorkerDispatch gained a ParentWorkerID field (pkg/codex/dispatch.go) so emitColonyLiveWorkerStarted can satisfy the 'dispatch that names a parent' acceptance criterion structurally; no current build/continue/plan call site sets it (every dispatched worker in these lanes is a direct Queen dispatch with no worker parent), matching Swarm's own already-shipped behavior of an empty ParentWorkerID."
  - "Planning's per-worker emission (worker-started immediately followed by worker-finished) reflects that planning dispatches are already fully resolved (Name/Caste/Status) by the point emitPlanCeremonyDispatchSequence runs -- unlike build's async per-worker loop, there is no genuine gap between a planning worker's start and its terminal result to observe."

patterns-established:
  - "Per-lane emission helper over the one boundary: every lifecycle lane calls a thin, typed cmd/live_events.go wrapper (never emitColonyLive directly at its own call site), mapping only the fields its already-available domain value (codex.WorkerDispatch, codex.DispatchResult, a check name/outcome, a recovery state) genuinely carries."

requirements-completed: [CEC-05, LIVE-01]

coverage:
  - id: D1
    description: "Every lifecycle lane -- planning, building, checking, recovery -- speaks on the live stream through a typed per-lane helper over the one emission boundary, never a hand-rolled emitColonyLive call at its own site."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_coverage_test.go#TestLiveLaneHelpersMapOnlyKnownFields"
        status: pass
      - kind: unit
        ref: "cmd/live_lane_coverage_test.go#TestEveryLiveEventGoesThroughOneBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "A build, driven through the real direct build lane, emits the ordered episode-started / wave-started / worker-started / worker-finished / wave-ended / episode-ended sequence, all sharing one episode ID and the build episode kind."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_wiring_test.go#TestBuildLaneEmitsOrderedLiveEvents"
        status: pass
    human_judgment: false
  - id: D3
    description: "A check run, driven through the real continue lane with documented verification commands, emits check-started followed by exactly one terminal event per check the deterministic floor actually ran, with check names matching the floor's own names."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_wiring_test.go#TestCheckLaneEmitsOneTerminalEventPerCheck"
        status: pass
    human_judgment: false
  - id: D4
    description: "A planning pass, driven through the real synthetic plan lane, emits episode/wave/worker events whose episode kind is the planning kind."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_wiring_test.go#TestPlanLaneEmitsPlanningEpisode"
        status: pass
    human_judgment: false
  - id: D5
    description: "A recovery transition, driven through the real buildExternalBuildRecoveryInstructions entry point, emits a recovery-changed event whose RecoveryState equals the action actually persisted to the durable recovery log."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_wiring_test.go#TestRecoveryLaneEmitsRecordedState"
        status: pass
    human_judgment: false
  - id: D6
    description: "A failed live-event publish never changes the result value or durable state of the build, check, plan, or recovery lane it describes."
    requirement: "LIVE-01"
    verification:
      - kind: unit
        ref: "cmd/live_lane_wiring_test.go#TestFailedLiveEmitNeverChangesLaneOutcome"
        status: pass
    human_judgment: false
  - id: D7
    description: "A lane that stops emitting live events is caught by a failing test naming the lane and its episode kind, not by an owner noticing an empty cockpit; the lane inventory is derived from the declared episode-kind vocabulary, not a hand-typed list."
    requirement: "CEC-05"
    verification:
      - kind: unit
        ref: "cmd/live_lane_coverage_test.go#TestEveryLifecycleLaneEmitsLiveEvents"
        status: pass
    human_judgment: false

duration: 58min
completed: 2026-09-11
status: complete
---

# Phase 202 Plan 03: Every Lifecycle Lane Speaks on the Live Stream Summary

**Extended the one live-event model from Swarm to build, continue, plan and recovery — 12 typed emission helpers, real wiring at each lane's actual dispatch/check/recovery transitions, and a coverage test that fails by name the moment a lane goes dark.**

## Performance

- **Duration:** 58 min
- **Started:** 2026-09-11T09:04:50+02:00 (first task commit)
- **Completed:** ~2026-09-11T10:02:00+02:00 (third task commit)
- **Tasks:** 3
- **Files modified:** 9 (2 created, 7 modified)

## Accomplishments

- Added 12 per-lane emission helpers to `cmd/live_events.go` (episode/wave/worker/check/recovery/signal moments), each mapping only the fields its already-available domain value genuinely carries, plus a `TestLiveLaneHelpersMapOnlyKnownFields` reflection-based test proving that field-by-field for every helper, including a nil-store no-op check per helper.
- Added `WorkerDispatch.ParentWorkerID` (`pkg/codex/dispatch.go`) so worker-started lineage reads from the dispatch's own field, and a declared `ColonyLiveEpisodeKinds()` vocabulary (`pkg/events/colony_live.go`) the coverage test derives its lane inventory from.
- Wired real emission at the build lane's actual transitions: episode start/end in `cmd/codex_build.go` around the direct build lane's own run handle, and wave/worker start/finish in `cmd/codex_build_worktree.go`'s two dispatch loops (in-repo and worktree) — the file that actually owns per-worker dispatch, immediately adjacent to the existing ceremony emission calls there.
- Wired the continue lane: one check episode wrapping the whole `runCodexContinue` run, and check-started/passed/failed emitted per non-skipped `runDeterministicFloor` step, using an active-episode carrier so the nested verification function can read the top-level episode ID.
- Wired the planning lane: an episode wrapping `runCodexPlanWithOptionsInSession`'s planning run, and a wave plus per-worker started/finished pair emitted immediately adjacent to the existing `emitPlanCeremonyDispatchSequence` ceremony call.
- Wired recovery: refactored `orchestrateRecovery`'s four classification branches to funnel through one `outcome` variable and one `emitColonyLiveRecoveryChanged` call before its single return, so every real caller (`buildExternalBuildRecoveryInstructions`, the continue check-fix path, and the queen wave lifecycle) gets live emission without duplicating the call.
- Proved non-blocking emission structurally: a test-only `colonyLiveEmissionFailureOverride` seam forces `emitColonyLive` to no-op at the real publish point, and `TestFailedLiveEmitNeverChangesLaneOutcome` runs one build, one check, one planning pass and one recovery transition twice (normal vs. forced-failure) against isolated fixtures, asserting identical outcomes.
- Added `TestEveryLifecycleLaneEmitsLiveEvents`, driving all five declared episode kinds (swarm, build, continue, plan, recovery) through their real public entry points, plus a negative case proving the guard can fail and names the stubbed lane.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the per-lane emission helpers over the one boundary** - `14930be1` (test)
2. **Task 2: Emit at the real transitions in build, continue, plan and recovery** - `30af684f` (feat)
3. **Task 3: Fail the build when a lifecycle lane goes dark** - `515a13d5` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/live_events.go` - 12 per-lane emission helpers, two active-episode carriers (build, continue), and the `colonyLiveEmissionFailureOverride` test seam.
- `cmd/live_lane_coverage_test.go` - `TestLiveLaneHelpersMapOnlyKnownFields` and `TestEveryLifecycleLaneEmitsLiveEvents` (created; extended across Tasks 1 and 3).
- `cmd/live_lane_wiring_test.go` - `TestBuildLaneEmitsOrderedLiveEvents`, `TestCheckLaneEmitsOneTerminalEventPerCheck`, `TestPlanLaneEmitsPlanningEpisode`, `TestRecoveryLaneEmitsRecordedState`, `TestFailedLiveEmitNeverChangesLaneOutcome`.
- `cmd/codex_build.go` - the direct build lane's live episode boundary (start/end), reusing the same run identifier `finishRuntimeSpawnRun` persists.
- `cmd/codex_build_worktree.go` - wave-started/worker-started/worker-finished/wave-ended emission in both `dispatchCodexBuildWorkersInRepo` and `dispatchCodexBuildWorkersWithReconciliation`.
- `cmd/codex_continue.go` - the check episode boundary and per-check-step check-started/passed/failed emission.
- `cmd/codex_plan.go` - the planning episode boundary and its wave/worker emission.
- `cmd/recovery_orchestrator.go` - `orchestrateRecovery` refactored to funnel through one outcome variable with a single `emitColonyLiveRecoveryChanged` call.
- `pkg/codex/dispatch.go` - `WorkerDispatch.ParentWorkerID`.
- `pkg/events/colony_live.go` - the episode-kind vocabulary and `ColonyLiveEpisodeKinds()`.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Real build per-worker wiring required `cmd/codex_build_worktree.go`, not just `cmd/codex_build.go`**
- **Found during:** Task 2, while locating the real dispatch/wave transition sites named in the plan's action text
- **Issue:** The plan's `files_modified` frontmatter and Task 2's action text named `cmd/codex_build.go` as the build wiring target. The actual per-worker dispatch loops — and the pre-existing `emitBuildCeremonyWaveStart`/`emitBuildCeremonyWorkerStarting`/`emitBuildCeremonyWorkerFinished`/`emitBuildCeremonyWaveEnd` ceremony calls the plan said to sit "immediately adjacent to" — live in `cmd/codex_build_worktree.go`'s `dispatchCodexBuildWorkersInRepo` and `dispatchCodexBuildWorkersWithReconciliation` functions. `codex_build.go` itself only computes the wave/dispatch plan and calls `executeCodexBuildDispatches`, which delegates to `queenWaveLifecycle` and ultimately those two functions.
- **Fix:** Added the build episode boundary (start/end) to `codex_build.go` as planned, and added wave/worker emission to `codex_build_worktree.go`'s two dispatch loops, immediately adjacent to the existing ceremony calls there — following the plan's substantive instruction over its literal file list.
- **Files modified:** `cmd/codex_build.go`, `cmd/codex_build_worktree.go`
- **Verification:** `TestBuildLaneEmitsOrderedLiveEvents` proves the full ordered sequence from a real fixture build; `TestBuildWritesDispatchArtifactsAndUpdatesState` (pre-existing) still passes unchanged.
- **Committed in:** `30af684f` (Task 2 commit)

**2. [Rule 2 - Missing Critical] Added a declared episode-kind vocabulary to `pkg/events/colony_live.go`**
- **Found during:** Task 3, while designing `TestEveryLifecycleLaneEmitsLiveEvents`
- **Issue:** The task's action text requires deriving the lane inventory "from the episode-kind vocabulary declared in `pkg/events/colony_live.go`," but 202-02 left `EpisodeKind` as a free-form string field with no declared vocabulary — there was nothing to derive from.
- **Fix:** Added `EpisodeKindSwarm`/`Build`/`Continue`/`Plan`/`Recovery` constants and `ColonyLiveEpisodeKinds()`, mirroring `ColonyLiveTopics()`'s own completeness contract.
- **Files modified:** `pkg/events/colony_live.go`
- **Verification:** `TestEveryLifecycleLaneEmitsLiveEvents` reads the inventory from this function; `TestColonyLiveTopicsAreComplete` (pre-existing, scoped to `LiveTopic*` constants only) is unaffected.
- **Committed in:** `515a13d5` (Task 3 commit)

**3. [Rule 2 - Missing Critical] Added `WorkerDispatch.ParentWorkerID`**
- **Found during:** Task 1, while implementing `emitColonyLiveWorkerStarted`'s "dispatch that names a parent" acceptance criterion
- **Issue:** `codex.WorkerDispatch` had no field naming a parent worker, so the helper had no structured value to read lineage from without deriving it from a rendered string — the exact prohibition the plan names.
- **Fix:** Added an optional `ParentWorkerID string` field, defaulting to empty (matching every current call site, since no build/continue/plan dispatch in these lanes currently has a worker parent).
- **Files modified:** `pkg/codex/dispatch.go`
- **Verification:** `TestLiveLaneHelpersMapOnlyKnownFields`'s "with a parent" and "without a parent" subtests.
- **Committed in:** `14930be1` (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (1 Rule 3 blocking file-location correction, 2 Rule 2 missing-critical additions the plan's own acceptance criteria required).
**Impact on plan:** All three were necessary for the plan's own stated behavior/acceptance criteria to hold. No scope creep — no production behavior beyond live-event emission was touched, and the build/continue/plan/recovery lanes' actual results and durable state are proven byte-for-byte unaffected by `TestFailedLiveEmitNeverChangesLaneOutcome`.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 202-06 (the cockpit) now has real live events to render from every command the owner runs, not only Swarm.
- Oracle's own live emission is explicitly out of this plan's scope (CEC-05 names it, but 202-CLASSIC-SYNTHESIS.md's SYN-202 decisions and this plan's task list cover planning/building/checking/recovery/verification, mirroring Swarm's already-shipped pattern) — a later plan in this phase owns Oracle's wiring if not already covered.
- The worktree build path's per-worker wiring landed in the same pass as the in-repo path (both loops in `cmd/codex_build_worktree.go`), so a worktree-mode build also speaks on the live stream, not only the default in-repo mode.
- No blockers.

## Self-Check: PASSED

- `cmd/live_events.go` — FOUND
- `cmd/live_lane_coverage_test.go` — FOUND
- `cmd/live_lane_wiring_test.go` — FOUND
- `cmd/codex_build.go` — FOUND
- `cmd/codex_build_worktree.go` — FOUND
- `cmd/codex_continue.go` — FOUND
- `cmd/codex_plan.go` — FOUND
- `cmd/recovery_orchestrator.go` — FOUND
- `pkg/codex/dispatch.go` — FOUND
- `pkg/events/colony_live.go` — FOUND
- Commit `14930be1` — FOUND in `git log --oneline --all`
- Commit `30af684f` — FOUND in `git log --oneline --all`
- Commit `515a13d5` — FOUND in `git log --oneline --all`
- `go test ./cmd -run '^(TestLiveLaneHelpersMapOnlyKnownFields|TestEveryLiveEventGoesThroughOneBoundary)$' -count=1` — PASS
- `go test ./cmd -run '^(TestBuildLaneEmitsOrderedLiveEvents|TestCheckLaneEmitsOneTerminalEventPerCheck|TestPlanLaneEmitsPlanningEpisode|TestRecoveryLaneEmitsRecordedState|TestFailedLiveEmitNeverChangesLaneOutcome)$' -count=1 && go vet ./cmd` — PASS
- `go test ./cmd -run '^TestEveryLifecycleLaneEmitsLiveEvents$' -count=1` — PASS
- `go test ./pkg/codex/... ./pkg/events/... -count=1` — PASS

---
*Phase: 202-swarm-oracle-and-live-colony*
*Completed: 2026-09-11*