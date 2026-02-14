# Roadmap: Aether Colony System v3.1 Open Chambers

**Milestone:** v3.1 Open Chambers
**Goal:** Implement intelligent model routing for worker castes, establish colony lifecycle management (archive/foundation) with ant-themed terminology, and create an immersive real-time visualization experience.
**Defined:** 2026-02-14
**Phases:** 4 (9-12)
**Requirements:** 27 v3.1 requirements

---

## Overview

This roadmap delivers the v3.1 "Open Chambers" milestone, expanding the Aether Colony System with verified model routing, colony lifecycle management, and immersive visualization. The work builds on the hardened v3.0.0 foundation to enable intelligent agent orchestration with ant-themed metaphors throughout.

**Key Deliverables:**
- Verified model-to-caste assignments with CLI configuration
- Colony lifecycle commands (`/ant:entomb`, `/ant:lay-eggs`, `/ant:tunnels`)
- Milestone auto-detection (First Mound → Open Chambers → Brood Stable...)
- Real-time colony visualization with caste colors and emojis

---

## Phase Structure

| Phase | Name | Goal | Requirements | Status |
|-------|------|------|--------------|--------|
| 9 | Caste Model Assignment | User can view, verify, and configure model assignments per caste | 8 | Complete |
| 10 | Entombment & Egg Laying | User can archive colonies and start fresh with lifecycle management | 5 | Not Started |
| 11 | Foraging Specialization | System intelligently routes tasks to optimal models based on content | 3 | Not Started |
| 12 | Colony Visualization | User sees immersive real-time colony activity with ant-themed presentation | 11 | Not Started |

---

## Phase 9: Caste Model Assignment

**Goal:** Users can view, verify, and configure which AI models are assigned to each worker caste, with proxy health verification and logging.

**Dependencies:** Phase 8 (v3.0.0 Core Reliability)

**Requirements Mapped:**
| ID | Requirement |
|----|-------------|
| MOD-01 | User can view current model assignments per caste |
| MOD-02 | User can override model for specific caste |
| MOD-03 | System verifies LiteLLM proxy health before spawning workers |
| MOD-04 | Model verification shows which provider each model routes to |
| MOD-05 | System logs actual model used per worker spawn |
| QUICK-01 | Surface Dreams in `/ant:status` |
| QUICK-02 | Auto-Load Context — commands recognize nestmates |
| QUICK-03 | `/ant:verify-castes` command |

**Plans:**
- [x] 09-01-PLAN.md — Foundation: js-yaml + model-profiles.js library
- [x] 09-02-PLAN.md — CLI commands: list, set, reset for caste-models
- [x] 09-03-PLAN.md — Proxy health verification + /ant:verify-castes command
- [x] 09-04-PLAN.md — Spawn logging with model tracking
- [x] 09-05-PLAN.md — Quick wins: Dreams in status, nestmate auto-load

**Success Criteria:**
1. User runs `aether caste-models list` and sees current assignments (Builder=kimi-k2.5, etc.)
2. User runs `aether caste-models set builder=claude-sonnet` and override persists
3. User runs `/ant:verify-castes` and sees proxy health + provider routing info
4. Worker spawn logs include actual model used (visible in activity.log)
5. `/ant:status` shows recent dream count and last dream time
6. Commands automatically load TO-DOs and colony state without manual steps

