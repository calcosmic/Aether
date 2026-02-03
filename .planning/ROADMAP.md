# Roadmap: Aether — Queen Ant Colony

## Overview

Aether delivers autonomous emergence through Claude-native skill prompts, JSON state, and recursive worker spawning. v1.0 built the foundation (phases 3-10), v2.0 added reactive event integration (phases 11-13), and v3.0 restores the sophistication lost during the Claude-native rebuild — visual identity, deep worker knowledge, infrastructure state, and an integrated dashboard that makes emergence visible.

## Milestones

- ✅ **v1.0 Queen Ant Colony** — Phases 3-10 (shipped 2026-02-02)
- ✅ **v2.0 Reactive Event Integration** — Phases 11-13 (shipped 2026-02-02)
- 🚧 **v3.0 Restore the Soul** — Phases 14-17 (in progress)

## Phases

<details>
<summary>✅ v1.0 Queen Ant Colony (Phases 3-10) — SHIPPED 2026-02-02</summary>

**Full details archived in:** .planning/milestones/v1-ROADMAP.md

**Summary:**
- 8 phases (3-10), 44 plans, 156 must-haves verified
- Autonomous spawning with Bayesian meta-learning
- Pheromone communication with time-based decay
- Triple-layer memory (Working -> Short-term -> Long-term)
- Multi-perspective verification with weighted voting
- Event-driven coordination with pub/sub event bus
- Production-ready with comprehensive testing

</details>

<details>
<summary>✅ v2.0 Reactive Event Integration (Phases 11-13) — SHIPPED 2026-02-02</summary>

**Summary:**
- 3 phases (11-13), 6 plans
- Event polling integration for all worker castes
- Visual indicators and documentation path cleanup
- E2E testing with 94 verification checks across 6 workflows

### Phase 11: Event Polling Integration
**Goal**: Worker Ants detect and react to colony events by polling the event bus at execution boundaries.
**Plans**: 3/3 complete

### Phase 12: Visual Indicators & Documentation
**Goal**: Users see colony activity at a glance through emoji-based status indicators and progress bars.
**Plans**: 2/2 complete

### Phase 13: E2E Testing
**Goal**: Comprehensive manual test guide documents all core workflows with verification checks.
**Plans**: 1/1 complete

</details>

### 🚧 v3.0 Restore the Soul (In Progress)

**Milestone Goal:** Restore the sophistication, visual identity, and depth lost during the Claude-native rebuild. No new Python, no new bash scripts, no new commands — restore capabilities through JSON state, enriched command prompts, and deeper worker specs.

- [x] **Phase 14: Visual Identity** — Commands display rich formatted output with box-drawing, progress indicators, and grouped visual elements
- [x] **Phase 15: Infrastructure State** — JSON state files for errors, memory, and events integrated into core commands
- [ ] **Phase 16: Worker Knowledge** — Deep worker specs with pheromone math, specialist watcher modes, and spawning scenarios
- [ ] **Phase 17: Integration & Dashboard** — Unified status dashboard, phase review, and spawn outcome tracking

## Phase Details

### Phase 14: Visual Identity

**Goal**: Users see polished, structured output from every command — box-drawing headers, step progress, pheromone decay visualizations, and grouped worker activity.

**Depends on**: Phase 13 (v2.0 complete)

**Requirements**: VIS-01, VIS-02, VIS-03, VIS-04

**Success Criteria** (what must be TRUE):
1. Running any major command displays a box-drawing header that visually separates the output section
2. Multi-step commands (init, build, continue) show real-time step progress with checkmarks for completed steps, arrows for current step, and empty brackets for pending steps
3. Pheromone display shows each active signal with a computed decay strength bar reflecting time-based decay
4. Worker listing groups ants by status (active, idle, error) with distinct emoji indicators for each group

**Plans**: 2 plans

Plans:
- [x] 14-01: Add box-drawing headers and step progress indicators to command prompts (init.md, build.md, continue.md, status.md, phase.md)
- [x] 14-02: Add pheromone decay bar rendering and worker activity grouping to status-displaying commands

---

### Phase 15: Infrastructure State

**Goal**: Core commands read and write structured JSON state files for errors, memory, and events — establishing the data layer that workers and the dashboard consume.

**Depends on**: Phase 14

**Requirements**: ERR-01, ERR-02, ERR-03, ERR-04, MEM-01, MEM-02, MEM-03, MEM-04, EVT-01, EVT-02, EVT-03, EVT-04

