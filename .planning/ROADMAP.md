# Roadmap: Aether v4.1

## Milestones

- v1.0 MVP -- Phases 3-10 (shipped 2026-02-02)
- v2.0 Event & Visual -- Phases 11-13 (shipped 2026-02-02)
- v3.0 Rebuild -- Phases 14-18 (shipped 2026-02-03)
- v4.0 Hybrid Foundation -- Phases 19-21 (shipped 2026-02-03)
- **v4.1 Cleanup & Enforcement** -- Phases 22-23 (shipped 2026-02-03)

## Overview

v4.1 closes the gap between what aether-utils.sh provides and what commands/specs actually use. Phase 22 wires 4 orphaned-but-useful subcommands into their natural consumers and removes 4 dead ones, eliminating all inline LLM duplicates of deterministic logic. Phase 23 adds enforcement gates so that spawn limits and pheromone quality are checked by shell code, not just advisory prompt text.

## Phases

- [x] **Phase 22: Cleanup** - Wire orphaned subcommands into commands, remove dead ones, eliminate inline formulas
- [x] **Phase 23: Enforcement** - Add spawn-check and pheromone-validate subcommands, enforce spec compliance

## Phase Details

### Phase 22: Cleanup
**Goal**: Every aether-utils.sh subcommand is either consumed by a command or spec, or removed -- no orphans, no inline duplicates
**Depends on**: Phase 21 (v4.0 complete)
**Requirements**: CLEAN-01, CLEAN-02, CLEAN-03, CLEAN-04, CLEAN-05
**Success Criteria** (what must be TRUE):
  1. plan.md, pause-colony.md, resume-colony.md, and colonize.md call aether-utils.sh pheromone-batch for decay calculation instead of inline formulas
  2. continue.md calls aether-utils.sh memory-compress instead of manual array truncation logic
  3. build.md calls aether-utils.sh error-pattern-check instead of manual error categorization
  4. continue.md and build.md call aether-utils.sh error-summary instead of manual error counting
  5. pheromone-combine, memory-token-count, memory-search, and error-dedup are removed from aether-utils.sh
**Plans**: 2 plans

Plans:
- [x] 22-01-PLAN.md -- Wire pheromone-batch into 4 commands, remove 4 dead subcommands (CLEAN-01, CLEAN-05)
- [x] 22-02-PLAN.md -- Wire memory-compress, error-pattern-check, error-summary into continue.md and build.md (CLEAN-02, CLEAN-03, CLEAN-04)

### Phase 23: Enforcement
**Goal**: Worker spec instructions have deterministic enforcement gates -- spawn limits and pheromone quality are validated by shell code before actions proceed
**Depends on**: Phase 22
**Requirements**: ENFO-01, ENFO-02, ENFO-03, ENFO-04, ENFO-05
**Success Criteria** (what must be TRUE):
  1. Running aether-utils.sh spawn-check returns pass/fail JSON based on worker count (max 5) and spawn depth (max 3) from COLONY_STATE.json
  2. All 6 worker specs call spawn-check before spawning and halt if the check fails
  3. Running aether-utils.sh pheromone-validate returns pass/fail JSON checking non-empty content and minimum length (>= 20 chars)
  4. continue.md auto-pheromone step calls pheromone-validate before writing and rejects invalid pheromones
  5. Worker specs include a post-action validation checklist of deterministic checks (state validated, spawn limits checked) that must pass before reporting done
**Plans**: 2 plans

Plans:
- [x] 23-01-PLAN.md -- Add spawn-check and pheromone-validate subcommands to aether-utils.sh (ENFO-01, ENFO-03)
- [x] 23-02-PLAN.md -- Wire enforcement gates into worker specs, continue.md, and build.md (ENFO-02, ENFO-04, ENFO-05)

## Progress

**Execution Order:** 22 -> 23

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 22. Cleanup | v4.1 | 2/2 | ✓ Complete | 2026-02-03 |
| 23. Enforcement | v4.1 | 2/2 | ✓ Complete | 2026-02-03 |

---
*Roadmap created: 2026-02-03*
*Milestone: v4.1 Cleanup & Enforcement*
