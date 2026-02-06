# Roadmap: Aether v5.1 System Simplification

## Overview

Transform Aether from a 7,400-line over-engineered system to a lean 1,800-line implementation based on the M4L-AnalogWave postmortem findings. The postmortem identified that 70% of context was consumed by framework overhead, state fell out of sync at context boundaries, and mathematical models (Bayesian, exponential decay) added complexity without value. This milestone follows the postmortem's recommended implementation order: consolidate state first, rewrite the highest-impact commands, then simplify supporting infrastructure.

## Milestones

- **v5.1 System Simplification** - Phases 33-40 (gap closure in progress)

## Phases

- [x] **Phase 33: State Foundation** - Consolidate 6 state files into single COLONY_STATE.json
- [x] **Phase 34: Core Command Rewrite** - Rewrite build and continue commands with start-of-next-command state updates
- [x] **Phase 35: Worker Simplification** - Collapse 6 worker specs into single workers.md
- [x] **Phase 36: Signal Simplification** - Replace pheromone exponential decay with simple TTL
- [x] **Phase 37: Command Trim & Utilities** - Shrink remaining commands and reduce aether-utils.sh
- [x] **Phase 38: Signal Schema Unification** - Fix init.md and signal path inconsistencies (gap closure)
- [ ] **Phase 39: Worker Reference Consolidation** - Update commands to use workers.md (gap closure)
- [ ] **Phase 40: State & Utility Alignment** - Fix remaining state file references and sync utilities (gap closure)

## Phase Details

### Phase 33: State Foundation
**Goal**: Single COLONY_STATE.json replaces 6 distributed state files
**Depends on**: Nothing (first phase of v5.1)
**Requirements**: SIMP-01
**Success Criteria** (what must be TRUE):
  1. All commands read state from single COLONY_STATE.json file
  2. All commands write state to single COLONY_STATE.json file
  3. Phase data, signals, learnings, errors, and events coexist in one JSON structure
  4. Migration script converts existing 6-file state to new format
**Plans**: 4 plans

Plans:
- [x] 33-01-PLAN.md — Schema design and migration script
- [x] 33-02-PLAN.md — Update init.md and status.md
- [x] 33-03-PLAN.md — Update signal and read-only commands
- [x] 33-04-PLAN.md — Update remaining complex commands

### Phase 34: Core Command Rewrite
**Goal**: build.md and continue.md rewritten with state updates at start-of-next-command
**Depends on**: Phase 33
**Requirements**: SIMP-02, SIMP-05 (partial: build + continue), SIMP-07
**Success Criteria** (what must be TRUE):
  1. ant:build writes only "EXECUTING" state, does not update task completion status
  2. ant:continue detects completed output files and updates state accordingly
  3. State survives context boundaries (no more orphaned EXECUTING status)
  4. build.md reduced from 1,080 lines to ~400 lines
  5. continue.md reduced from 534 lines to ~150 lines
**Plans**: 3 plans

Plans:
- [x] 34-01-PLAN.md — Rewrite build.md with minimal state write pattern
- [x] 34-02-PLAN.md — Rewrite continue.md with detection and reconciliation
- [x] 34-03-PLAN.md — Verify build/continue integration and polish

### Phase 35: Worker Simplification
**Goal**: Six worker specs collapsed into single workers.md (~200 lines)
**Depends on**: Phase 34
**Requirements**: SIMP-04
**Success Criteria** (what must be TRUE):
  1. Single workers.md file contains all 6 role definitions
  2. Sensitivity matrices removed from worker definitions
  3. Spawning protocols simplified to role assignment guidelines
  4. Total worker spec lines reduced from 1,866 to ~200
**Plans**: 2 plans

Plans:
- [x] 35-01-PLAN.md — Create consolidated workers.md with all 6 roles
- [x] 35-02-PLAN.md — Update commands and delete old worker files

### Phase 36: Signal Simplification
**Goal**: Pheromone system uses simple TTL instead of exponential decay
**Depends on**: Phase 35
**Requirements**: SIMP-03
**Success Criteria** (what must be TRUE):
  1. Signals use expires_at timestamp instead of half-life math
  2. Priority field (high/normal/low) replaces sensitivity matrix calculations
  3. Expired signals filtered on read (no cleanup command needed)
  4. All pheromone math removed from aether-utils.sh
**Plans**: 5 plans

