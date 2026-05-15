---
phase: 128-swarm-watch-host-bridge
plan: 01
subsystem: ts-host
tags: [typescript, go-bridge, status, swarm, dashboard, json]

requires:
  - phase: 127-ts-host-oracle-lifecycle
    provides: callGoJSON pattern, GoBridgeOptions, dashboard.ts, narrator.ts

provides:
  - StatusResult TypeScript type matching Go buildStatusResult JSON output
  - SwarmManifest and SwarmWorkerPlan TypeScript types matching Go swarmManifest
  - watch-display.ts with runWatchDisplay() for colony status rendering
  - swarm-display.ts with runSwarmDisplay() for swarm manifest rendering
  - Dashboard and plain-text output modes for both watch and swarm

affects:
  - Phase 129 (final integration / host command surface completion)
  - Future TS host commands consuming Go JSON output

tech-stack:
  added: []
  patterns:
    - "Go JSON envelope consumption via callGoJSON<T>"
    - "Dashboard mode (TTY) vs plain-text mode (non-TTY) dual rendering"
    - "Progress bar helper reused across display modules"
    - "exactOptionalPropertyTypes compatibility with undefined union types"

key-files:
  created:
    - .aether/ts-host/src/watch-display.ts
    - .aether/ts-host/src/swarm-display.ts
  modified:
    - .aether/ts-host/src/types.ts

key-decisions:
  - "Used exactOptionalPropertyTypes-compatible optional fields (| undefined) to satisfy strict TS config"
  - "Mapped Go omitempty fields to TypeScript optional (?); non-omitempty to required"
  - "Dashboard refresh loop capped at minimum 1000ms per threat model T-128-02"
  - "Plain-text mode is default when stdout is not a TTY or --no-dashboard is set"

patterns-established:
  - "Display modules follow {runXDisplay, renderXText, renderXFrame} naming convention"
  - "All display modules extend GoBridgeOptions and accept dashboard/planOnly flags"
  - "Error handling returns {success:false, error} instead of throwing to caller"

requirements-completed:
  - SWB-01
  - SWB-02

duration: 18min
completed: 2026-05-15
---

# Phase 128 Plan 01: Swarm/Watch Host Bridge Summary

**TS host watch and swarm display modules that consume structured Go JSON output and render live dashboard or plain-text views, completing the major Aether workflow command surface.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-05-15T09:00:00Z
- **Completed:** 2026-05-15T09:18:00Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Added StatusResult, SwarmManifest, SwarmWorkerPlan, PheromoneSummary, and MemoryHealth TypeScript types matching Go JSON output shapes
- Created watch-display.ts with runWatchDisplay(), renderStatusText(), and renderStatusFrame() supporting dashboard refresh loop and plain-text snapshot modes
- Created swarm-display.ts with runSwarmDisplay(), renderSwarmText(), and renderSwarmFrame() supporting wave-grouped worker display
- Both modules use callGoJSON for structured Go communication and follow existing host.ts command patterns
- All code passes npm run typecheck with exactOptionalPropertyTypes enabled

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Status and Swarm TypeScript types** - `909bd6a8` (feat)
2. **Task 2: Create watch-display.ts** - `8ffe18d3` (feat)
3. **Task 3: Create swarm-display.ts** - `4a14bcf7` (feat)

## Files Created/Modified

- `.aether/ts-host/src/types.ts` - Added StatusResult, SwarmManifest, SwarmWorkerPlan, PheromoneSummary, MemoryHealth types
- `.aether/ts-host/src/watch-display.ts` - Watch display module with dashboard and plain-text rendering
- `.aether/ts-host/src/swarm-display.ts` - Swarm display module with wave-grouped rendering

## Decisions Made

- Followed exactOptionalPropertyTypes pattern from oracle-lifecycle.ts: optional fields use `field?: T | undefined` instead of just `field?: T`
- Chose minimum 1000ms refresh interval for watch dashboard to mitigate DoS threat T-128-02
- Kept dashboard one-shot for swarm (no refresh loop) since swarm --plan-only is a static manifest

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- TypeScript exactOptionalPropertyTypes errors on optional interface fields required adding `| undefined` to `final_status` and `error` in WatchDisplayResult, and `manifest` and `error` in SwarmDisplayResult. Fixed inline before committing.
- Go tests ran successfully (17 packages passing, no failures).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Watch and swarm commands are ready to be wired into host.ts command router
- All major Aether workflows (plan, build, continue, oracle, watch, swarm) now have TS host modules
- Phase 129 can focus on final integration, testing, and documentation

---
*Phase: 128-swarm-watch-host-bridge*
*Completed: 2026-05-15*
