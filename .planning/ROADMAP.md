# Roadmap: Aether — Queen Ant Colony

## Overview

Aether delivers autonomous emergence through Claude-native skill prompts, JSON state, and recursive worker spawning. v1.0 built the foundation (phases 3-10), v2.0 added reactive event integration (phases 11-13), v3.0 restored sophistication lost during the rebuild (phases 14-17), and v4.0 adds a thin shell utility layer for deterministic operations while fixing all audit-identified issues.

## Milestones

- ✅ **v1.0 Queen Ant Colony** — Phases 3-10 (shipped 2026-02-02)
- ✅ **v2.0 Reactive Event Integration** — Phases 11-13 (shipped 2026-02-02)
- ✅ **v3.0 Restore the Soul** — Phases 14-17 (shipped 2026-02-03)
- **v4.0 Hybrid Foundation** — Phases 19-21

## Phases

<details>
<summary>v1.0 Queen Ant Colony (Phases 3-10) — SHIPPED 2026-02-02</summary>

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
<summary>v2.0 Reactive Event Integration (Phases 11-13) — SHIPPED 2026-02-02</summary>

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

<details>
<summary>v3.0 Restore the Soul (Phases 14-17) — SHIPPED 2026-02-03</summary>

**Milestone Goal:** Restore the sophistication, visual identity, and depth lost during the Claude-native rebuild. No new Python, no new bash scripts, no new commands — restore capabilities through JSON state, enriched command prompts, and deeper worker specs.

- [x] **Phase 14: Visual Identity** — Commands display rich formatted output with box-drawing, progress indicators, and grouped visual elements
- [x] **Phase 15: Infrastructure State** — JSON state files for errors, memory, and events integrated into core commands
- [x] **Phase 16: Worker Knowledge** — Deep worker specs with pheromone math, specialist watcher modes, and spawning scenarios
- [x] **Phase 17: Integration & Dashboard** — Unified status dashboard, phase review, and spawn outcome tracking

### Phase 14: Visual Identity

**Goal**: Users see polished, structured output from every command — box-drawing headers, step progress, pheromone decay visualizations, and grouped worker activity.

**Depends on**: Phase 13 (v2.0 complete)

**Requirements**: VIS-01, VIS-02, VIS-03, VIS-04

**Success Criteria** (what must be TRUE):
1. Running any major command displays a box-drawing header that visually separates the output section
2. Multi-step commands (init, build, continue) show real-time step progress with checkmarks for completed steps, arrows for current step, and empty brackets for pending steps
3. Pheromone display shows each active signal with a computed decay strength bar reflecting time-based decay
4. Worker listing groups ants by status (active, idle, error) with distinct emoji indicators for each group

**Plans**: 2/2 complete

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

**Plans**: 3/3 complete

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

**Plans**: 3/3 complete

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

**Plans**: 3/3 complete

</details>

### v4.0 Hybrid Foundation

**Milestone Goal:** Add a thin shell utility layer for deterministic operations and fix all 11 audit-identified issues. The system becomes hybrid: prompts reason and decide, shell scripts compute and validate. This makes pheromone math, state validation, memory management, and error tracking reliable instead of LLM-approximated.

- [x] **Phase 19: Audit Fixes + Utility Scaffold** — Fix all 11 audit issues and create the aether-utils.sh scaffold with subcommand dispatch
- [ ] **Phase 20: Utility Modules** — Implement all 4 utility modules (pheromone math, state validation, memory ops, error tracking)
- [ ] **Phase 21: Command Integration** — Update command prompts to call utilities where deterministic results are needed

## Phase Details

### Phase 19: Audit Fixes + Utility Scaffold

**Goal**: The existing system is stable and correct -- all audit issues are resolved, state fields are canonical, and the utility script scaffold is ready for module implementation.

**Depends on**: Phase 17 (v3.0 complete)

**Requirements**: FIX-01, FIX-02, FIX-03, FIX-04, FIX-05, FIX-06, FIX-07, FIX-08, FIX-09, FIX-10, FIX-11, UTIL-01, UTIL-02, UTIL-03, UTIL-04

**Success Criteria** (what must be TRUE):
1. Running `source .aether/utils/atomic-write.sh` succeeds without errors, and `acquire_lock` / `release_lock` are available as functions (file-lock.sh sourced correctly)
2. COLONY_STATE.json has exactly one `goal` field at `.goal` and one `current_phase` field at `.current_phase` -- all commands read and write these canonical paths without referencing old/duplicate fields
3. Running `bash .aether/aether-utils.sh help` prints available subcommands and exits 0 -- the script dispatches subcommands, sources shared infrastructure, and outputs JSON on both success and error
4. Temp files created during state operations include PID and timestamp in their names (no collisions under concurrent access), and all jq operations check exit codes before writing results
5. Running a state-modifying operation creates a timestamped backup in `.aether/data/backups/` before writing, with at most 3 backups retained per file