**Success Criteria** (what must be TRUE):
1. Running `/ant:init` on a fresh project creates errors.json, memory.json, and events.json with correct initial schemas in `.aether/data/`
2. When a build encounters a failure, the error is recorded in errors.json with category, severity, description, root cause, and phase
3. After 3 errors of the same category accumulate, the pattern is flagged and surfaced in subsequent status checks
4. Running `/ant:continue` at a phase boundary extracts learnings and stores them in memory.json before advancing
5. State-changing commands (init, build, continue) write event records to events.json with type, source, and timestamp

**Plans**: 3 plans

Plans:
- [x] 15-01: Create JSON schemas and integrate state file initialization into init.md (errors.json, memory.json, events.json)
- [x] 15-02: Integrate error logging and event writing into build.md; add pattern flagging logic
- [x] 15-03: Integrate memory extraction into continue.md; add decision logging to commands (focus.md, redirect.md, feedback.md)

---

### Phase 16: Worker Knowledge

**Goal**: Worker specs contain deep domain knowledge — pheromone math, signal combination effects, feedback interpretation, event awareness, and spawning scenarios — enabling truly autonomous behavior without external scripts.

**Depends on**: Phase 15

**Requirements**: WATCH-01, WATCH-02, WATCH-03, WATCH-04, SPEC-01, SPEC-02, SPEC-03, SPEC-04, SPEC-05

**Success Criteria** (what must be TRUE):
1. watcher-ant.md contains 4 specialist modes (security, performance, quality, test-coverage) each with activation triggers, focus areas, severity rubric, and detection pattern checklist
2. Every worker spec includes a worked pheromone math example showing sensitivity multiplied by strength equals effective signal, with numeric values
3. Every worker spec includes a combination effects section describing behavior when conflicting pheromone signals are active simultaneously
4. Every worker spec reads events.json at startup and describes how to filter and react to recent events
5. Every worker spec includes a complete spawning scenario with a full Task tool prompt example showing recursive spec propagation

**Plans**: 3 plans

Plans:
- [ ] 16-01: Expand watcher-ant.md with 4 specialist modes (security, performance, quality, test-coverage)
- [ ] 16-02: Add pheromone math, combination effects, and feedback interpretation to all 6 worker specs
- [ ] 16-03: Add event awareness and spawning scenario examples to all 6 worker specs

---

### Phase 17: Integration & Dashboard

**Goal**: Users see the full colony state through an integrated dashboard, get phase reviews before advancing, and benefit from spawn outcome tracking that improves autonomous decisions over time.

**Depends on**: Phase 15, Phase 16

**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04, REV-01, REV-02, REV-03, SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04

**Success Criteria** (what must be TRUE):
1. Running `/ant:status` displays a unified dashboard with sections for workers, pheromones, errors, memory, and events — all populated from live JSON state files
2. The pheromone section of the dashboard shows each active signal with a computed decay bar and numeric strength
3. The error section shows recent errors from errors.json and highlights any flagged patterns
4. Running `/ant:continue` shows a phase completion summary (tasks completed, key decisions, errors encountered) before advancing to the next phase
5. COLONY_STATE.json includes spawn_outcomes per caste with alpha/beta parameters, and workers check spawn confidence before spawning

**Plans**: 3 plans

Plans:
- [ ] 17-01: Rebuild status.md as a full colony health dashboard reading all JSON state files
- [ ] 17-02: Add phase review workflow to continue.md showing completion summary and learning extraction
- [ ] 17-03: Add spawn outcome tracking to COLONY_STATE.json, build.md, and continue.md with Bayesian confidence

## Progress

**Execution Order:**
Phases execute in numeric order: 14 -> 15 -> 16 -> 17

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 3-10 | v1.0 | 44/44 | Complete | 2026-02-02 |
| 11. Event Polling Integration | v2.0 | 3/3 | Complete | 2026-02-02 |
| 12. Visual Indicators & Documentation | v2.0 | 2/2 | Complete | 2026-02-02 |
| 13. E2E Testing | v2.0 | 1/1 | Complete | 2026-02-02 |
| 14. Visual Identity | v3.0 | 2/2 | Complete | 2026-02-03 |
| 15. Infrastructure State | v3.0 | 3/3 | Complete | 2026-02-03 |
| 16. Worker Knowledge | v3.0 | 0/3 | Not started | - |
| 17. Integration & Dashboard | v3.0 | 0/3 | Not started | - |

---

*Aether: Queen Ant Colony — Autonomous Emergence in Claude-Native Form*
