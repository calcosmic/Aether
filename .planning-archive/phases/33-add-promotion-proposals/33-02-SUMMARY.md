---
phase: 33-add-promotion-proposals
plan: "02"
subsystem: learning-observation
tags: [promotion, threshold, proposals, queen, wisdom]
dependency_graph:
  requires:
    - learning-observe (33-01)
  provides:
    - learning-check-promotion (new)
  affects:
    - .aether/aether-utils.sh
key_files:
  modified:
    - .aether/aether-utils.sh
metrics:
  duration: "10 minutes"
  completed: "2026-02-20"
  tasks: 2
  files: 1
---

# Phase 33 Plan 02: Add Promotion Proposals - learning-check-promotion Summary

## Overview

Created the `learning-check-promotion` function in aether-utils.sh to check which learnings meet promotion thresholds for QUEEN.md wisdom. This enables the system to propose learnings that have accumulated enough evidence across colonies.

## What Was Built

### learning-check-promotion Function

New function (~50 lines) added to aether-utils.sh after `learning-observe`:

**Arguments:**
- `path_to_observations_file` (optional) - Path to learning-observations.json (default: `$DATA_DIR/learning-observations.json`)

**Thresholds per Wisdom Type (META-01):**
- philosophy: 5 observations
- pattern: 3 observations
- redirect: 2 observations
- stack: 1 observation
- decree: 0 observations (always eligible)

**Features:**
- Reads learning-observations.json
- Filters observations that meet their type's threshold
- Returns proposals in structured JSON format
- Handles edge cases (missing file, empty observations)

**Return Format:**
```json
{
  "proposals": [
    {
      "content": "learning text",
      "wisdom_type": "pattern",
      "observation_count": 3,
      "threshold": 3,
      "colonies": ["colony-a", "colony-b"],
      "ready": true
    }
  ]
}
```

**Integration:**
- Added to commands list in help output
- Added to "Queen Commands" section

## Verification

```bash
# Check all proposals meeting thresholds
bash .aether/aether-utils.sh learning-check-promotion

# Test specific thresholds
# - Philosophy with count 1: correctly excluded (needs 5)
# - Pattern with count 3: correctly included (needs 3)
# - Stack with count 1: correctly included (needs 1)
# - Decree with count 1: correctly included (needs 0)

# Test edge case: missing file
bash .aether/aether-utils.sh learning-check-promotion /nonexistent.json
# Returns: {"proposals":[]}
```

## Requirements Validated

- **META-01:** Thresholds vary by wisdom type (philosophy:5, pattern:3, redirect:2, stack:1, decree:0)
- **OBS-02:** Proposals include observation count and contributing colonies

## Deviations from Plan

### Auto-fixed Issue: macOS Bash Compatibility

**[Rule 3 - Blocking] Fixed associative array usage**
- **Found during:** Task 1 implementation
- **Issue:** Used `declare -A` for thresholds map, but macOS bash 3.2 doesn't support associative arrays
- **Fix:** Moved threshold logic entirely into jq query using `def get_threshold(type)` function
- **Files modified:** `.aether/aether-utils.sh`
- **Commit:** 172727a

## Auth Gates: None

No authentication required for this implementation.

## Self-Check

- [x] learning-check-promotion function exists in aether-utils.sh
- [x] Function is callable from CLI with optional path argument
- [x] Correct thresholds applied per wisdom type
- [x] Returns proper JSON structure with proposals array
- [x] Handles missing/empty observations file (returns empty proposals)
- [x] Philosophy with count 1 correctly excluded (threshold 5)
- [x] Pattern with count 3 correctly included (threshold 3)
- [x] Stack with count 1 correctly included (threshold 1)
- [x] Decree with count 1 correctly included (threshold 0)
- [x] Added to commands list and Queen Commands section
- [x] Committed: 172727a

## Self-Check: PASSED