**Plans:** 3 plans

Plans:
- [x] 19-01-PLAN.md -- Canonicalize v3 state schema and verify command field paths
- [x] 19-02-PLAN.md -- Harden atomic-write backup dir/rotation, add pheromone cleanup and validation guidance
- [x] 19-03-PLAN.md -- Create aether-utils.sh scaffold and document colony system

---

### Phase 20: Utility Modules

**Goal**: All deterministic operations that LLMs get wrong -- pheromone decay math, state schema validation, memory token management, and error pattern detection -- are handled by shell functions that produce correct, reproducible results.

**Depends on**: Phase 19

**Requirements**: PHER-01, PHER-02, PHER-03, PHER-04, PHER-05, VALID-01, VALID-02, VALID-03, VALID-04, VALID-05, VALID-06, MEM-01, MEM-02, MEM-03, ERR-01, ERR-02, ERR-03, ERR-04

**Success Criteria** (what must be TRUE):
1. Running `aether-utils pheromone-decay 1.0 3600 3600` outputs JSON with `strength` approximately 0.5 (exponential decay at one half-life), and `aether-utils pheromone-batch` reads pheromones.json and outputs all signals with their current computed strengths
2. Running `aether-utils validate-state all` checks every JSON state file against its schema and reports per-file pass/fail with specific field-level errors for any violations
3. Running `aether-utils memory-token-count` outputs an approximate token count of memory.json, and `aether-utils memory-compress` removes oldest entries when the count exceeds the threshold
4. Running `aether-utils error-add build high "Test failure in auth module"` appends a timestamped, auto-ID error record to errors.json, and `aether-utils error-pattern-check` flags categories with 3+ occurrences
5. Every subcommand outputs valid JSON to stdout on success and returns non-zero with a JSON error message on invalid input or missing files

Plans:
- [ ] 20-01
- [ ] 20-02
- [ ] 20-03
- [ ] 20-04

---

### Phase 21: Command Integration

**Goal**: Command prompts delegate deterministic operations to aether-utils.sh instead of asking the LLM to compute them -- pheromone decay, error logging, state validation, and cleanup happen via shell calls that produce reliable results.

**Depends on**: Phase 20

**Requirements**: INT-01, INT-02, INT-03, INT-04, INT-05

**Success Criteria** (what must be TRUE):
1. status.md instructs Claude to run `aether-utils pheromone-batch` via Bash tool and render decay bars from the JSON output, instead of computing decay math inline
2. build.md instructs Claude to run `aether-utils error-add` via Bash tool when logging errors, instead of manually constructing and writing error JSON
3. continue.md instructs Claude to run `aether-utils pheromone-cleanup` via Bash tool at phase boundaries, removing expired signals deterministically
4. Worker ant specs document that `aether-utils pheromone-effective` should be called via Bash tool to compute signal response strength, replacing inline multiplication
5. init.md instructs Claude to run `aether-utils validate-state all` via Bash tool after creating state files, confirming initialization correctness

Plans:
- [ ] 21-01
- [ ] 21-02

## Progress

**Execution Order:**
Phases execute in numeric order: 19 -> 20 -> 21

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 3-10 | v1.0 | 44/44 | Complete | 2026-02-02 |
| 11. Event Polling Integration | v2.0 | 3/3 | Complete | 2026-02-02 |
| 12. Visual Indicators & Documentation | v2.0 | 2/2 | Complete | 2026-02-02 |
| 13. E2E Testing | v2.0 | 1/1 | Complete | 2026-02-02 |
| 14. Visual Identity | v3.0 | 2/2 | Complete | 2026-02-03 |
| 15. Infrastructure State | v3.0 | 3/3 | Complete | 2026-02-03 |
| 16. Worker Knowledge | v3.0 | 3/3 | Complete | 2026-02-03 |
| 17. Integration & Dashboard | v3.0 | 3/3 | Complete | 2026-02-03 |
| 19. Audit Fixes + Utility Scaffold | v4.0 | 3/3 | Complete | 2026-02-03 |
| 20. Utility Modules | v4.0 | 0/4 | Pending | - |
| 21. Command Integration | v4.0 | 0/2 | Pending | - |

---

*Aether: Queen Ant Colony — Autonomous Emergence in Claude-Native Form*
