---
phase: 92-system-hardening-validation
plan: 01
subsystem: worker-lifecycle
tags: [heartbeat, goroutine, liveness, file-based, emitVisualProgress, worker-monitoring]

# Dependency graph
requires: []
provides:
  - HeartbeatFile struct for worker liveness signals
  - StartHeartbeatMonitor goroutine for background staleness detection
  - scanHeartbeatFiles for heartbeat directory scanning
  - cleanupHeartbeatFiles for per-worker heartbeat cleanup
  - cleanupAllHeartbeatFiles for build-completion cleanup
  - Heartbeat Protocol section in worker briefs
  - Monitor lifecycle integration in executeCodexBuildDispatches
affects: [92-02, 92-03, 92-04, worker-cleanup, build-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns: [file-based-liveness-signal, background-goroutine-monitor, visual-staleness-warning, heartbeat-prompt-instruction]

key-files:
  created:
    - cmd/heartbeat_monitor.go
    - cmd/heartbeat_monitor_test.go
  modified:
    - cmd/codex_build.go
    - cmd/codex_build_test.go

key-decisions:
  - "90s stale threshold chosen as 3x the ~30s write interval (per RESEARCH recommendation)"
  - "15s scan interval balances detection latency against overhead"
  - "Heartbeat prompt embedded in worker brief rather than agent definition for phase-specific context"
  - "Monitor cleanup deferred alongside dispatch cleanup for guaranteed resource release"

patterns-established:
  - "File-based heartbeat: workers write heartbeat-{name}.json, runtime scans for staleness"
  - "Visual staleness warning: stale workers reported via emitVisualProgress for runtime UX"
  - "Prompt-driven liveness: heartbeat write instruction included in worker task brief"

requirements-completed: [PLAT-03]

# Metrics
duration: 10min
completed: 2026-05-02
---

# Phase 92 Plan 01: Worker Heartbeat Liveness Detection Summary

**File-based worker heartbeat monitoring with 15s scan interval, 90s staleness threshold, and prompt-driven liveness instructions in worker briefs**

## Performance

- **Duration:** 10 min
- **Started:** 2026-05-02T14:06:10Z
- **Completed:** 2026-05-02T14:16:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Heartbeat monitor goroutine scans `.aether/data/heartbeat-*.json` files every 15 seconds
- Stale workers (no heartbeat update for 90+ seconds) trigger visual warnings via emitVisualProgress
- Worker briefs include Heartbeat Protocol section with worker-specific file paths and JSON templates
- Monitor lifecycle tied to dispatch: starts at dispatch, cleanup deferred at completion
- All heartbeat files cleaned up after build dispatch finishes

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Failing tests for heartbeat monitor** - `4adead11` (test) - TDD RED gate
2. **Task 1 (GREEN): Heartbeat monitor implementation** - `61d35687` (feat) - TDD GREEN gate
3. **Task 2: Heartbeat prompt and lifecycle integration** - `8e91e9da` (feat) - Worker brief + monitor lifecycle

## Files Created/Modified
- `cmd/heartbeat_monitor.go` - HeartbeatFile struct, StartHeartbeatMonitor, scanHeartbeatFiles, cleanupHeartbeatFiles, cleanupAllHeartbeatFiles, formatDuration
- `cmd/heartbeat_monitor_test.go` - 9 unit tests covering marshal/unmarshal, fresh/stale detection, non-heartbeat skip, malformed JSON skip, worker cleanup, nonexistent cleanup, cancel stop, cleanup-all
- `cmd/codex_build.go` - Heartbeat Protocol section in renderCodexBuildWorkerBrief, StartHeartbeatMonitor/cleanupAllHeartbeatFiles lifecycle in executeCodexBuildDispatches
- `cmd/codex_build_test.go` - TestBuildWorkerBriefContainsHeartbeat, TestBuildDispatchStartsHeartbeatMonitor

## Decisions Made
- 90s stale threshold as 3x the ~30s write interval per RESEARCH open question recommendation
- Heartbeat prompt in worker brief (not agent definition) because worker name and phase are dispatch-specific
- Malformed JSON skipped silently per threat model T-92-01 mitigation (filepath.Base for path traversal)
- cleanupAllHeartbeatFiles uses defer alongside monitorCancel for guaranteed cleanup even on errors

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added AETHER_FORCE_VISUAL env var to tests requiring emitVisualProgress output**
- **Found during:** Task 1 (GREEN phase - heartbeat monitor tests)
- **Issue:** Tests that verify visual output from scanHeartbeatFiles failed because stdout is a *strings.Builder in tests, which fails shouldRenderVisualOutput's terminal check
- **Fix:** Added `t.Setenv("AETHER_FORCE_VISUAL", "1")` to tests that rely on emitVisualProgress output
- **Files modified:** cmd/heartbeat_monitor_test.go
- **Verification:** All 9 heartbeat tests pass with visual output captured
- **Committed in:** 61d35687 (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor adjustment to test infrastructure. No scope creep.

## Issues Encountered
- Pre-existing build errors in cmd/codex_worker_cleanup_test.go, cmd/context_freshness_test.go, and cmd/colony_prime_audit_test.go prevent `go test ./cmd/...` from compiling. These are from other parallel plans and out of scope. Verified all plan-specific tests pass by temporarily excluding broken files.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Heartbeat monitor infrastructure complete and ready for integration with worker cleanup path (cleanupHeartbeatFiles per-worker)
- Build dispatch lifecycle integration complete
- Ready for Plans 92-02, 92-03, 92-04 to build on the heartbeat and worker lifecycle foundation

---
*Phase: 92-system-hardening-validation*
*Completed: 2026-05-02*

## Self-Check: PASSED

All created files verified present. All 3 commit hashes verified in git log.
