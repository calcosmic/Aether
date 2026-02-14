---
phase: 08-build-polish-output-timing-integration
plan: 01
type: execute
subsystem: build-system
tags: [build, timing, foreground-execution, worker-spawn]

dependency_graph:
  requires: [07-core-reliability-state-guards-update-system]
  provides: [Fixed build.md with foreground Task execution]
  affects: [All future ant:build command executions]

tech_stack:
  added: []
  patterns: [Foreground Task execution for synchronous worker completion]

file_tracking:
  key_files_created: []
  key_files_modified:
    - /Users/callumcowie/.claude/commands/ant/build.md

decisions:
  - Removed run_in_background from all worker spawns to fix output timing
  - Updated documentation to reflect foreground execution model
  - Step 5.2, 5.4.1, 5.4.2 now describe parsing returned results instead of blocking waits

metrics:
  duration: "<1 minute"
  completed: 2026-02-14
---

# Phase 8 Plan 1: Fix Build.md Timing - Foreground Execution

## Summary

Fixed misleading output timing in the Aether build command by removing `run_in_background: true` from all worker spawns. The previous implementation showed the spawn plan summary BEFORE workers actually completed because background execution returns immediately. This created a confusing UX where the summary appeared before work was done.

## Changes Made

### Task 1: Step 5.1 - Wave 1 Worker Spawns
**File:** `/Users/callumcowie/.claude/commands/ant/build.md` (line 312)

**Before:**
```
For each Wave 1 task, use Task tool with `subagent_type="general-purpose"` and `run_in_background: true`:
```

**After:**
```
For each Wave 1 task, use Task tool with `subagent_type="general-purpose"`:
```

### Task 2: Remove All run_in_background References

Searched entire file for `run_in_background` occurrences. No additional occurrences found beyond Step 5.1 (already removed in Task 1).

### Task 3: Update Documentation for Foreground Execution

**Step 5.2 - Collect Wave 1 Results (line 463):**
- **Before:** `For each spawned worker, call TaskOutput with block: true to wait for completion:`
- **After:** `For each spawned worker, parse the returned result (foreground execution means workers have already completed):`

**Step 4.5 - Archaeologist Results (line 239):**
- **Before:** `Wait for results (blocking — use TaskOutput with block: true).`
- **After:** `The archaeologist result is available immediately (foreground execution means the Scout has already completed).`

**Step 5.4.1 - Watcher Results (line 586):**
- **Before:** `Call TaskOutput with block: true using the Watcher's task_id:`
- **After:** `Parse the Watcher's returned result (foreground execution means the Watcher has already completed):`

**Step 5.4.2 - Chaos Ant Results (line 662):**
- **Before:** `Call TaskOutput with block: true using the Chaos Ant's task_id:`
- **After:** `Parse the Chaos Ant's returned result (foreground execution means the Chaos Ant has already completed):`

## Verification

All verification criteria from the plan satisfied:

1. ✅ `grep "run_in_background: true" /Users/callumcowie/.claude/commands/ant/build.md` returns no results
2. ✅ Step 5.1, 5.4, and 5.4.2 all use foreground Task execution
3. ✅ Step 5.2 documentation reflects the new execution model
4. ✅ No syntax errors or broken references in build.md

## Success Criteria

- ✅ **BUILD-01 satisfied:** All `run_in_background: true` flags removed from worker spawns
- ✅ **BUILD-02 satisfied:** Output will now display in correct order (spawn → complete → summary)
- ✅ **BUILD-03 satisfied:** Foreground Task calls with natural blocking behavior

## Notes

**File Location Note:** The modified file `/Users/callumcowie/.claude/commands/ant/build.md` is located in the global Claude commands directory (outside the Aether repository). This is the correct location for Claude Code custom commands. Changes were made directly to this file but cannot be committed to the Aether repository since it resides outside the repo boundary.

## Next Phase Readiness

This plan completes the first step of Phase 8 (Build Polish). The build command now has correct output timing with foreground worker execution. Ready to proceed with additional Phase 8 plans for output formatting and integration improvements.
