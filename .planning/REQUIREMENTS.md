# Requirements: Aether v4.1

**Defined:** 2026-02-03
**Core Value:** Autonomous Emergence — Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v1 Requirements

Requirements for v4.1 Cleanup & Enforcement. Each maps to roadmap phases.

### Cleanup

- [x] **CLEAN-01**: Wire pheromone-decay into plan.md, pause-colony.md, resume-colony.md, colonize.md — replacing inline decay formulas with aether-utils.sh pheromone-batch calls
- [x] **CLEAN-02**: Wire memory-compress into continue.md — replacing manual array truncation logic with aether-utils.sh memory-compress call
- [x] **CLEAN-03**: Wire error-pattern-check into build.md — replacing manual error categorization with aether-utils.sh error-pattern-check call
- [x] **CLEAN-04**: Wire error-summary into continue.md and build.md — replacing manual error counting with aether-utils.sh error-summary call
- [x] **CLEAN-05**: Remove pheromone-combine, memory-token-count, memory-search, error-dedup from aether-utils.sh

### Enforcement

- [x] **ENFO-01**: Add spawn-check subcommand to aether-utils.sh — reads COLONY_STATE.json, checks worker count (<= 5) and spawn depth (<= 3), returns pass/fail JSON
- [x] **ENFO-02**: Update all 6 worker specs to call spawn-check before spawning — hard gate that prevents spawn if check fails
- [x] **ENFO-03**: Add pheromone-validate subcommand to aether-utils.sh — checks non-empty content, minimum length (>= 20 chars), returns pass/fail JSON
- [x] **ENFO-04**: Update continue.md auto-pheromone step to call pheromone-validate before writing pheromone
- [x] **ENFO-05**: Add post-action validation checklist to worker specs — deterministic checks (state validated, spawn limits checked) that workers must complete before reporting done

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Enforcement

- **ENFO-06**: Automated compliance scoring — shell utility that reads worker output and scores spec compliance
- **ENFO-07**: Pheromone content quality scoring — semantic analysis of pheromone descriptions beyond length checks

## Out of Scope

| Feature | Reason |
|---------|--------|
| Enforcing LLM reasoning quality | Can't deterministically check "did the LLM think carefully?" — judgment stays with prompts |
| New subcommands beyond spawn-check and pheromone-validate | Keep utility layer thin (<300 lines) |
| Rewriting worker specs from scratch | Targeted edits to add enforcement gates, not full rewrites |
| New commands | v4.0 decision: enrich existing 12 commands, don't add new ones |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLEAN-01 | Phase 22 | Complete |
| CLEAN-02 | Phase 22 | Complete |
| CLEAN-03 | Phase 22 | Complete |
| CLEAN-04 | Phase 22 | Complete |
| CLEAN-05 | Phase 22 | Complete |
| ENFO-01 | Phase 23 | Complete |
| ENFO-02 | Phase 23 | Complete |
| ENFO-03 | Phase 23 | Complete |
| ENFO-04 | Phase 23 | Complete |
| ENFO-05 | Phase 23 | Complete |

**Coverage:**
- v1 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0

---
*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after Phase 23 completion*
