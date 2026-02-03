---
phase: 16-worker-knowledge
plan: 03
subsystem: worker-specs
tags: [events, memory, spawning, worker-knowledge, spec-enrichment]
dependency-graph:
  requires: ["16-01", "16-02"]
  provides: ["event-aware-workers", "memory-reading-workers", "spawning-scenarios"]
  affects: ["17-integration-dashboard"]
tech-stack:
  added: []
  patterns: ["startup-awareness-protocol", "recursive-spec-propagation"]
key-files:
  created: []
  modified:
    - .aether/workers/watcher-ant.md
    - .aether/workers/builder-ant.md
    - .aether/workers/scout-ant.md
    - .aether/workers/architect-ant.md
    - .aether/workers/colonizer-ant.md
    - .aether/workers/route-setter-ant.md
decisions:
  - decision: "Event awareness and memory reading sections placed between Feedback Interpretation and Workflow"
    rationale: "Logical flow: pheromone knowledge -> event/memory awareness -> workflow execution"
  - decision: "Each caste spawns a different caste in its scenario"
    rationale: "Demonstrates the full diversity of cross-caste spawning: builder->scout, watcher->builder, scout->colonizer, architect->watcher, colonizer->architect, route-setter->colonizer"
metrics:
  duration: "3 min"
  completed: "2026-02-03"
---

# Phase 16 Plan 03: Event Awareness, Memory Reading, and Spawning Scenarios Summary

**One-liner:** All 6 worker specs enriched with events.json startup reading, memory.json knowledge access, and caste-specific spawning scenarios with full Task tool prompt examples demonstrating recursive spec propagation.

## What Was Done

### Task 1: Event Awareness and Memory Reading (0ba8407)

Added two new sections to all 6 worker specs:

**Event Awareness** -- Each spec now instructs the worker to read `.aether/data/events.json` at startup, filter events by time (last 30 minutes or since phase start), and includes a caste-specific relevance table mapping all 6 event types (phase_started, phase_completed, error_logged, pheromone_set, decision_logged, learning_extracted) to relevance levels (HIGH/MEDIUM/LOW) and specific actions. Each caste has different priority rankings reflecting their role.

**Memory Reading** -- Each spec now instructs the worker to read `.aether/data/memory.json` at startup, check decisions and phase_learnings arrays, and includes caste-specific guidance on what to look for. Builder looks for tech choices and constraints; Scout looks for research strategies; Colonizer looks for structural decisions; Watcher looks for quality standards; Architect looks for all decisions as synthesis input; Route-setter looks for planning approaches.

### Task 2: Caste-Specific Spawning Scenarios (b800126)

Added a `### Spawning Scenario` subsection to the existing spawning section in all 6 worker specs. Each scenario includes:

1. A realistic situation description (1-2 sentences)
2. Decision process with effective signal calculation and spawn budget check
3. A complete Task tool prompt example with `--- WORKER SPEC ---`, `--- ACTIVE PHEROMONES ---`, and `--- TASK ---` sections
4. Explanation of recursive spec propagation

Cross-caste spawning diversity:
- Builder spawns Scout (needs auth library research)
- Watcher spawns Builder (needs performance benchmarks)
- Scout spawns Colonizer (needs architecture mapping)
- Architect spawns Watcher (needs test results for pattern validation)
- Colonizer spawns Architect (needs business logic synthesis)
- Route-setter spawns Colonizer (needs codebase structure for planning)

## Line Counts

| Spec | Lines | Target | Status |
|------|-------|--------|--------|
| watcher-ant.md | 324 | ~220 | Exceeds (has specialist modes) |
| route-setter-ant.md | 230 | ~200 | Exceeds |
| scout-ant.md | 224 | ~200 | Exceeds |
| architect-ant.md | 214 | ~200 | Exceeds |
| colonizer-ant.md | 210 | ~200 | Exceeds |
| builder-ant.md | 209 | ~200 | Exceeds |

## Deviations from Plan

None -- plan executed exactly as written.

## Success Criteria Met

- SPEC-04: Every worker spec reads events.json at startup with filtering and caste-specific relevance tables
- SPEC-05: Every worker spec includes a complete spawning scenario with Task tool prompt example
- MEM-04: Workers read memory.json entries at startup with caste-specific guidance
- EVT-03: Workers read events.json at startup with time filtering
- All 6 specs exceed target ~200 lines (watcher at 324 with specialist modes)

## Next Phase Readiness

Phase 16 (Worker Knowledge) is now complete. All 3 plans delivered:
- 16-01: Pheromone math, combination effects, feedback interpretation
- 16-02: Pheromone math and feedback knowledge
- 16-03: Event awareness, memory reading, spawning scenarios

Worker specs are fully enriched from ~90 lines to 200-324 lines each. Ready for Phase 17 (Integration & Dashboard).
