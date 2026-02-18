---
phase: 16-lock-lifecycle-hardening
plan: 01
subsystem: infra
tags: [bash, file-locking, signals, atomic-write, trap]

requires:
  - phase: 15-distribution-chain
    provides: Distribution chain verified; clean codebase baseline before hardening

provides:
  - "Stale lock user confirmation prompt with TTY detection in acquire_lock"
  - "Uniform trap-based EXIT cleanup in all 4 flag commands (flag-add, flag-auto-resolve, flag-resolve, flag-acknowledge)"
  - "atomic_write_from_file backup created before JSON validation (matching atomic_write ordering)"
  - "SIGHUP added to global cleanup trap for SSH disconnect safety"

affects:
  - lock-lifecycle-hardening
  - flag commands
  - atomic-write

tech-stack:
  added: []
  patterns:
    - "acquire -> trap EXIT -> work -> trap - EXIT -> release -> json_ok (uniform lock lifecycle for all flag commands)"
    - "TTY detection via [[ -t 2 ]] before interactive prompts; JSON error to stderr in non-interactive mode"
    - "Lock age checked before PID liveness to guard against PID reuse race"

key-files:
  created: []
  modified:
    - .aether/utils/file-lock.sh
    - .aether/utils/atomic-write.sh
    - .aether/aether-utils.sh

key-decisions:
  - "Stale lock removal requires user confirmation via [y/N] TTY prompt (never auto-remove)"
  - "Non-interactive stale lock detection emits structured JSON error to stderr and returns 1"
  - "release_lock takes no arguments — all callers must drop the flags_file argument"
  - "Lock age check precedes PID check to handle PID reuse on macOS (32768 PID space)"
  - "SIGHUP added to trap list for SSH disconnect safety (low cost, high safety)"
  - "atomic_write_from_file backup ordering fixed to match atomic_write: backup before validate"

patterns-established:
  - "Flag command lock pattern: acquire -> trap 'release_lock 2>/dev/null || true' EXIT -> work -> trap - EXIT -> release_lock 2>/dev/null || true -> json_ok"
  - "Stale lock TTY gate: [[ -t 2 ]] guards interactive prompt; else emit JSON error"

requirements-completed:
  - LOCK-01
  - LOCK-02
  - LOCK-03

duration: 3min
completed: 2026-02-18
---

# Phase 16 Plan 01: Lock Lifecycle Hardening Summary

**Trap-based lock cleanup unified across all 4 flag commands with user-prompted stale lock confirmation and atomic-write backup ordering fix**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-18T17:28:46Z
- **Completed:** 2026-02-18T17:31:37Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Replaced silent stale lock auto-removal with TTY-gated `[y/N]` prompt; non-interactive contexts get a structured JSON error and return 1
- Added lock age detection (before PID check) to guard against PID reuse race on macOS
- Standardized all four flag commands (flag-add, flag-auto-resolve, flag-resolve, flag-acknowledge) to identical trap pattern — removing local `lock_acquired` variables and inconsistent manual release calls
- Added EXIT trap to flag-resolve and flag-acknowledge (they previously had none)
- Fixed `atomic_write_from_file` to create backup before JSON validation, matching `atomic_write` ordering (LOCK-03)
- Added SIGHUP to global cleanup trap for SSH disconnect safety

## Task Commits

Each task was committed atomically:

1. **Task 1: Stale lock user confirmation and HUP trap** - `13b229b` (fix)
2. **Task 2: Standardize trap-based lock cleanup in all 4 flag commands** - `709ea49` (fix)
3. **Task 3: Fix atomic_write_from_file backup ordering (LOCK-03)** - `eeb6ec7` (fix)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `.aether/utils/file-lock.sh` - Added TTY-gated stale lock prompt, lock age detection, HUP trap
- `.aether/aether-utils.sh` - Uniform trap pattern in all 4 flag commands; removed local lock_acquired variables
- `.aether/utils/atomic-write.sh` - Moved backup creation before JSON validation in atomic_write_from_file

## Decisions Made

- Stale lock prompt reads from `/dev/tty` (not stdin) so it works even when stdout/stdin are piped
- Non-interactive detection uses `[[ -t 2 ]]` (stderr TTY) not `[[ -t 0 ]]` (stdin TTY) — matches the guidance in research
- Lock age computed as `$(date +%s) - stat_mtime` with macOS/Linux `stat` platform detection
- `release_lock` takes no arguments — removed all `release_lock "$flags_file"` calls (argument was silently ignored anyway)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The lint:sync warning (Claude Code has 34 commands, OpenCode has 33) is a pre-existing mismatch unrelated to this plan — deferred.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Lock lifecycle hardening complete for all flag commands and atomic-write
- LOCK-01, LOCK-02, LOCK-03 requirements satisfied
- LOCK-04 (context-update locking) is a separate plan — context-update already has locking at line 216-219 from a prior fix; whether that's sufficient is for the next plan to assess

---
*Phase: 16-lock-lifecycle-hardening*
*Completed: 2026-02-18*

## Self-Check: PASSED

- FOUND: `.aether/utils/file-lock.sh`
- FOUND: `.aether/utils/atomic-write.sh`
- FOUND: `.planning/phases/16-lock-lifecycle-hardening/16-01-SUMMARY.md`
- FOUND commit: `13b229b` (Task 1)
- FOUND commit: `709ea49` (Task 2)
- FOUND commit: `eeb6ec7` (Task 3)
