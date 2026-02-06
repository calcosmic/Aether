# Roadmap: Aether v5.1 System Simplification

## Overview

Transform Aether from a 7,400-line over-engineered system to a lean 1,800-line implementation based on the M4L-AnalogWave postmortem findings. The postmortem identified that 70% of context was consumed by framework overhead, state fell out of sync at context boundaries, and mathematical models (Bayesian, exponential decay) added complexity without value. This milestone follows the postmortem's recommended implementation order: consolidate state first, rewrite the highest-impact commands, then simplify supporting infrastructure.

## Milestones

- **v5.1 System Simplification** - Phases 33-37 (in progress)

## Phases

- [x] **Phase 33: State Foundation** - Consolidate 6 state files into single COLONY_STATE.json
- [ ] **Phase 34: Core Command Rewrite** - Rewrite build and continue commands with start-of-next-command state updates
- [ ] **Phase 35: Worker Simplification** - Collapse 6 worker specs into single workers.md
- [ ] **Phase 36: Signal Simplification** - Replace pheromone exponential decay with simple TTL
- [ ] **Phase 37: Command Trim & Utilities** - Shrink remaining commands and reduce aether-utils.sh

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
- [ ] 34-01-PLAN.md — Rewrite build.md with minimal state write pattern
- [ ] 34-02-PLAN.md — Rewrite continue.md with detection and reconciliation
- [ ] 34-03-PLAN.md — Verify build/continue integration and polish

### Phase 35: Worker Simplification
**Goal**: Six worker specs collapsed into single workers.md (~200 lines)
**Depends on**: Phase 34
**Requirements**: SIMP-04
**Success Criteria** (what must be TRUE):
  1. Single workers.md file contains all 6 role definitions
  2. Sensitivity matrices removed from worker definitions
  3. Spawning protocols simplified to role assignment guidelines
  4. Total worker spec lines reduced from 1,866 to ~200
**Plans**: TBD

Plans:
- [ ] 35-01: TBD

### Phase 36: Signal Simplification
**Goal**: Pheromone system uses simple TTL instead of exponential decay
**Depends on**: Phase 35
**Requirements**: SIMP-03
**Success Criteria** (what must be TRUE):
  1. Signals use expires_at timestamp instead of half-life math
  2. Priority field (high/normal/low) replaces sensitivity matrix calculations
  3. Expired signals filtered on read (no cleanup command needed)
  4. All pheromone math removed from aether-utils.sh
**Plans**: TBD

Plans:
- [ ] 36-01: TBD

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
**Plans**: TBD

Plans:
- [ ] 37-01: TBD
- [ ] 37-02: TBD

## Progress

**Execution Order:** Phases 33 through 37 in sequence

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 33. State Foundation | 4/4 | Complete | 2026-02-06 |
| 34. Core Command Rewrite | 0/3 | Planned | - |
| 35. Worker Simplification | 0/TBD | Not started | - |
| 36. Signal Simplification | 0/TBD | Not started | - |
| 37. Command Trim & Utilities | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-06*
*Source: COLONY_POSTMORTEM.md*
