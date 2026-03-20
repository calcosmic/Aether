---
phase: 32-wire-queen-md-into-commands
plan: "02"
subsystem: colony-priming
tags: [wisdom, pheromones, worker-priming, build-commands]
dependency_graph:
  requires:
    - colony-prime (from 32-01)
  provides:
    - Unified worker context in build.md
  affects:
    - .claude/commands/ant/build.md
tech_stack:
  added: []
  patterns:
    - Single colony-prime() call replaces three separate calls
    - Unified prompt_section contains wisdom + signals
key_files:
  created: []
  modified:
    - .claude/commands/ant/build.md
decisions:
  - "build.md uses colony-prime() for unified worker context"
  - "Single call replaces pheromone-prime + queen-read + pheromone-read"
  - "prompt_section contains full formatted context for workers"
  - "FAIL HARD if QUEEN.md missing (handled by colony-prime)"
metrics:
  duration: "5 minutes"
  completed: "2026-02-20"
  tasks: 1
  files: 1
---

# Phase 32 Plan 02: Wire colony-prime into build.md Summary

## Overview

Updated build.md to use single `colony-prime()` call instead of three separate calls (pheromone-prime, queen-read, pheromone-read). Workers now receive unified context containing wisdom + pheromones + instincts.

## What Was Done

### Consolidated Worker Context Loading
- **Before:** Three separate calls in build.md:
  - Step 4: pheromone-prime (signals + instincts)
  - Step 4.1: queen-read (wisdom)
  - Step 4.1.6: pheromone-read (duplicate signals)

- **After:** Single unified call:
  - Step 4: colony-prime (combines wisdom + signals + instincts)

### Updated Variable Handling
- Removed separate `pheromone_section` and `queen_wisdom_section` variables
- Workers now receive unified `prompt_section` from colony-prime
- Removed duplicate Queen Wisdom Section Template
- Removed duplicate Active Signals Section Template

### Error Handling
- Per locked decisions: FAIL HARD if QUEEN.md missing (colony-prime handles this)
- Per locked decisions: WARN but continue if pheromones.json missing

## Verification

```bash
# Verify colony-prime is called
grep -n "colony-prime" .claude/commands/ant/build.md

# Verify old calls removed (only docs remain)
grep -c "pheromone-prime\|queen-read\|pheromone-read" .claude/commands/ant/build.md
# Expected: 3 (documentation hints only)

# Verify workers use prompt_section
grep "prompt_section" .claude/commands/ant/build.md
```

## Deviation: None

Plan executed exactly as written. No auto-fixes needed.

## Auth Gates: None

No authentication required for this implementation.

## Self-Check

- [x] Single colony-prime() call in Step 4
- [x] No remaining pheromone-prime, queen-read, or pheromone-read calls (only docs)
- [x] Workers receive unified prompt_section
- [x] Removed duplicate context injection templates
- [x] Committed: 7bbe639

## Self-Check: PASSED
