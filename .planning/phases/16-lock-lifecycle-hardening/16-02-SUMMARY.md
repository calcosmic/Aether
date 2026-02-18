---
phase: 16-lock-lifecycle-hardening
plan: 02
subsystem: infra
tags: [file-locking, concurrency, shell, context-update]

requires:
  - phase: 16-lock-lifecycle-hardening/16-01
    provides: acquire_lock/release_lock utilities and feature_enabled("file_locking") guard

provides:
  - lock-wrapped context-update function (all 11 actions protected)
  - force-unlock subcommand for emergency lock recovery

affects:
  - colony command workflows that call context-update concurrently (swarm, build, worker-spawn)

tech-stack:
  added: []
  patterns:
    - "_ctx_lock_held local variable as primary gate for explicit release"
    - "EXIT trap as permanent safety net (not cleared on success path, function returns not exits)"
    - "feature_enabled guard for graceful no-op when locking unavailable"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh

key-decisions:
  - "Use _ctx_lock_held local variable as primary release gate — EXIT trap is safety net only, not primary mechanism"
  - "Do not trap - EXIT after release — function returns not exits, clearing trap removes safety net without benefit"
  - "force-unlock requires --yes in non-interactive mode — safe default prevents accidental lock clearing in scripts"
  - "lock_count uses grep -c '\.lock$' to count only primary lock files, not .pid sidecar files"

patterns-established:
  - "acquire/release pair with local _held variable: matches pattern used in flag-add/flag-resolve"
  - "EXIT trap stays permanently active as secondary safety net for the process lifetime"

requirements-completed:
  - LOCK-04

duration: 4min
completed: 2026-02-18
---

# Phase 16 Plan 02: Context-Update Locking + Force-Unlock Summary

**File-lock wrapping for all 11 context-update actions via single acquire/release pair, plus force-unlock escape hatch subcommand for stuck-lock recovery**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-18T17:29:00Z
- **Completed:** 2026-02-18T17:32:42Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- All context-update operations (init, update-phase, activity, constraint, decision, safe-to-clear, build-start, worker-spawn, worker-complete, build-progress, build-complete) now acquire a single lock on CONTEXT.md before any file modifications
- Concurrent context-update calls wait for the lock rather than racing to write the same file, preventing corruption (GAP-009)
- force-unlock subcommand added: lists and removes all .lock and .lock.pid files from .aether/locks/, requires --yes in non-interactive mode, prompts interactively when stderr is a TTY

## Task Commits

Each task was committed atomically:

1. **Task 1: Add file locking to context-update** - `f72f92b` (feat)
2. **Task 2: Add force-unlock subcommand** - `8586bf7` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `.aether/aether-utils.sh` - lock acquisition block added to `_cmd_context_update()` after empty-action check; release block added after closing esac; force-unlock case handler added before catch-all *); force-unlock added to help commands array

## Decisions Made

- `_ctx_lock_held` local variable is the primary release gate. The EXIT trap is set immediately after `acquire_lock` as a safety net but is NOT cleared on the success path. This is intentional: `_cmd_context_update` RETURNs (doesn't call `exit`), so `trap - EXIT` would remove the safety net without benefit — the trap fires on process exit, not function return.
- `force-unlock` requires `--yes` flag in non-interactive mode (stderr not a TTY) to prevent silent lock clearing in automated scripts.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- LOCK-04 (context-update concurrency) is addressed
- force-unlock provides the user recovery escape hatch described in the plan
- Phase 16 continues with plans 03+ (remaining lock lifecycle hardening gaps if any)

---
*Phase: 16-lock-lifecycle-hardening*
*Completed: 2026-02-18*
