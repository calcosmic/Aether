---
phase: 75-intelligence-core
plan: 01
subsystem: memory, parallel-dispatch
tags: [circuit-breaker, trust-scoring, go, bayesian, memory-capture, continue-ceremony]

# Dependency graph
requires:
  - phase: 74-suggest-analyze
    provides: suggest-analyze pheromone pipeline (companion intelligence feature)
provides:
  - memory-capture --source-type and --evidence-type flags for trust scoring
  - CircuitBreaker struct with per-worker-instance failure tracking
  - Circuit breaker integration in both in-repo and worktree dispatch paths
  - CeremonyTopicBuildCircuitBreak event for build output visibility
affects: [76+, continue-ceremony, build-ceremony, parallel-dispatch]

# Tech tracking
tech-stack:
  added: []
  patterns: [circuit-breaker-per-worker-instance, per-wave-reset, same-caste-peer-redistribution]

key-files:
  created:
    - cmd/circuit_breaker.go
    - cmd/circuit_breaker_test.go
  modified:
    - cmd/learning.go
    - cmd/codex_build_worktree.go
    - cmd/ceremony_emitter.go
    - pkg/events/ceremony.go
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-full.md

key-decisions:
  - "In-memory circuit breaker (no persistence) -- per-wave reset means no state survives between waves"
  - "CircuitBreaker struct in cmd/ package (not pkg/) -- small enough to be co-located with dispatch code"
  - "Default threshold of 3 consecutive failures -- matches research recommendation D-04"
  - "Error resolution source type for midden captures -- error_resolution/multi_phase scores higher than defaults"
  - "Wave-end summary event emitted when any workers tripped -- gives visibility without noise"

patterns-established:
  - "Circuit breaker pattern: consecutive failure counter with per-wave reset and same-caste peer redistribution"
  - "Trust scoring flag wiring: --source-type and --evidence-type on CLI commands, playbook-driven values"

requirements-completed: [INTEL-04, INTEL-05]

# Metrics
duration: 12min
completed: 2026-04-29
---

# Phase 75 Plan 01: Intelligence Core Summary

**Bayesian trust scoring wired into memory-capture and continue ceremony, plus circuit breaker preventing cascade failures in parallel worker dispatch**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-29T16:08:13Z
- **Completed:** 2026-04-29T16:20:58Z
- **Tasks:** 5
- **Files modified:** 8

## Accomplishments
- `memory-capture` now accepts `--source-type` and `--evidence-type` flags, producing meaningful trust scores instead of defaulting to observation/anecdotal
- Continue ceremony playbooks pass explicit trust flags: `success_pattern`/`multi_phase` for learnings, `error_resolution`/`multi_phase` for midden error patterns
- Circuit breaker tracks consecutive failures per worker instance, trips at threshold (default 3), redistributes tasks to same-caste peers
- Both in-repo and worktree parallel dispatch modes protected by circuit breaker
- Wave-end summary emits ceremony event listing tripped workers

## Task Commits

Each task was committed atomically:

1. **Task 1: Add trust scoring flags to memory-capture command** - `392474ea` (feat)
2. **Task 2: Wire trust scoring flags into continue ceremony playbooks** - `70c73f24` (feat)
3. **Task 3: Implement circuit breaker struct with tests** - `3e1e3b1a` (feat)
4. **Task 4: Integrate circuit breaker into parallel worker dispatch** - `a09dbb65` (feat)
5. **Task 5: Add circuit breaker wave-end summary output** - `1998955a` (feat)

## Files Created/Modified
- `cmd/circuit_breaker.go` - CircuitBreaker struct with Allow/RecordSuccess/RecordFailure/Reset/TrippedWorkers API, FindSameCastePeer, CircuitBreakerEvent
- `cmd/circuit_breaker_test.go` - 14 tests covering trip, reset, wave reset, per-worker isolation, peer selection, concurrent access
- `cmd/learning.go` - Added --source-type and --evidence-type flags to memory-capture, switched from Capture() to CaptureWithTrust()
- `cmd/codex_build_worktree.go` - Integrated circuit breaker into both dispatchCodexBuildWorkersInRepo and dispatchCodexBuildWorkers with per-wave reset and peer redistribution
- `cmd/ceremony_emitter.go` - Added emitBuildCeremonyCircuitBreak for circuit breaker event output
- `pkg/events/ceremony.go` - Added CeremonyTopicBuildCircuitBreak topic constant
- `.aether/docs/command-playbooks/continue-advance.md` - Added trust flags to learning and midden memory-capture calls
- `.aether/docs/command-playbooks/continue-full.md` - Added trust flags to learning and midden memory-capture calls

## Decisions Made
- In-memory circuit breaker (no persistence needed -- per-wave reset means state is ephemeral)
- CircuitBreaker struct lives in `cmd/` package co-located with dispatch code rather than a separate `pkg/` package
- Default threshold of 3 consecutive failures (per research recommendation D-04)
- Midden error pattern captures use `error_resolution`/`multi_phase` source/evidence types for higher trust scores
- Wave-end summary only emits when workers were actually tripped (no noise for clean waves)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing .aether/rules/ directory in worktree**
- **Found during:** Task 1 (testing memory-capture changes)
- **Issue:** The `go:embed all:.aether/rules` directive in `embedded_assets.go` failed because `.aether/rules/` directory was missing from the worktree
- **Fix:** Copied `.aether/rules/aether-colony.md` from main repo to worktree
- **Files modified:** `.aether/rules/aether-colony.md` (not committed -- runtime-only fix for worktree)
- **Verification:** `go test ./cmd/` passes after fix

**2. [Rule 1 - Bug] Test assertion logic inverted in TestCircuitBreakerResetOnSuccessAfterTrip**
- **Found during:** Task 3 (running circuit breaker tests)
- **Issue:** Test used `if !cb.Allow()` expecting worker to be blocked, but the error message said "expected worker blocked" -- the condition was inverted (`!` should have been absent)
- **Fix:** Changed `if !cb.Allow()` to `if cb.Allow()` with corrected error message
- **Files modified:** `cmd/circuit_breaker_test.go`
- **Verification:** Test passes after fix
- **Committed in:** `3e1e3b1a` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both necessary for correctness. No scope creep.

## Issues Encountered
- Edit tool had repeated failures matching tab-indented Go code in `codex_build_worktree.go` -- resolved using Python-based string replacement
- `.aether/rules/` not present in worktree (pre-existing issue with git worktree setup) -- worked around by copying from main repo

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- INTEL-04 and INTEL-05 requirements fulfilled
- Trust scoring pipeline fully wired: CLI flags, playbook integration, trust engine unchanged
- Circuit breaker protects both parallel modes
- Ready for subsequent intelligence core features (INTEL-06 through INTEL-09 are deferred to v2)

## Self-Check: PASSED

---
*Phase: 75-intelligence-core*
*Completed: 2026-04-29*
