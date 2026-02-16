---
phase: 07-core-reliability-state-guards-update-system
plan: 02
type: execute
subsystem: state-management
tags: [state-guard, iron-law, file-lock, idempotency, testing]

dependency_graph:
  requires:
    - 06-01 (Testing Infrastructure)
    - 06-02 (Checkpoint System)
    - 07-RESEARCH (State Guards Research)
  provides:
    - StateGuard class with Iron Law enforcement
    - FileLock with stale detection
    - StateGuardError structured errors
  affects:
    - 07-03 (Update Transaction)
    - 07-04 (Integration)

tech-stack:
  added: []
  patterns:
    - Iron Law enforcement (verification evidence required)
    - PID-based file locking with stale detection
    - Atomic file writes (temp + rename)
    - Structured error with recovery info

key-files:
  created:
    - bin/lib/state-guard.js (532 lines)
    - tests/unit/state-guard.test.js (521 lines)
  modified:
    - tests/unit/helpers/mock-fs.js (added openSync, closeSync, renameSync stubs)

decisions:
  - StateGuardError extends Error with code, details, recovery, timestamp
  - FileLock uses atomic openSync with 'wx' flag for exclusive creation
  - Iron Law requires: checkpoint_hash, test_results, timestamp (all fresh)
  - Idempotency prevents rebuilding completed phases AND skipping phases
  - Phase transitions must be sequential (n -> n+1 only)
  - Lock always released in finally block

metrics:
  duration: 326s
  completed: 2026-02-14
---

# Phase 7 Plan 2: StateGuard — Iron Law Enforcement Summary

**One-liner:** StateGuard class enforcing Iron Law with fresh verification evidence requirement, idempotency checks, and PID-based file locking.

## What Was Built

### StateGuard Module (`bin/lib/state-guard.js`)

A comprehensive state guard system with three main components:

1. **StateGuardError Class** (lines 29-78)
   - Structured error with code, message, details, recovery suggestion
   - `toJSON()` for structured output
   - `toString()` for display
   - Error codes: E_IRON_LAW_VIOLATION, E_IDEMPOTENCY_CHECK, E_LOCK_TIMEOUT, E_INVALID_TRANSITION, E_STATE_NOT_FOUND, E_STATE_INVALID

2. **FileLock Class** (lines 83-203)
   - PID-based file locking with stale lock detection
   - Automatic cleanup of stale locks (dead process detection)
   - Configurable retry with exponential backoff
   - Uses atomic `openSync` with 'wx' flag

3. **StateGuard Class** (lines 208-532)
   - `acquireLock()` / `releaseLock()` - Lock management
   - `loadState()` / `saveState()` - State persistence with atomic writes
   - `hasFreshEvidence()` - Validates evidence has required fields and is fresh
   - `enforceIronLaw()` - Throws if evidence missing/stale
   - `checkIdempotency()` - Prevents rebuilding completed phases or skipping
   - `validateTransition()` - Enforces sequential transitions only
   - `transitionState()` - Updates phase and adds audit event
   - `advancePhase()` - Full guarded phase advancement with lock

### Unit Tests (`tests/unit/state-guard.test.js`)

18 comprehensive tests covering:

| Test | Description |
|------|-------------|
| advancePhase succeeds | Valid evidence allows transition |
| advancePhase throws without evidence | Iron Law enforced (E_IRON_LAW_VIOLATION) |
| advancePhase throws with stale evidence | Timestamp before init rejected |
| Idempotency prevents rebuild | Already complete phases skipped |
| Idempotency prevents skip | Previous incomplete phases blocked |
| Sequential transitions only | Non-sequential throws E_INVALID_TRANSITION |
| Lock released on error | finally block ensures cleanup |
| hasFreshEvidence validation | Missing fields return false |
| StateGuardError toJSON | Structured error format |
| StateGuardError toString | Human-readable format |
| loadState validation | Missing version/phase/events rejected |
| loadState missing file | E_STATE_NOT_FOUND thrown |
| loadState invalid JSON | E_STATE_INVALID thrown |
| acquireLock timeout | E_LOCK_TIMEOUT thrown |
| saveState atomic | Writes to temp, then renames |
| transitionState audit | Event added to events array |
| Invalid timestamp | Non-ISO timestamps rejected |
| releaseLock safe | No-op when not locked |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added missing fs stubs to mock-fs helper**

- **Found during:** Task 3 (creating tests)
- **Issue:** `openSync`, `closeSync`, `renameSync` stubs missing from mock-fs helper
- **Fix:** Added three new stubs to `createMockFs()` function
- **Files modified:** `tests/unit/helpers/mock-fs.js`
- **Commit:** 1adb0b9

**2. [Rule 1 - Bug] Fixed resetMockFs to use resetBehavior**

- **Found during:** Task 3 (test debugging)
- **Issue:** `resetMockFs` used `stub.reset()` which doesn't clear `callsFake` behaviors
- **Fix:** Changed to `stub.resetBehavior()` for proper cleanup between tests
- **Files modified:** `tests/unit/helpers/mock-fs.js`
- **Commit:** 1adb0b9

### None - Plan executed as specified

All other aspects of the plan were implemented exactly as written.

## Verification Results

```
✔ All 18 StateGuard unit tests pass
✔ StateGuard class loads without errors
✔ Iron Law enforcement verified (no advancement without evidence)
✔ Idempotency checks verified (completed phases skipped)
✔ Lock acquisition/releases verified
```

## Key Design Decisions

1. **Iron Law Evidence Requirements:**
   - `checkpoint_hash`: SHA-256 hash of checkpoint
   - `test_results`: boolean or object with test outcomes
   - `timestamp`: ISO 8601 date string, must be after state initialization

2. **Idempotency Behavior:**
   - If `current_phase > fromPhase`: Return `{ status: 'already_complete' }`
   - If `current_phase < fromPhase`: Throw `E_IDEMPOTENCY_CHECK`
   - If `current_phase === fromPhase`: Proceed with transition

3. **Lock Safety:**
   - Always acquire lock BEFORE reading state
   - Always release lock in `finally` block
   - Stale lock detection via PID checking

4. **Atomic Writes:**
   - Write to `.tmp` file first
   - Rename to target file (atomic on POSIX)
   - Prevents partial writes on crash

## Next Phase Readiness

This plan provides the foundation for:

- **07-03 (Update Transaction):** Can use StateGuard for phase advancement during updates
- **07-04 (Integration):** StateGuard can be integrated into CLI commands
- **Future phases:** Iron Law enforcement prevents phase advancement loops

## Commits

| Commit | Message |
|--------|---------|
| d1c6649 | feat(07-02): create StateGuardError class and StateGuard foundation |
| 100a018 | feat(07-02): implement Iron Law enforcement and idempotency checks |
| 1adb0b9 | test(07-02): add comprehensive unit tests for StateGuard |

## Time Tracking

- **Started:** 2026-02-14T02:02:21Z
- **Completed:** 2026-02-14T02:07:47Z
- **Duration:** 326 seconds (~5.5 minutes)
