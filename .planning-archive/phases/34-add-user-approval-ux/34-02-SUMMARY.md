---
phase: 34-add-user-approval-ux
plan: 02
completed: 2026-02-20
duration: 4 minutes
tasks: 3
files_created: 0
files_modified: 1
key-decisions:
  - Used bash 3.x compatible string pattern matching instead of associative arrays for deduplication (macOS compatibility)
  - learning-select-proposals uses same data source as learning-display-proposals (all observations) to maintain index consistency
  - Preview shows full content and wisdom type for each selected item
  - Below-threshold warnings displayed during preview phase
  - --yes flag for scripting, --dry-run for testing
---

# Phase 34 Plan 02: Selection Parsing and Capture - Summary

## What Was Built

Interactive selection system for tick-to-approve UX that converts user input (space-separated numbers) into validated selection indices for batch promotion.

### Functions Added

**1. parse-selection** (lines 3679-3778)
- Parses space-separated numbers (1-indexed) into 0-indexed array indices
- Validates range and warns on invalid numbers (continues, does not fail)
- Deduplicates selections using string pattern matching (bash 3.x compatible)
- Returns `defer_all` action for empty input
- Outputs JSON with `selected`, `deferred`, `count`, `action`, and optional `warnings` arrays

**2. learning-select-proposals** (lines 4480-4588)
- Displays proposals using existing `learning-display-proposals`
- Captures user input with `read -r selection`
- Calls `parse-selection` to validate and convert input
- Shows preview of selected items with full content and wisdom type
- Displays below-threshold warnings during preview
- Confirmation prompt: "Proceed with promotion? (y/n)"
- Outputs JSON with `selected`, `deferred`, `count`, `action`, `confirmed`, and `proposals`

### Flags Supported

| Flag | Purpose |
|------|---------|
| `--verbose` | Passes through to display function for full content |
| `--dry-run` | Selects all proposals without user input, shows what would happen |
| `--yes` | Skips confirmation prompt (for scripting) |

### UX Flow

```
1. Display proposals with checkboxes [ ] 1. Pattern: "Always validate inputs" ●●● (3/3)
2. User enters: "1 3 5"
3. Show summary: "3 proposal(s) selected, 2 deferred"
4. Preview selected items with full content and threshold status
5. Show below-threshold warnings if applicable
6. Prompt: "Proceed with promotion? (y/n)"
7. Output JSON with selected/deferred arrays
```

## Files Modified

- `.aether/aether-utils.sh`: Added `parse-selection` and `learning-select-proposals` functions

## Commits

| Commit | Message |
|--------|---------|
| 9916e73 | feat(34-02): add parse-selection helper function |
| fd63c10 | feat(34-02): add learning-select-proposals function |
| c4c898f | feat(34-02): add selection preview and confirmation |

## Verification Results

All tests passing:
- parse-selection correctly converts 1-indexed to 0-indexed: PASS
- Invalid numbers skipped with warnings: PASS
- Empty input signals defer-all: PASS
- Duplicates are deduplicated: PASS
- learning-select-proposals captures input and outputs JSON: PASS
- Preview shows selected items before confirmation: PASS
- Confirmation prompt works (y proceeds, n defers): PASS
- --yes flag skips confirmation: PASS

## Deviation from Plan

None. Plan executed exactly as written.

## Next Steps

Phase 34-03 will implement the actual promotion execution using `queen-promote` on selected items, and deferred proposal storage in `learning-deferred.json`.