Plans:
- [x] 36-01-PLAN.md — Update signal emission commands with TTL schema
- [x] 36-02-PLAN.md — Update signal consumption commands with TTL filtering
- [x] 36-03-PLAN.md — Remove pheromone math from utilities and update docs
- [x] 36-04-PLAN.md — (gap closure) Remove decay commands from runtime/aether-utils.sh
- [x] 36-05-PLAN.md — (gap closure) Update plan/organize/colonize commands, delete runtime/workers/*.md

### Phase 37: Command Trim & Utilities
**Goal**: Remaining commands shrunk and aether-utils.sh reduced to ~80 lines
**Depends on**: Phase 36
**Requirements**: SIMP-05 (remaining commands), SIMP-06
**Success Criteria** (what must be TRUE):
  1. colonize.md reduced from 538 lines to ~150 lines
  2. status.md reduced from 303 lines to ~80 lines
  3. Signal commands (focus, redirect, feedback) reduced to ~40 lines each
  4. aether-utils.sh reduced from 372 lines to ~80 lines (keeping validate-state, error-add)
  5. Total system lines at ~1,800 (from ~7,400)
**Plans**: 4 plans

Plans:
- [x] 37-01-PLAN.md — Reduce signal commands (focus, redirect, feedback) to ~40 lines each
- [x] 37-02-PLAN.md — Reduce status.md to ~80 lines with quick-glance output
- [x] 37-03-PLAN.md — Reduce colonize.md to ~150 lines with surface scan pattern
- [x] 37-04-PLAN.md — Reduce aether-utils.sh to ~80 lines and sync command directories

### Phase 38: Signal Schema Unification
**Goal**: Fix init.md to use TTL signal schema and ensure all signal paths are consistent
**Depends on**: Phase 37
**Gap Closure**: Addresses audit integration gaps #3, flow gaps #1 and #2
**Success Criteria** (what must be TRUE):
  1. init.md writes signals with TTL schema (priority, expires_at) not legacy schema (strength, half_life)
  2. All signal commands write to COLONY_STATE.json signals section
  3. build.md reads signals from COLONY_STATE.json (not pheromones.json)
  4. Colony initialization flow works end-to-end
**Plans**: 1 plan

Plans:
- [x] 38-01-PLAN.md — Update init.md signal schema and unify signal paths across all commands

### Phase 39: Worker Reference Consolidation
**Goal**: Update all commands to reference consolidated workers.md instead of individual worker files
**Depends on**: Phase 38
**Gap Closure**: Addresses audit integration gap #2
**Success Criteria** (what must be TRUE):
  1. build.md references ~/.aether/workers.md (not individual caste files)
  2. plan.md references ~/.aether/workers.md
  3. organize.md references ~/.aether/workers.md
  4. No command references ~/.aether/workers/{caste}-ant.md pattern

Plans:
- [ ] 39-01-PLAN.md — Update worker references in build.md, plan.md, organize.md

### Phase 40: State & Utility Alignment
**Goal**: Complete state consolidation and sync utility scripts
**Depends on**: Phase 39
**Gap Closure**: Addresses audit integration gaps #1, #4, #5, flow gap #3
**Success Criteria** (what must be TRUE):
  1. No command reads/writes PROJECT_PLAN.json, pheromones.json, errors.json, memory.json, or events.json
  2. runtime/aether-utils.sh matches ~/.aether/aether-utils.sh (or runtime removed)
  3. Documentation reflects actual line counts
  4. Build/continue handoff uses single state file

Plans:
- [ ] 40-01-PLAN.md — Audit and fix remaining separate file references
- [ ] 40-02-PLAN.md — Sync utility scripts and update documentation

## Progress

**Execution Order:** Phases 33 through 40 in sequence

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 33. State Foundation | 4/4 | Complete | 2026-02-06 |
| 34. Core Command Rewrite | 3/3 | Complete | 2026-02-06 |
| 35. Worker Simplification | 2/2 | Complete | 2026-02-06 |
| 36. Signal Simplification | 5/5 | Complete | 2026-02-06 |
| 37. Command Trim & Utilities | 4/4 | Complete | 2026-02-06 |
| 38. Signal Schema Unification | 1/1 | Complete | 2026-02-06 |
| 39. Worker Reference Consolidation | 0/1 | Pending | - |
| 40. State & Utility Alignment | 0/2 | Pending | - |

---
*Roadmap created: 2026-02-06*
*Source: COLONY_POSTMORTEM.md*
