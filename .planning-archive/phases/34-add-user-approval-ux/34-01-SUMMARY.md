---
phase: 34-add-user-approval-ux
plan: 01
type: execute
subsystem: learning-system
completed: 2026-02-20
duration: 25m
tasks: 3
requirements:
  - PHER-EVOL-03
key-decisions:
  - All proposals displayed (not just threshold-meeting ones) to support threshold override
  - Sequential numbering across all wisdom type groups for simpler selection
  - Unicode circles (●/○) with ASCII [=--] fallback for non-UTF-8 terminals
  - Color auto-detection via TTY check with --no-color override flag
tech-stack:
  added: []
  patterns:
    - Bash case statement handlers for subcommand routing
    - jq for JSON transformation and filtering
    - Index-based loops to avoid subshell variable scoping issues
key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
---

# Phase 34 Plan 01: Proposal Display Function Summary

## Overview

Created the visual foundation for the tick-to-approve UX — the `learning-display-proposals` function that shows users what they're approving before they can select it.

## What Was Built

### 1. `generate-threshold-bar` Helper

A utility function that creates visual threshold progress bars:

```bash
bash .aether/aether-utils.sh generate-threshold-bar 3 5
# Output: {"bar":"●●●○○","count":3,"threshold":5}
```

**Features:**
- Unicode circles (●/○) for UTF-8 terminals
- ASCII fallback `[=---]` for non-UTF-8 (LANG=C)
- Handles edge cases: count > threshold, threshold = 0 (returns "immediate")

### 2. `learning-display-proposals` Function

Displays all promotion proposals in a checkbox-style UI:

```bash
bash .aether/aether-utils.sh learning-display-proposals [observations_file] [--verbose] [--no-color]
```

**Features:**
- Groups proposals by wisdom type with emoji headers (📜 Philosophies, 🧭 Patterns, etc.)
- Sequential numbering across all groups (1, 2, 3...)
- Checkbox format `[ ]` for selection interface
- Visual threshold bars showing progress toward promotion
- Below-threshold warnings (⚠️ below threshold) with color support
- Content truncation to 40 chars (full with --verbose)
- Graceful empty state handling

**Example Output:**
```
🧠 Promotion Proposals
=====================

Select proposals to promote to QUEEN.md wisdom:
(Enter numbers like '1 3 5', or press Enter to defer all)

📜 Philosophies (threshold: 5)
  [ ] 1. "Keep functions small and focused" ●●●●● (5/5)
  [ ] 2. "Test-driven development ensures quality" ●●●○○ (3/5) ⚠️ below threshold

🧭 Patterns (threshold: 3)
  [ ] 3. "Always validate inputs" ●●● (3/3)
  [ ] 4. "Use jq for JSON manipulation" ●●○ (2/3) ⚠️ below threshold

───────────────────────────────────────────────────
```

### 3. Unicode/ASCII Detection & Color Support

- UTF-8 detection via `LANG` and `LC_ALL` environment variables
- Color support auto-detected via `[[ -t 1 ]]` (TTY check)
- `--no-color` flag forces plain output
- Piped output automatically disables colors

## Deviations from Plan

None — plan executed exactly as written.

## Verification Results

| Criterion | Status | Evidence |
|-----------|--------|----------|
| generate_threshold_bar outputs correct format | PASS | `●●●○○` for 3/5 |
| learning-display-proposals shows grouped proposals | PASS | Groups by type with emoji headers |
| Threshold bars display correctly | PASS | Unicode circles and ASCII fallback both work |
| Below-threshold warnings visible | PASS | Shows `⚠️ below threshold` for 2/3, 3/5, 4/5 |
| Empty state handled gracefully | PASS | Shows helpful message, no error |
| UTF-8/ASCII fallback works | PASS | `[==---]` shown when LANG=C |

## Commits

- `b4e949b`: feat(34-01): add generate-threshold-bar helper function
- `b3603b6`: feat(34-01): add learning-display-proposals function

## Next Steps

This plan provides the display foundation. Phase 34 Plan 02 will add:
- User input handling (number selection)
- Batch promotion execution via `queen-promote`
- Deferred proposal storage in `learning-deferred.json`
- Undo functionality after promotion

## Self-Check: PASSED

- [x] All created functions exist and are callable
- [x] All commits exist in git history
- [x] Verification tests pass
- [x] No breaking changes to existing functionality
