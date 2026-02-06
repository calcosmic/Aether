# Requirements: Aether v5.1 System Simplification

**Defined:** 2026-02-06
**Core Value:** Stigmergic Emergence -- Worker Ants detect capability gaps and spawn specialists through pheromone-guided coordination
**Source:** M4L-AnalogWave postmortem (2026-02-05)

## v5.1 Requirements

Requirements derived from postmortem Section 5 (Recommended Fixes), ordered by severity.

### Critical Priority

- [x] **SIMP-01**: Consolidate 6 state files into single COLONY_STATE.json
  - Merge: COLONY_STATE, PROJECT_PLAN, pheromones, memory, errors, events
  - One read, one write per command
  - Event log as append-only strings within the file

- [x] **SIMP-02**: Move state updates from end-of-command to start-of-next-command
  - /ant:build writes "EXECUTING" state only
  - /ant:continue detects completed output files and updates state
  - Prevents state loss at context boundaries

### High Priority

- [ ] **SIMP-03**: Replace pheromone exponential decay with simple TTL
  - Signals use `expires_at` timestamp instead of half-life math
  - Priority field: "high" (REDIRECT), "normal" (FOCUS), "low" (FEEDBACK)
  - Filter expired signals on read, no cleanup command needed
  - Remove all sensitivity matrices from worker specs

- [ ] **SIMP-04**: Collapse 6 worker specs into single workers.md (~200 lines)
  - Preserve role concept (colonizer, scout, builder, watcher, architect, route-setter)
  - Remove sensitivity matrices, spawning protocols, visual identity systems
  - Include assignment guidelines

- [ ] **SIMP-05**: Shrink command files by 60-70%
  - Remove verbose display templates (let agent format naturally)
  - Remove redundant per-step state validation
  - Remove Bayesian spawn tracking
  - Remove step-by-step progress templates
  - Target: ~100 lines average (from ~340)

### Medium Priority

- [ ] **SIMP-06**: Reduce aether-utils.sh from 372 lines to ~80 lines
  - Keep: validate-state, error-add
  - Remove: pheromone math (decay, effective, batch, cleanup, validate)
  - Remove: memory compression, spawn check
  - Inline trivial operations (error summary, activity logging)

- [x] **SIMP-07**: Adopt output-as-state for build results
  - `.planning/phase-N/SUMMARY.md` existence = phase complete
  - State file tracks only: current phase number, colony goal, active signals
  - /ant:continue reads output files to determine completion

## Out of Scope

| Feature | Reason |
|---------|--------|
| Bayesian spawn tracking | Not enough data points to be meaningful (need 50+) |
| Caste sensitivity matrices | Replaced by plain-language priority rules |
| Half-life pheromone decay | Replaced by TTL |
| Verbose ASCII output templates | Let LLM format naturally |
| Per-step state validation | Validate once at command start |

## Preserved Features

Per postmortem Section 7 (What to Preserve):

- Parallel task decomposition (Wave 1 -> Wave 2 pattern)
- Structured phase output (`.planning/phase-N/` directories)
- Worker role concept (6 castes)
- Signal/constraint concept (simplified implementation)
- Quality gates (watcher review before advancement)
- Pause/resume capability
- Phase-based planning

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| State files per command | 6 | 1 |
| Shell utility calls per command | 2-5 | 0-1 |
| Command file avg lines | ~340 | ~100 |
| Worker spec total lines | 1,866 | ~200 |
| Total system lines | ~7,400 | ~1,800 |
| Framework context overhead | ~70% | ~20% |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SIMP-01 | Phase 33 | Complete |
| SIMP-02 | Phase 34 | Complete |
| SIMP-03 | Phase 36 | Pending |
| SIMP-04 | Phase 35 | Pending |
| SIMP-05 | Phase 34 (build, continue), Phase 37 (remaining) | Partial (build, continue complete) |
| SIMP-06 | Phase 37 | Pending |
| SIMP-07 | Phase 34 | Complete |

**Coverage:**
- v5.1 requirements: 7 total
- Mapped to phases: 7/7
- Unmapped: 0

---
*Requirements defined: 2026-02-06*
*Traceability updated: 2026-02-06*
*Source: COLONY_POSTMORTEM.md Sections 5-6*
