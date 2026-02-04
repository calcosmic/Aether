# Roadmap: Aether v4.3

## Milestones

- v1.0 MVP -- Phases 3-10 (shipped 2026-02-02)
- v2.0 Event & Visual -- Phases 11-13 (shipped 2026-02-02)
- v3.0 Rebuild -- Phases 14-18 (shipped 2026-02-03)
- v4.0 Hybrid Foundation -- Phases 19-21 (shipped 2026-02-03)
- v4.1 Cleanup & Enforcement -- Phases 22-23 (shipped 2026-02-03)
- v4.2 Colony Hardening -- Phase 24 (shipped 2026-02-03)
- **v4.3 Live Visibility & Auto-Learning** -- Phases 25-26

## Overview

v4.3 addresses two gaps found during codebase analysis: (1) workers output progress markers but they're invisible to users because Task tool subagents don't stream output — fixed by having the Queen orchestrate spawns directly and display results incrementally; (2) learning extraction requires a manual `/ant:continue` call that's easy to forget — fixed by auto-extracting learnings in build.md Step 7.

## Phases

- [ ] **Phase 25: Live Visibility** - Activity log + incremental Queen display so users see worker progress during execution
- [ ] **Phase 26: Auto-Learning** - Automatic learning extraction in build.md, skip duplicate extraction in continue.md

## Phase Details

### Phase 25: Live Visibility
**Goal**: Users see what each worker did as it completes, not after the entire Phase Lead returns — workers write to activity log, Queen spawns workers directly and displays results incrementally
**Depends on**: Phase 24 (v4.2 complete)
**Requirements**: VIS-01, VIS-02, VIS-03
**Success Criteria** (what must be TRUE):
  1. Workers write structured progress lines to `.aether/data/activity.log` with timestamps, caste emoji, and action type
  2. build.md spawns workers sequentially through the Queen (not delegated to Phase Lead) so each worker's results are visible before the next spawns
  3. After each worker returns, the Queen displays that worker's activity log entries and result summary to the user
  4. The Phase Lead role changes from "spawn and manage all workers" to "plan task assignments" — execution moves to Queen level
  5. Activity log is cleared at phase start to prevent stale data
**Plans**: TBD (created during /cds:plan-phase)

### Phase 26: Auto-Learning
**Goal**: Phase learnings are automatically captured at the end of build execution — no manual `/ant:continue` required for learning extraction
**Depends on**: Phase 25
**Requirements**: LEARN-01, LEARN-02, LEARN-03
**Success Criteria** (what must be TRUE):
  1. build.md Step 7 reads errors.json, events.json, and task outcomes, then synthesizes actionable learnings and appends to memory.json (same logic as continue.md Step 4)
  2. build.md Step 7 auto-emits a FEEDBACK pheromone summarizing what worked/failed, validated via pheromone-validate before writing
  3. continue.md detects whether learnings were already extracted (checks memory.json for learnings from current phase) and skips extraction if so
  4. Learning extraction in build.md respects existing memory-compress limits (20 learnings max)
**Plans**: TBD (created during /cds:plan-phase)

## Progress

**Execution Order:** 25 -> 26

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 25. Live Visibility | v4.3 | 0/? | ○ Pending | - |
| 26. Auto-Learning | v4.3 | 0/? | ○ Pending | - |

---
*Roadmap created: 2026-02-04*
*Milestone: v4.3 Live Visibility & Auto-Learning*
