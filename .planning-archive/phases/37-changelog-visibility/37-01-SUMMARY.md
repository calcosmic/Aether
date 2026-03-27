---
phase: 37-changelog-visibility
plan: 01
name: Memory Health Metrics Utilities
completed: 2026-02-21
subsystem: aether-utils
requirements:
  - VIS-02
tags:
  - memory-metrics
  - visibility
  - dashboard
  - utilities
dependency_graph:
  requires:
    - 36-03 (midden system)
    - 34-03 (learning approval)
  provides:
    - Data foundation for /ant:status
    - Data foundation for /ant:resume
  affects:
    - .aether/aether-utils.sh
tech_stack:
  added: []
  patterns:
    - jq JSON parsing
    - Shell command dispatch
    - Graceful file missing handling
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh (added 3 functions)
decisions:
  - MIDDEN_DIR replaced with $DATA_DIR/midden for consistency
  - Functions use existing data structures (no new storage)
  - All functions return JSON for downstream consumption
  - Missing files handled gracefully (return 0/null, not error)
metrics:
  duration: 15 minutes
  tasks_completed: 4
  files_modified: 1
  functions_added: 3
  lines_added: 562
---

# Phase 37 Plan 01: Memory Health Metrics Utilities Summary

**Purpose:** Create utility functions that aggregate memory health data from existing sources (QUEEN.md, learning-observations.json, learning-deferred.json, midden.json) to provide the data foundation for visibility features in `/ant:status` and `/ant:resume`.

## What Was Built

Three new utility functions added to `.aether/aether-utils.sh`:

### 1. memory-metrics
Aggregates all four required memory health metrics:
- **Wisdom count**: Total entries in QUEEN.md by category (philosophy, pattern, redirect, stack, decree)
- **Pending promotions**: Observations meeting thresholds not yet in QUEEN.md + deferred proposals
- **Recent failures**: Count and details from midden.json
- **Last activity**: Timestamps for QUEEN.md updates and learning capture

Returns JSON structure with all metrics for display.

### 2. midden-recent-failures
Extracts recent failure entries from midden.json:
- Accepts optional limit parameter (default: 5)
- Filters for type == "failure"
- Sorts by created_at descending (newest first)
- Returns count and failures array

### 3. resume-dashboard
Generates structured dashboard data for `/ant:resume`:
- Current phase, state, and goal from COLONY_STATE.json
- Memory health summary (wisdom count, pending promotions, recent failures)
- Recent decisions (last 5 from memory.decisions)
- Recent events (last 10 from events array)
- Drill-down command reference

## Verification Results

All functions tested and working:
- memory-metrics: Returns valid JSON with all four metric categories
- midden-recent-failures: Returns valid JSON with failures array
- midden-recent-failures with limit: Respects limit parameter
- resume-dashboard: Returns valid JSON with current phase and memory health

## Deviations from Plan

None - plan executed exactly as written.

## Commits

- `f745349`: feat(37-01): add memory health metrics utility functions

## Next Steps

These utility functions provide the data foundation for:
- `/ant:status` command (memory health display)
- `/ant:resume` command (dashboard with current state + memory health)
- Future changelog visibility features

The data is now available; the commands that consume it will be built in subsequent plans.
