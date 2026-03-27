---
phase: 33-add-promotion-proposals
plan: "03"
subsystem: learning-promotion
tags: [promotion, proposals, queen, metadata, evolution]
dependency_graph:
  requires:
    - learning-observe (33-01)
    - learning-check-promotion (33-02)
    - queen-promote (existing)
  provides:
    - Integrated promotion pipeline
  affects:
    - .claude/commands/ant/continue.md
    - .aether/aether-utils.sh
key_files:
  modified:
    - .claude/commands/ant/continue.md
    - .aether/aether-utils.sh
metrics:
  duration: "98 seconds"
  completed: "2026-02-20"
  tasks: 3
  files: 2
---

# Phase 33 Plan 03: Integrate Promotion Proposals Summary

## Overview

Integrated the promotion pipeline components into a complete workflow: observe -> check thresholds -> display proposals -> user approves -> promote to QUEEN.md. This completes the wisdom evolution cycle by connecting all the pieces together.

## What Was Built

### 1. continue.md - Promotion Proposals Display (PHER-EVOL-02)

Added Step 2.1.5 to continue.md that:
- Calls `learning-check-promotion` to get proposals meeting thresholds
- Displays proposals in a clear format showing:
  - Wisdom type (pattern, philosophy, redirect, stack, decree)
  - Content preview
  - Observation count vs threshold
  - Contributing colonies
- Uses AskUserQuestion for user approval before promotion

**Display format:**
```
🧠 Promotion Proposals
====================
The following learnings have met promotion thresholds:

[{type}] {content_preview}...
─────────────────────────
Observations: {count}/{threshold}
Contributed by: {colony1}, {colony2}

Approve promotion?
1. Yes, promote all
2. Yes, promote selected
3. No, skip promotion
```

### 2. continue.md - User Approval for Validated Learnings (INT-03)

Modified Step 2.2 to require explicit user approval:
- Displays validated learnings before promotion
- Uses AskUserQuestion with options: promote all, promote selected, or skip
- Only calls `queen-promote` if user explicitly approves
- **Never auto-promotes** - all promotions require user approval

### 3. queen-promote - Threshold Validation (QUEEN-04)

Enhanced queen-promote function with threshold enforcement:
- Validates wisdom_type is one of: philosophy, pattern, redirect, stack, decree
- Checks observation count in learning-observations.json
- Enforces thresholds per type:
  - philosophy: 5 observations
  - pattern: 3 observations
  - redirect: 2 observations
  - stack: 1 observation
  - decree: 0 observations (always allowed)
- Returns error with explanation if threshold not met

### 4. queen-promote - Metadata Tracking (META-02, META-04)

Added QUEEN.md metadata updates:

**META-02 - evolution_log:**
- Appends entry with timestamp, action="promote", wisdom_type, content_hash, colony
- Tracks the complete history of wisdom changes over time

**META-04 - colonies_contributed:**
- Maps content_hash to array of colonies that contributed observations
- Shows which colonies contributed to each piece of wisdom
- Data sourced from learning-observations.json

## Verification

```bash
# Test 1: Verify learning-check-promotion is called from continue.md
grep -n "learning-check-promotion" .claude/commands/ant/continue.md

# Test 2: Verify user approval prompts exist
grep -n "Approve promotion" .claude/commands/ant/continue.md
grep -n "Promote these to QUEEN" .claude/commands/ant/continue.md

# Test 3: Verify queen-promote threshold checking
grep -n "QUEEN-04" .aether/aether-utils.sh

# Test 4: Verify metadata tracking
grep -n "META-02\|META-04\|evolution_log\|colonies_contributed" .aether/aether-utils.sh
```

## Requirements Validated

- **PHER-EVOL-02:** continue.md displays proposals at threshold
- **INT-03:** All promotions require user approval
- **QUEEN-04:** queen-promote enforces type validation and thresholds
- **META-02:** Evolution log tracks wisdom changes over time
- **META-04:** colonies_contributed tracks wisdom origins

## Deviation: None

Plan executed exactly as written. No auto-fixes needed.

## Auth Gates: None

No authentication required for this implementation.

## Self-Check

- [x] continue.md calls learning-check-promotion and displays proposals
- [x] User must approve promotions in continue.md (two locations)
- [x] queen-promote validates wisdom type
- [x] queen-promote checks observation thresholds (QUEEN-04)
- [x] Promotion only proceeds with user approval
- [x] QUEEN.md metadata includes evolution_log entries (META-02)
- [x] QUEEN.md metadata includes colonies_contributed mapping (META-04)
- [x] Committed: 59bcdfe, a4d47a3

## Self-Check: PASSED
