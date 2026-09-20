---
phase: 114-real-worker-dispatch
plan: 02
subsystem: orchestration
tags: [typescript, worker-dispatch, parallel, retry, wave-orchestrator, cli]

requires:
  - phase: 114-01
    provides: "Real worker dispatch foundation (dispatchSingleWorker, platform-dispatcher)"

provides:
  - "Wave orchestrator with parallel dispatch, retry, and timeout options"
  - "Worker dispatch module updated to delegate to wave orchestrator"
  - "Lifecycle defaults to real dispatch with graceful simulation fallback"
  - "--simulate CLI flag for testing without real worker spawning"
  - "Unit tests for wave orchestrator (6) and worker dispatch (4)"

affects:
  - 115-ceremony-narrator
  - 116-ts-host-integration

tech-stack:
  added: []
  patterns:
    - "Wave grouping + Promise.all for parallel within-wave dispatch"
    - "Exponential backoff retry loop with configurable limit"
    - "Mutable module-level reference for test injection in ESM"
    - "Graceful fallback: detect platforms, warn, simulate if none"

key-files:
  created:
    - .aether/ts-host/src/wave-orchestrator.ts
    - .aether/ts-host/test/wave-orchestrator.test.ts
  modified:
    - .aether/ts-host/src/worker-dispatch.ts
    - .aether/ts-host/src/lifecycle.ts
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/test/worker-dispatch.test.ts

key-decisions:
  - "Kept retry logic inside wave-orchestrator rather than platform-dispatcher to keep timeout enforcement at the spawn level and retry at the orchestration level"
  - "Used mutable _dispatchSingleWorker reference with __setDispatchSingleWorker test helper because ESM exports are read-only and jest-style module mocking is unavailable in node:test"
  - "Default simulateWorkers changed from true to false in lifecycle, with automatic fallback when no platforms detected, preserving backward compatibility for tests that explicitly pass simulateWorkers: true"

patterns-established:
  - "WaveOrchestratorOptions extends DispatchOptions: parallel, retryLimit, retryDelayMs, timeoutMs"
  - "WaveResult per wave: results, failures, retried count"
  - "Lifecycle emits ceremony.build.wave.start/end to stderr as placeholder until event bridge is wired in Phase 115"

requirements-completed:
  - TS-02
  - TS-03

metrics:
  duration: 16m
  completed: 2026-05-13
---

# Phase 114 Plan 02: Wave Orchestration and Real Dispatch Wiring Summary

**Wave orchestrator with Promise.all parallel dispatch, exponential backoff retry, and lifecycle defaulting to real workers with graceful simulation fallback via --simulate CLI flag.**

## Performance

- **Duration:** 16m
- **Started:** 2026-05-13T19:58:54Z
- **Completed:** 2026-05-13T20:15:00Z
- **Tasks:** 5
- **Files modified:** 6

## Accomplishments

- Built wave-orchestrator.ts with dispatchWave, retryDispatch, dispatchWaves
- Wired worker-dispatch.ts to delegate multi-worker dispatch to wave orchestrator
- Updated lifecycle.ts to default to real dispatch, detect platforms, fallback gracefully
- Added --simulate CLI flag to host.ts entry point
- Wrote 10 unit tests (6 wave orchestrator + 4 worker dispatch) all passing

## Task Commits

1. **Task 1 + 2: Create wave orchestrator and wire into worker dispatch** - `a1d85361` (feat)
2. **Task 3: Wire real dispatch into lifecycle with graceful simulation fallback** - `c3364ac5` (feat)
3. **Task 4: Add --simulate CLI flag to host entry point** - `ea7b89bf` (feat)
4. **Task 5: Add wave orchestrator and updated worker dispatch unit tests** - `f6a66eb2` (test)

## Files Created/Modified

- `.aether/ts-host/src/wave-orchestrator.ts` - Wave grouping, parallel dispatch, retry with backoff, timeout options
- `.aether/ts-host/src/worker-dispatch.ts` - Updated DispatchOptions with orchestrator fields; dispatchWorkers delegates to dispatchWaves
- `.aether/ts-host/src/lifecycle.ts` - Platform detection before dispatch; defaults simulateWorkers to false; emits wave ceremony events
- `.aether/ts-host/src/host.ts` - Parses --simulate flag; passes to lifecycle options; updates usage text
- `.aether/ts-host/test/wave-orchestrator.test.ts` - 6 unit tests for parallel, sequential, retry, limit, grouping, failures
- `.aether/ts-host/test/worker-dispatch.test.ts` - 4 unit tests for flattening, order, simulation, WorkerResult mapping

## Decisions Made

- Kept retry logic at the orchestration level (wave-orchestrator) rather than inside platform-dispatcher, so timeout enforcement stays at the subprocess spawn level and retry stays at the wave level.
- Used a mutable module-level `_dispatchSingleWorker` reference with `__setDispatchSingleWorker` / `__restoreDispatchSingleWorker` test helpers because ESM exports are read-only and `node:test` does not provide jest-style module mocking.
- Changed `simulateWorkers` default from `true` to `false` in lifecycle, with automatic fallback to simulation when no platform CLI is available. Tests that explicitly pass `simulateWorkers: true` continue to work unchanged.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- ESM read-only exports prevented direct module mocking in tests. Solved by adding `__setDispatchSingleWorker` and `__restoreDispatchSingleWorker` helpers in `wave-orchestrator.ts`.
- Initial `dispatchWave returns failures array` test failed because the retry loop consumed the second mock result for `FAIL-1`, causing the retried result to be the default success. Fixed by adding a `FAIL-1:1` mock result in the test so the retry also fails, matching the expected retry-limit behavior.

## Next Phase Readiness

- Wave orchestration is complete and tested.
- Ready for Phase 115 (Ceremony Narrator) to consume wave events from the event bridge instead of stderr stubs.
- Ready for Phase 116 (TS Host Integration) to wire the host into the full build/continue pipeline.

---
*Phase: 114-real-worker-dispatch*
*Completed: 2026-05-13*
