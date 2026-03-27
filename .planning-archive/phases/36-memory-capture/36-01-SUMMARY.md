---
phase: 36-memory-capture
plan: 01
subsystem: learning-system
tags: [thresholds, promotion, wisdom-types, MEM-03]
dependency_graph:
  requires: []
  provides: [MEM-03]
  affects: [learning-observe, learning-check-promotion, QUEEN.md accumulation]
tech_stack:
  added: []
  patterns: [threshold-based promotion, user approval gate]
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh (lines 4195-4202, 4255-4265)
decisions:
  - "All wisdom types promote after 1 observation (was 5/3/2/1/0)"
  - "User approval remains the quality gate for all promotions"
  - "Keep decree at 0 for immediate promotion without observation"
metrics:
  duration_minutes: 5
  tasks_completed: 3
  files_modified: 1
  lines_changed: 18
  completion_date: 2026-02-21
---

# Phase 36 Plan 01: Lower Promotion Thresholds Summary

## One-Liner
Lowered wisdom promotion thresholds from 5/3/2/1/0 to 1/1/1/1/0 so QUEEN.md accumulates wisdom after each approved phase instead of requiring 5 observations.

## What Was Built

Updated threshold configuration in `.aether/aether-utils.sh` at two locations:

1. **learning-observe function (lines 4195-4202):**
   - philosophy: 5 → 1
   - pattern: 3 → 1
   - redirect: 2 → 1
   - stack: 1 (unchanged)
   - decree: 0 (unchanged)

2. **learning-check-promotion function (lines 4255-4265):**
   - Updated jq `get_threshold` function with same values
   - Updated comment documenting thresholds (META-01)

## Why This Matters

The 5-observation threshold was why QUEEN.md stayed empty. If something is worth capturing once and the user approves it, it is valid wisdom. User approval remains the quality gate — the threshold just controls when the system considers an observation "ripe" for promotion consideration.

## Verification Results

Test observation with philosophy type:
- Threshold: 1 (was 5)
- threshold_met: true after single observation

All wisdom types now promote after 1 observation except decree (0 observations).

## Commits

- `67de004`: feat(36-01): lower promotion thresholds to 1 for all wisdom types

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- [x] learning-observe case statement shows threshold=1 for philosophy, pattern, redirect
- [x] learning-check-promotion jq function shows "then 1" for philosophy, pattern, redirect
- [x] Comment documenting thresholds is updated
- [x] Test observation shows threshold_met=true after single observation
