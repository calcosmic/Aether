---
phase: 05
plan: 01
name: State Loading Utility
subsystem: state-management
tags: [bash, state, locking, validation, handoff]

dependency_graph:
  requires: [04-03]
  provides: [05-02, 05-03]
  affects: [all-ant-commands]

tech_stack:
  added: []
  patterns:
    - File locking with flock-based lock acquisition
    - Structured JSON error output
    - State validation on every load
    - Handoff detection for pause/resume

key_files:
  created:
    - .aether/utils/state-loader.sh
    - tests/unit/state-loader.test.js
  modified:
    - .aether/aether-utils.sh

decisions:
  - id: D05-01-001
    description: State loader sources existing utilities (file-lock.sh, error-handler.sh) rather than reimplementing
    rationale: Follows DRY principle, leverages tested infrastructure
  - id: D05-01-002
    description: Handoff file is removed after successful display
    rationale: Handoff is temporary like a pheromone trail that evaporates after delivery
  - id: D05-01-003
    description: Lock is always released on validation failure
    rationale: Prevents lock starvation from corrupted state

metrics:
  duration: "~5 minutes"
  completed: 2026-02-13
---

# Phase 5 Plan 1: State Loading Utility Summary

## One-Liner

Created state-loader.sh with file lock protection, validation, and handoff detection - the foundation for all state-dependent operations.

## What Was Built

### Core Functions

1. **load_colony_state()** - Main loading function:
   - Checks if COLONY_STATE.json exists (errors with E_FILE_NOT_FOUND if not)
   - Acquires file lock using acquire_lock from file-lock.sh
   - Runs validate-state colony via aether-utils.sh
   - If validation fails: releases lock, outputs structured JSON error, returns 1
   - Reads state into LOADED_STATE variable
   - Checks for HANDOFF.md existence
   - Sets HANDOFF_DETECTED and HANDOFF_CONTENT if handoff exists
   - Exports LOADED_STATE, STATE_LOCK_ACQUIRED=true on success

2. **unload_colony_state()** - Cleanup function:
   - Checks if STATE_LOCK_ACQUIRED is true
   - Calls release_lock if needed
   - Unsets STATE_LOCK_ACQUIRED, LOADED_STATE, HANDOFF content

3. **get_handoff_summary()** - Extracts brief summary from handoff:
   - Parses HANDOFF.md for Phase line
   - Returns "Phase X - Name" format for display

4. **display_resumption_context()** - Shows resume message:
   - If handoff detected, outputs: "Resuming: Phase X - Name"
   - Removes handoff file after successful display

### CLI Integration

Added `load-state` and `unload-state` subcommands to aether-utils.sh:
- `load-state`: Sources state-loader.sh and calls load_colony_state
- `unload-state`: Sources state-loader.sh and calls unload_colony_state
- Both output structured JSON with status
- load-state includes handoff detection status when applicable

### Test Coverage

Created comprehensive test suite in tests/unit/state-loader.test.js:
- State loading succeeds with valid COLONY_STATE.json
- State loading fails when COLONY_STATE.json missing
- State loading detects handoff when HANDOFF.md exists
- Validation failure handling with lock release
- Lock acquisition and release verification
- Handoff summary extraction
- Resumption context display and cleanup
- CLI command testing

## Deviations from Plan

None - plan executed exactly as written.

## Key Implementation Details

### Error Handling

Uses error-handler.sh for structured JSON errors:
- E_FILE_NOT_FOUND: COLONY_STATE.json not found
- E_LOCK_FAILED: Failed to acquire state lock
- E_VALIDATION_FAILED: State validation failed

### Lock Management

- Lock acquired before reading state
- Lock released on all exit paths (success, validation failure, errors)
- Uses file-lock.sh trap for cleanup on script exit

### Handoff Pattern

- Handoff file at .aether/HANDOFF.md
- Detected during load, summary extracted
- File removed after display_resumption_context()
- Temporary like a scout ant's trail

## Verification

All tests pass:
```
✔ state-loader.sh can be sourced without errors
✔ load_colony_state function is defined
✔ unload_colony_state function is defined
✔ get_handoff_summary function is defined
✔ display_resumption_context function is defined
✔ load_colony_state succeeds with valid COLONY_STATE.json
✔ unload_colony_state releases lock properly
✔ load_colony_state fails when COLONY_STATE.json is missing
✔ load_colony_state detects handoff when HANDOFF.md exists
✔ get_handoff_summary extracts phase information
✔ display_resumption_context shows resume message and removes handoff
✔ load_colony_state releases lock on validation failure
✔ CLI load-state command returns JSON with loaded status
✔ CLI unload-state command returns JSON with unloaded status
✔ CLI load-state detects handoff and returns summary
```

## Commits

- `23bc726`: feat(05-01): create state-loader.sh with core loading functions
- `38ec121`: feat(05-01): add load-state and unload-state subcommands to aether-utils.sh
- `1523029`: test(05-01): add state loading integration tests

## Next Phase Readiness

This plan provides the foundation for:
- Plan 05-02: Context restoration integration (using state loader in commands)
- Plan 05-03: Spawn tree persistence (loading spawn-tree.txt alongside state)

All ant commands can now reliably load and validate colony state with proper concurrency protection.
