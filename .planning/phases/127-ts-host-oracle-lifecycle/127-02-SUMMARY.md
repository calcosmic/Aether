---
phase: 127-ts-host-oracle-lifecycle
plan: 02
subsystem: ts-host-oracle

requires:
  - plan: 127-01
    provides: Oracle types, runOracleLifecycle(), and host.ts wiring

provides:
  - Oracle lifecycle integration into full plan->build->continue sequence
  - host.ts lifecycle command accepts optional Oracle topic
  - Comprehensive unit tests for Oracle lifecycle with mocked Go/worker boundaries

affects:
  - phase 128-swarm-watch-host-bridge
  - phase 129-release-gate

tech-stack:
  added: []

key-files:
  created:
    - .aether/ts-host/test/oracle-lifecycle.test.ts
  modified:
    - .aether/ts-host/src/lifecycle.ts
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/src/oracle-lifecycle.ts

key-decisions:
  - "Test mock injection follows existing project pattern (__set*/__restore* refs)"
  - "Tests mock callGoJSON, writeCompletionFile, and dispatchSingleWorker to avoid real subprocess spawning"
  - "Integration tests verify boundary enforcement: no writes to .aether/data/oracle/ or .aether/oracle/"

requirements-completed:
  - TOL-04
  - TOL-05

# Metrics
duration: 25min
completed: 2026-05-15
---

# Phase 127 Plan 02: TS Host Oracle Lifecycle — Integration & Tests Summary

## Performance

- **Duration:** 25 min
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Integrated Oracle lifecycle as optional pre-plan step in `runLifecycle()`
- Updated `host.ts` lifecycle command to accept optional Oracle topic as second positional argument
- Added test-only mock injection points to `oracle-lifecycle.ts` (`__setCallGoJSON`, `__setDispatchSingleWorker`, `__setWriteCompletionFile`)
- Created comprehensive unit tests covering max iterations, confidence target, worker failure, and boundary enforcement
- All 177 TS tests pass; Go tests pass

## Task Commits

1. **Task 1: Integrate Oracle into full lifecycle** - `4a00e3a5` (feat)
2. **Task 2: Write tests + mock injection** - `56465ee0` (fix + test)

## Files Created/Modified

- `.aether/ts-host/src/lifecycle.ts` - Added optional `runOracle` / `oracleTopic` to LifecycleOptions; Oracle step before plan
- `.aether/ts-host/src/host.ts` - lifecycle command accepts optional Oracle topic positional arg
- `.aether/ts-host/src/oracle-lifecycle.ts` - Added test-only mock injection refs
- `.aether/ts-host/test/oracle-lifecycle.test.ts` - 7 tests covering loop behavior, stop conditions, worker failure, boundary enforcement

## Decisions Made

- Mock injection uses mutable refs + `__set*` / `__restore*` pattern (consistent with lifecycle.ts and wave-orchestrator.ts)
- Tests use synthetic confidence values (70/30) since Go manifest doesn't yet include real current_confidence

## Deviations from Plan

- Plan expected 5 tests; delivered 7 (added `buildOracleWorkerResponse` and `checkStopConditions` unit tests)
- Mock injection points added to oracle-lifecycle.ts to enable unit testing without real Go CLI subprocesses

## Issues Encountered

- TypeScript `exactOptionalPropertyTypes` required careful handling of optional array access in test assertions.
- Initial Plan 127-01 had type mismatch between TS types and Go JSON output — fixed during Phase 126 commit by updating Go commands to use `outputOK` envelope.

## Next Phase Readiness

- Oracle lifecycle fully wired into TS host with tests.
- Ready for Phase 128 (Swarm/Watch Host Bridge) and Phase 129 (Release Gate).

---
*Phase: 127-ts-host-oracle-lifecycle*
*Completed: 2026-05-15*
