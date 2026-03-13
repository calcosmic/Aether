# Milestones

## v1.0 Aether Colony Wiring (Shipped: 2026-03-07)

**Phases completed:** 5 phases, 11 plans, 17 tasks
**Timeline:** 22 days (2026-02-13 -> 2026-03-07)
**Lines changed:** 8,378 insertions, 92 deletions across 41 files
**Integration tests:** 45 tests across 5 test files

**Key accomplishments:**
- Instinct pipeline: high-confidence patterns from continue auto-create instincts that reach builders via colony-prime
- Learnings injection: validated phase learnings from previous phases auto-inject into builder prompts
- Context expansion: CONTEXT.md decisions and blocker flags automatically reach builders
- Pheromone auto-emission: decisions, errors, and success patterns auto-emit pheromones during continue
- Wisdom promotion: observations auto-promote to QUEEN.md and promoted wisdom reaches builders

**Delivered:** Complete self-improving colony pipeline -- learnings capture through instinct creation, pheromone signaling, wisdom promotion, context assembly, and builder injection. All 12 v1 requirements satisfied.

---


## v1.1 Oracle Deep Research (Shipped: 2026-03-13)

**Phases completed:** 6 phases, 13 plans, 28 tasks
**Timeline:** 1 day (2026-03-13)
**Lines changed:** 17,013 insertions, 1,603 deletions across 68 files
**Oracle tests:** 168 tests (87 Ava + 81 bash assertions)

**Key accomplishments:**
- Structured state architecture: state.json, plan.json, gaps.md, synthesis.md, and research-plan.md replace flat progress.md append model
- Phase-aware iteration prompts with gap-driven targeting, 6-tier confidence rubric, and depth enforcement (read-before-write)
- Multi-signal convergence detection with diminishing returns, synthesis-on-every-exit path, and JSON recovery from backups
- Source tracking and trust layer with inline citations, multi-source verification, and trust ratio reporting
- Mid-session steering via pheromone signals with configurable strategy (breadth/depth/adaptive) and focus areas
- Colony knowledge integration with promote-to-colony pipeline (80%+ threshold) and 5 research strategy templates

**Delivered:** Complete deep research engine -- structured RALF-loop pattern with gap-driven iteration, source verification, configurable strategy, pheromone steering, and colony knowledge promotion. All 20 v1.1 requirements satisfied.

**Known tech debt:**
- REQUIREMENTS.md checkboxes not updated during execution
- OpenCode wizard missing stale session guard (Steps 1.5, 2.5)
- promote_to_colony function in oracle.sh is dead code (wizard duplicates inline by design)
- 1 pre-existing test failure in context-continuity (predates v1.1)

---

