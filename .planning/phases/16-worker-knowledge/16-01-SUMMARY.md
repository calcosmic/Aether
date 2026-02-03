---
phase: 16-worker-knowledge
plan: 01
subsystem: worker-specs
tags: [watcher, specialist-modes, severity-rubric, detection-checklist]

dependency_graph:
  requires: [14-visual-identity, 15-infrastructure-state]
  provides: [watcher-specialist-modes, severity-rubrics, detection-checklists]
  affects: [16-02, 16-03, 17-integration]

tech_stack:
  added: []
  patterns: [specialist-mode-structure, activation-triggers, severity-rubric-table]

files:
  created: []
  modified:
    - .aether/workers/watcher-ant.md

decisions:
  - id: WATCH-MODES
    choice: "4 specialist modes with activation triggers, focus areas, severity rubric, detection checklist"
    rationale: "Transforms flat checklist into deep domain knowledge the watcher can autonomously apply"

metrics:
  duration: 1min
  completed: 2026-02-03
---

# Phase 16 Plan 01: Watcher Specialist Modes Summary

**One-liner:** 4 specialist watcher modes (security, performance, quality, test-coverage) with pheromone-triggered activation, 4-level severity rubrics, and 6-item detection checklists each.

## What Was Done

Replaced the flat 4-section Validation Checklist in watcher-ant.md with a structured Specialist Modes section. Each mode contains:

1. **Activation Triggers** - When the mode activates, referencing FEEDBACK pheromone keywords and task context
2. **Focus Areas** - 5 domain-specific areas the watcher should examine
3. **Severity Rubric** - CRITICAL/HIGH/MEDIUM/LOW table with criteria and concrete examples
4. **Detection Checklist** - 6 actionable checkbox items for systematic verification

The Workflow step 4 was updated to reference specialist mode activation instead of generic validation.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Replace flat Validation Checklist with 4 deep specialist modes | 8271377 | .aether/workers/watcher-ant.md |

## Verification Results

- watcher-ant.md contains "## Specialist Modes" section: PASS
- 4 modes present (Security, Performance, Quality, Test Coverage): PASS
- 4x Activation Triggers: PASS
- 4x Severity Rubric (4-level tables): PASS
- 4x Detection Checklist (6 items each): PASS
- Spawning section preserved: PASS
- File expanded to 195 lines (from 104, target 170+): PASS

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| WATCH-MODES | Structured specialist modes with activation/severity/detection pattern | Deep domain knowledge enables autonomous watcher assessment without external scripts |

## Next Phase Readiness

- watcher-ant.md now has deep specialist knowledge for autonomous validation
- Ready for 16-02 (expanding other worker specs)
- No blockers or concerns