**Anti-Patterns to Avoid:**
- Model routing without verification (configuration exists but doesn't execute)
- Proxy auth failures silently defaulting to fallback models
- Environment variables not inherited by Task tool

---

## Phase 10: Entombment & Egg Laying

**Goal:** Users can archive completed colonies (entomb), start fresh colonies (lay eggs), browse history (explore tunnels), and see automatic milestone detection.

**Dependencies:** Phase 9

**Requirements Mapped:**
| ID | Requirement |
|----|-------------|
| LIFE-01 | `/ant:entomb` — archive colony to `.aether/chambers/` with pheromone trails |
| LIFE-02 | `/ant:lay-eggs` — start fresh colony (First Eggs milestone) |
| LIFE-03 | Milestone auto-detection from state |
| LIFE-04 | `/ant:tunnels` — browse archived colonies |
| LIFE-05 | Entombment includes pheromone manifest (manifest.json) |

**Success Criteria:**
1. User runs `/ant:entomb` and colony is archived to `.aether/chambers/{timestamp}/` with manifest.json
2. Manifest includes: date, goal, phases completed, learnings preserved (pheromone trails)
3. User runs `/ant:lay-eggs` and fresh COLONY_STATE.json is created (First Eggs milestone)
4. User can spawn from entombed chamber or start completely fresh
5. `/ant:status` shows current milestone (First Mound, Open Chambers, Brood Stable, etc.)
6. User runs `/ant:tunnels` and sees list of archived colonies with summaries

**Anti-Patterns to Avoid:**
- Archive destroying user data (learnings/decisions)
- Pause/resume losing model context
- Entombment without metadata (unreproducible archives)

---

## Phase 11: Foraging Specialization

**Goal:** System intelligently routes tasks to optimal models based on task content keywords, with performance telemetry and per-command overrides.

**Dependencies:** Phase 9 (model verification must work first)

**Requirements Mapped:**
| ID | Requirement |
|----|-------------|
| MOD-06 | Task-based routing — keyword detection routes to appropriate model |
| MOD-07 | Model performance telemetry — track success rates per model/caste |
| MOD-08 | Model override per command (`--model` flag) |

**Success Criteria:**
1. Task containing "design" or "architecture" routes to glm-5 automatically
2. Task containing "implement" or "build" routes to kimi-k2.5 automatically
3. User can run `aether build --model=claude-opus` for one-time override
4. System tracks success/failure rates per model-caste combination
5. User can view performance telemetry via `/ant:caste-models performance`
6. Task routing decision is logged with keywords matched

**Anti-Patterns to Avoid:**
- Task routing never triggered (keywords don't match)
- Performance telemetry without enough data to be meaningful
- Override flag breaking context consistency

---

## Phase 12: Colony Visualization

**Goal:** Users experience immersive real-time colony activity display with ant-themed presentation, collapsible views, and comprehensive metrics.

**Dependencies:** Phase 10 (lifecycle commands provide data), Phase 9 (caste assignments provide colors)

**Requirements Mapped:**
| ID | Requirement |
|----|-------------|
| VIZ-01 | Real-time foraging display with caste emoji |
| VIZ-02 | Collapsible tunnel view for nested agent spawns |
| VIZ-03 | Tool usage stats (Read/Grep/Edit/Bash counts) |
| VIZ-04 | Trophallaxis metrics (token usage) |
| VIZ-05 | Timing information (duration, elapsed, ETA) |
| VIZ-06 | Ant-themed presentation ("3 foragers excavating...") |
| VIZ-07 | Chamber activity map (nest zones with active ants) |
| VIZ-08 | Live excavation progress bars |
| VIZ-09 | Color + caste emoji together (not replacing each other) |
| LIFE-06 | ASCII art anthill visualization showing maturity journey |
| LIFE-07 | Chamber comparison — compare pheromone trails across colonies |

**Success Criteria:**
1. `/ant:swarm` shows real-time display: "3 foragers excavating..." with caste emojis
2. Each caste has distinct color (Builder=blue, Watcher=green, Scout=yellow, Chaos=red, Prime=purple)
3. Each caste shows emoji alongside color (🔨🐜, 👁️🐜, 🔍🐜, 🎲🐜)
4. Tunnel view can expand/collapse to show nested agent spawns
5. Tool usage stats show Read/Grep/Edit/Bash counts per ant
6. Trophallaxis metrics display token consumption per task
7. Progress bars show live excavation status for long operations
8. Chamber activity map shows which nest zones have active ants (Fungus Garden, Nursery, Refuse Pile)
9. `/ant:maturity` shows ASCII art anthill with journey from First Mound to Crowned Anthill
10. User can compare pheromone trails across two entombed chambers

**Anti-Patterns to Avoid:**
- Colors OR emojis (must have both together)
- Visualization without real data (mock displays)
- Activity map that's always empty or always full

---

## Progress

| Phase | Name | Requirements | Status | Completed |
|-------|------|--------------|--------|-----------|
| 9 | Caste Model Assignment | 8 | Planned | 0% |
| 10 | Entombment & Egg Laying | 5 | Not Started | 0% |
| 11 | Foraging Specialization | 3 | Not Started | 0% |
| 12 | Colony Visualization | 11 | Not Started | 0% |

**Overall Progress:** 0/27 requirements (0%)

---

## Dependencies Between Phases

```
Phase 9 (Caste Model Assignment)
    │
    ├──→ Phase 10 (Entombment & Egg Laying) ──→ Phase 12 (Colony Visualization)
    │                                               (needs lifecycle data)
    │
    └──→ Phase 11 (Foraging Specialization)
         (needs verified routing first)
```

**Critical Path:** 9 → 10 → 12 (visualization needs lifecycle data)
**Parallel Work:** 9 → 11 can proceed in parallel with 10

---

## Coverage Validation

**v3.1 Requirements:** 27 total
**Mapped to Phases:** 27
**Unmapped:** 0 ✓

| Category | Count | Phases |
|----------|-------|--------|
| Model Routing (MOD) | 8 | 9, 11 |
| Colony Lifecycle (LIFE) | 7 | 10, 12 |
| Visualization (VIZ) | 9 | 12 |
| Quick Wins (QUICK) | 3 | 9 |

---

## Ant Terminology Reference

| Concept | Ant Term | Used In |
|---------|----------|---------|
| Archive | Entomb | Phase 10 |
| New Milestone | Lay Eggs / First Eggs | Phase 10 |
| History | Tunnels / Explore Tunnels | Phase 10 |
| Metadata | Pheromone Trails | Phase 10, 12 |
| Status | Nest Status | Phase 9, 10 |
| Activity Log | Foraging Trails | Phase 12 |
| Progress | Excavation | Phase 12 |
| Resources/Tokens | Trophallaxis | Phase 12 |
| Model Assignment | Caste Assignment | Phase 9 |
| Task Routing | Foraging Specialization | Phase 11 |

---

*Roadmap created: 2026-02-14*
*Starting phase: 9 (continuing from v3.0.0)*
