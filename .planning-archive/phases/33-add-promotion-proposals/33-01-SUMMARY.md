---
phase: 33-add-promotion-proposals
plan: "01"
subsystem: learning-observation
tags: [observations, promotion, threshold, queen, wisdom]
dependency_graph:
  requires:
    - queen-promote (existing)
    - queen-read (existing)
  provides:
    - learning-observe (new)
  affects:
    - .aether/aether-utils.sh
    - .aether/data/learning-observations.json
key_files:
  created:
    - .aether/data/learning-observations.json (runtime)
  modified:
    - .aether/aether-utils.sh
metrics:
  duration: "15 minutes"
  completed: "2026-02-20"
  tasks: 2
  files: 1
---

# Phase 33 Plan 01: Add Promotion Proposals - learning-observe Summary

## Overview

Created the `learning-observe` function in aether-utils.sh to record observations of learnings across colonies. This enables tracking how many times each learning is observed/used to determine when it meets promotion thresholds for QUEEN.md wisdom.

## What Was Built

### learning-observe Function

New function (~140 lines) added to aether-utils.sh after `queen-promote`:

**Arguments:**
- `content` - The learning text/content to observe
- `wisdom_type` - Type: philosophy, pattern, redirect, stack, decree
- `colony_name` (optional) - Name of observing colony (defaults to "unknown")

**Features:**
- SHA256 content hashing for deduplication (OBS-04)
- Cross-colony accumulation - same content from different colonies increments count (OBS-03)
- File locking for concurrent access safety
- Threshold detection based on wisdom type:
  - philosophy: 5 observations
  - pattern: 3 observations
  - redirect: 2 observations
  - stack: 1 observation
  - decree: 0 observations (immediate)

**Storage Format:**
```json
{
  "observations": [
    {
      "content_hash": "sha256:abc123...",
      "content": "learning text",
      "wisdom_type": "pattern",
      "observation_count": 3,
      "first_seen": "2026-02-20T10:00:00Z",
      "last_seen": "2026-02-20T15:30:00Z",
      "colonies": ["colony-a", "colony-b"]
    }
  ]
}
```

**Return Value:**
```json
{
  "content_hash": "sha256:...",
  "content": "...",
  "wisdom_type": "pattern",
  "observation_count": 3,
  "threshold": 3,
  "threshold_met": true,
  "colonies": ["colony-a", "colony-b"],
  "is_new": false
}
```

## Verification

```bash
# Test new observation
bash .aether/aether-utils.sh learning-observe "Always validate inputs" "pattern" "test-colony"
# Returns: observation_count=1, is_new=true, threshold_met=false

# Test same content from different colony
bash .aether/aether-utils.sh learning-observe "Always validate inputs" "pattern" "other-colony"
# Returns: observation_count=2, colonies=["test-colony","other-colony"]

# Test threshold met (3rd observation for pattern type)
bash .aether/aether-utils.sh learning-observe "Always validate inputs" "pattern" "third-colony"
# Returns: observation_count=3, threshold_met=true
```

## Requirements Validated

- **OBS-01:** Observations accumulate across colonies (not just per-colony)
- **OBS-03:** Cross-colony accumulation works - different colonies add to same observation
- **OBS-04:** Content hashing prevents duplicate entries - same content produces same hash

## Deviation: None

Plan executed exactly as written. No auto-fixes needed.

## Auth Gates: None

No authentication required for this implementation.

## Self-Check

- [x] learning-observe function exists in aether-utils.sh
- [x] Function is callable from CLI with proper arguments
- [x] learning-observations.json created in correct location (.aether/data/)
- [x] Hash deduplication works (same content = same hash)
- [x] Colony accumulation works (different colonies add to list, no duplicates)
- [x] Count increments correctly
- [x] Threshold detection works for all wisdom types
- [x] File locking implemented for concurrent access
- [x] Commands list updated in help output
- [x] Committed: 0fa9f2b

## Self-Check: PASSED
