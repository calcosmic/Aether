---
phase: 75-intelligence-core
plan: 02
subsystem: parallel-dispatch
tags: [circuit-breaker, goroutine-safety, worker-dispatch, failure-protection]

# Dependency graph
requires: []
provides:
  - CircuitBreaker struct with Allow/RecordSuccess/RecordFailure/Reset/TrippedWorkers
  - Per-worker-instance consecutive failure tracking
  - Per-wave breaker reset
  - Same-caste peer redistribution on trip
  - --circuit-breaker-threshold CLI flag (default 3)
affects: [build-dispatch, parallel-execution, worker-failure-handling]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Circuit breaker pattern with sync.Mutex for goroutine-safe state"
    - "Per-worker-instance failure isolation"
    - "Per-wave reset for fresh chances"

key-files:
  created:
    - cmd/circuit_breaker.go
    - cmd/circuit_breaker_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_worktree.go
    - cmd/codex_workflow_cmds.go

key-decisions:
  - "In-memory breaker state (no persistence) -- per-wave reset means no persistence needed"
  - "Threshold clamped to minimum 3 when value < 1"
  - "Emit printf-based ceremony events for breaker trip/redistribution/no-peer"
  - "Redistribution to first available non-tripped same-caste peer (deterministic selection order)"

patterns-established:
  - "Circuit breaker: consecutive failure counter with configurable threshold, per-worker granularity"
  - "Peer redistribution: findSameCastePeer scans dispatches for first non-tripped same-caste worker"

requirements-completed: [INTEL-05]

# Metrics
duration: 12min
completed: 2026-04-29
---

# Phase 75 Plan 02: Circuit Breaker Summary

**Per-worker circuit breaker with consecutive failure tracking, per-wave reset, and same-caste peer redistribution**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-29T16:08:16Z
- **Completed:** 2026-04-29T16:20:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- CircuitBreaker struct with goroutine-safe methods (sync.Mutex on all state access)
- Integration into both in-repo (serial) and worktree (parallel) dispatch paths
- Task redistribution to same-caste peers when a worker trips
- Graceful failure handling when no peer is available
- Configurable threshold via --circuit-breaker-threshold flag (default 3)
- 8 unit tests covering trip, reset, success, concurrency, peer selection, and edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1: Create CircuitBreaker struct with unit tests** - `60f73a82` (test), `5ceb45c4` (feat)
2. **Task 2: Integrate circuit breaker into worker dispatch** - `26fc84cc` (feat)

_Note: TDD RED/GREEN cycle for Task 1 with test commit then implementation commit_

## Files Created/Modified
- `cmd/circuit_breaker.go` - CircuitBreaker struct with Allow, RecordSuccess, RecordFailure, Reset, FailureCount, TrippedWorkers, findSameCastePeer, and ceremony emission helpers
- `cmd/circuit_breaker_test.go` - 8 tests: trip, success reset, wave reset, tripped workers listing, concurrent access (-race), custom threshold, peer selection, partial trip+reset
- `cmd/codex_build.go` - Added CircuitBreakerThreshold to codexBuildOptions, created breaker in executeCodexBuildDispatches, passed to dispatchCodexBuildWorkers
- `cmd/codex_build_worktree.go` - Added cb parameter to both dispatch functions, cb.Allow check before worker invocation, cb.RecordSuccess/RecordFailure after results, cb.Reset at wave boundaries
- `cmd/codex_workflow_cmds.go` - Added --circuit-breaker-threshold flag (default 3) to build command

## Decisions Made
- In-memory breaker state -- per D-06, per-wave reset means no persistence needed. Simpler and correct.
- Threshold clamped to 3 when value < 1 -- prevents misconfiguration from disabling the breaker
- Printf-based ceremony events -- consistent with existing emitBuildCeremony* pattern
- First-match peer selection -- deterministic, simple, avoids complexity of round-robin or random selection

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- 6 pre-existing test failures in cmd package (TestContinueEmitsLifecycleCeremonyEvents, TestContinueBlocksWhenWatcherUsesFakeInvoker, TestClaudeOpenCodeCommandParity, TestIntegrityDetectSourceContext, TestLifecycleCommandDocsPreferRuntimeCLI, TestQueenWisdomHygiene) -- confirmed these fail identically on the base commit before any changes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Circuit breaker fully integrated into build dispatch for both parallel modes
- No state persistence needed (per-wave reset, in-memory only)
- Ready for any subsequent intelligence core work

## Self-Check: PASSED

- FOUND: cmd/circuit_breaker.go
- FOUND: cmd/circuit_breaker_test.go
- FOUND: 60f73a82 (test commit)
- FOUND: 5ceb45c4 (feat commit)
- FOUND: 26fc84cc (feat commit)
- All 8 circuit breaker tests pass with -race flag
- go build ./cmd/ succeeds
- grep verification checks pass (2x cb.Allow, 2x cb.Reset, 1x NewCircuitBreaker, 1x circuit-breaker-threshold flag)

---
*Phase: 75-intelligence-core*
*Completed: 2026-04-29*
