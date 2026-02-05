---
phase: 31-architecture-evolution
plan: 03
subsystem: spawning
tags: [spawn-tree, queen-mediated-delegation, depth-limit, worker-specs, colony-state]

# Dependency graph
requires:
  - phase: 30-automation
    provides: Post-wave advisory review, debugger-ant, retry logic in build.md
  - phase: 31-01
    provides: Updated aether-utils.sh with learning subcommands
  - phase: 31-02
    provides: Learning injection in colonize.md
provides:
  - Queen-mediated spawn tree engine in build.md (SPAWN REQUEST parsing, sub-spawn fulfillment)
  - Updated worker specs with Requesting Sub-Spawns section (SPAWN REQUEST output format)
  - spawn-check depth limit reduced from 3 to 2
  - validate-state colony with optional spawn_tree field check
affects: [future phases using hierarchical delegation]

# Tech tracking
tech-stack:
  added: []
  patterns: [queen-mediated-delegation, spawn-request-output-pattern, depth-capped-tree]

key-files:
  created: []
  modified:
    - .claude/commands/ant/build.md
    - .aether/workers/builder-ant.md
    - .aether/workers/watcher-ant.md
    - .aether/workers/colonizer-ant.md
    - .aether/workers/scout-ant.md
    - .aether/workers/architect-ant.md
    - .aether/workers/route-setter-ant.md
    - .aether/aether-utils.sh

key-decisions:
  - id: 31-03-d1
    decision: "Queen-mediated delegation via SPAWN REQUEST blocks in worker output rather than direct Task tool spawning"
    rationale: "Claude Code platform constraint prevents subagents from spawning subagents (Task tool unavailable). Workers signal needs, Queen fulfills between waves."
  - id: 31-03-d2
    decision: "Max 2 sub-spawns per wave cap"
    rationale: "Prevents runaway delegation chains. Workers should handle most tasks inline -- sub-spawning is for genuinely independent sub-tasks."
  - id: 31-03-d3
    decision: "Depth limit reduced from 3 to 2 across all specs and spawn-check"
    rationale: "Queen->worker->sub-worker is the practical maximum. Depth-2 workers told they cannot sub-spawn and must handle inline."

metrics:
  duration: "~7 min"
  completed: "2026-02-05"
---

# Phase 31 Plan 03: Queen-Mediated Spawn Tree Engine Summary

Queen-mediated delegation via SPAWN REQUEST output blocks with depth-2 cap, spawn_tree in COLONY_STATE.json, and delegation tree visual display

## What Was Done

### Task 1: Add spawn tree engine to build.md
Added 6 targeted modifications to build.md:
1. **spawn_tree initialization** -- `spawn_tree = {}` and `queued_sub_spawns = []` in Step 5c counters
2. **Depth-1 worker tracking** -- Each spawned worker recorded in spawn_tree with depth:1, parent:"queen"
3. **Step 5c.e2** -- SPAWN REQUEST parsing after each worker returns (extract caste/task/context/files, validate depth, cap 2 per wave)
4. **Step 5c.j** -- Post-wave sub-spawn fulfillment (record in spawn_tree, log, display, spawn via Task tool with depth-2 prompt, update status)
5. **Step 6 spawn_tree recording** -- Write spawn_tree to COLONY_STATE.json
6. **Step 7e Delegation Tree visual** -- ANSI-colored tree display showing Queen -> depth-1 workers -> depth-2 sub-workers with status

### Task 2: Update all worker specs and aether-utils.sh
**Part A -- 6 worker specs updated:**
- Replaced "You Can Spawn Other Ants" (with Spawn Gate, Confidence Check, Spawning Scenario subsections) with "Requesting Sub-Spawns" section
- New section includes: SPAWN REQUEST output format example, rules (handle inline when possible, max 1-2 requests, depth check), available castes list, spawn limits
- Updated Post-Action Validation depth from /3 to /2
- Each spec has a caste-appropriate example scenario in the SPAWN REQUEST block

**Part B -- spawn-check updated:**
- max_depth: 3 -> 2
- pass condition: depth < 3 -> depth < 2
- depth_limit check: depth >= 3 -> depth >= 2

**Part C -- validate-state colony updated:**
- Added `opt` function for optional field validation (passes when absent, validates type when present)
- Added `opt("spawn_tree";["object"])` check (backward compatible)

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

1. build.md contains spawn tree engine -- all 6 modifications present
2. All 6 worker specs have "Requesting Sub-Spawns", none have "You Can Spawn Other Ants"
3. No worker spec references "Max depth 3"
4. `spawn-check 1` returns pass:true, `spawn-check 2` returns pass:false with reason:depth_limit
5. `validate-state colony` returns pass:true with 6 checks (spawn_tree optional)
6. Delegation Tree visual in Step 7e shows parent-child relationships with ANSI colors
7. Depth-2 worker prompt includes "You CANNOT request further sub-spawns"
8. build.md total: 1119 lines (within target range)

## Key Metrics

- Worker spec reduction: 669 lines removed, 186 lines added (net -483 lines across 7 files)
- build.md grew by 59 lines (from ~1060 to 1119)
- Total files modified: 8

## Next Phase Readiness

Phase 31 plan 03 is the final plan in this phase. All 3 plans complete:
- 31-01: Two-tier learning system (global learnings promotion + injection infrastructure)
- 31-02: Learning injection in colonize.md (not executed by this agent but exists as plan)
- 31-03: Queen-mediated spawn tree engine (this plan)

Blocker CP-1 (recursive spawning platform constraint) is now resolved -- the Queen-mediated delegation pattern provides equivalent functionality without requiring Task tool availability in subagents.
