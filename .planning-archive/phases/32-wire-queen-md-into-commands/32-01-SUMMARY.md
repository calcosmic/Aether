---
phase: 32-wire-queen-md-into-commands
plan: "01"
subsystem: colony-priming
tags: [wisdom, pheromones, worker-priming, two-level-loading]
dependency_graph:
  requires:
    - queen-read (existing)
    - pheromone-prime (existing)
  provides:
    - colony-prime (new)
  affects:
    - build.md (integration point)
    - init.md (integration point)
tech_stack:
  added:
    - colony-prime function (340+ lines)
    - two-level QUEEN.md loading
  patterns:
    - unified worker context injection
    - fail-hard for missing QUEEN.md
    - warn-but-continue for missing pheromones.json
key_files:
  created: []
  modified:
    - .aether/aether-utils.sh
decisions:
  - "colony-prime() combines queen-read + pheromone-prime into single unified call"
  - "Two-level loading: global ~/.aether/QUEEN.md loads first, local .aether/docs/QUEEN.md extends"
  - "QUEEN.md missing = FAIL HARD with actionable error to run /ant:init"
  - "pheromones.json missing = WARN but continue (workers just don't get signals)"
  - "Categories only to workers: Philosophies, Patterns, Redirects, Stack Wisdom, Decrees"
  - "Metadata and Evolution Log excluded from worker context"
metrics:
  duration: "10 minutes"
  completed: "2026-02-20"
  tasks: 2
  files: 1
---

# Phase 32 Plan 01: Wire QUEEN.md into Commands Summary

## Overview

Implemented unified `colony-prime()` function in aether-utils.sh that combines queen-read (wisdom) and pheromone-prime (signals + instincts) into a single worker context call. Also updated queen-read() to support two-level QUEEN.md loading.

## What Was Built

### colony-prime() Function
- New unified function (~340 lines) in aether-utils.sh
- Calls queen-read internally to get wisdom from QUEEN.md
- Calls pheromone-prime internally to get signals + instincts
- Returns unified JSON with:
  - metadata: version, stats, thresholds from QUEEN.md
  - wisdom: combined wisdom from global + local QUEEN.md
  - signals: signal_count, instinct_count, active_signals
  - prompt_section: formatted markdown ready for worker injection
  - log_line: status message

### Two-Level QUEEN.md Loading
- Global QUEEN.md loads first from `~/.aether/QUEEN.md`
- Local QUEEN.md loads second from `.aether/docs/QUEEN.md`
- Local wisdom extends global - entries appended per category
- Categories: Philosophies, Patterns, Redirects, Stack Wisdom, Decrees
- Metadata and Evolution Log excluded from worker context (per locked decision)

### Error Handling (per locked decisions)
- QUEEN.md missing: FAIL HARD with clear error requiring /ant:init
- pheromones.json missing: WARN but continue (workers just won't receive signals)

## Verification

```bash
# Test colony-prime
AETHER_ROOT="$PWD" bash .aether/aether-utils.sh colony-prime

# Test queen-read with two-level loading
AETHER_ROOT="$PWD" bash .aether/aether-utils.sh queen-read

# Test missing QUEEN.md error
# (temporarily move QUEEN.md)
# Expected: clear error with actionable message
```

## Deviation: None

Plan executed exactly as written. No auto-fixes needed.

## Auth Gates: None

No authentication required for this implementation.

## Self-Check

- [x] colony-prime function exists in aether-utils.sh
- [x] Returns unified JSON with wisdom + signals + prompt_section
- [x] Two-level loading works (global first, then local)
- [x] queen-read updated with two-level loading
- [x] Fails hard if QUEEN.md missing
- [x] Warns but continues if pheromones.json missing
- [x] Committed: 36c5407

## Self-Check: PASSED
