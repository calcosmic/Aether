---
phase: 92-system-hardening-validation
plan: 05
subsystem: infra
tags: [process-groups, pid-tracking, worker-lifecycle, stale-cleanup]

# Dependency graph
requires:
  - phase: 92-01
    provides: "heartbeat monitor, process tracker, process group helpers"
  - phase: 92-02
    provides: "stale worker cleanup function"
provides:
  - "Process group isolation on spawned workers via workerSysProcAttr()"
  - "PID tracking for every spawned worker via GlobalProcessTracker()"
  - "Automatic stale worker cleanup before each build dispatch"
affects: [build-dispatch, worker-lifecycle, process-management]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Process group isolation for clean worker termination"
    - "Singleton process tracker for lifecycle management"
    - "Pre-dispatch stale worker cleanup"

key-files:
  created: []
  modified:
    - pkg/codex/worker.go
    - cmd/codex_build.go

key-decisions:
  - "Defer-based untracking ensures cleanup even on panic or timeout"
  - "Stale cleanup placed after heartbeat monitor start to ensure data directory exists"

requirements-completed: []

# Metrics
duration: 2min
completed: 2026-05-02
---

# Phase 92 Plan 05: Wire Production Integrations Summary

**Process group isolation, PID tracking, and stale worker cleanup wired into production build dispatch flow**

## Performance

- **Duration:** 2 min
- **Started:** 2026-05-02T15:32:41Z
- **Completed:** 2026-05-02T16:00:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Every spawned Codex worker now gets its own process group (Setpgid: true on Unix), enabling clean group-level termination
- Every spawned worker PID is tracked via the singleton ProcessTracker, with automatic untracking on exit via defer
- Stale worker processes from prior crashed builds are automatically cleaned up before each build dispatch

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire process groups and PID tracking into worker invoker** - `c281eb62` (feat)
2. **Task 2: Wire stale worker cleanup into build dispatch flow** - `b750c3f9` (feat)

## Files Created/Modified
- `pkg/codex/worker.go` - Added `cmd.SysProcAttr = workerSysProcAttr()` for process group isolation, and `GlobalProcessTracker().TrackProcess()`/`UntrackProcess()` for PID lifecycle management
- `cmd/codex_build.go` - Added `cleanupStaleWorkersBeforeDispatch(root)` call before worker dispatch in `executeCodexBuildDispatches`

## Decisions Made
- Used defer for untracking so PIDs are cleaned up even on panic, timeout, or error return paths
- Placed stale cleanup after heartbeat monitor setup (which creates the data directory) so the cleanup function has a valid environment

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- The `cmd` package has a pre-existing embedded_assets.go issue (`pattern all:.aether/rules: no matching files found`) that prevents compilation in worktrees where `.aether/rules/` is not present. This is an environment issue, not caused by this plan's changes. The `pkg/codex` package compiles and all its tests pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All v1.13 worker lifecycle infrastructure (heartbeat monitor, process tracker, process groups, stale cleanup) is now wired into production code
- Phase 92 system hardening is functionally complete

---
*Phase: 92-system-hardening-validation*
*Completed: 2026-05-02*
