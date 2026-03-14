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


## v1.2 Integration Gaps (Shipped: 2026-03-14)

**Phases completed:** 3 phases, 6 plans, 12 tasks
**Timeline:** 1 day (2026-03-14)
**Lines changed:** 5,330 insertions, 1,221 deletions across 36 files
**New tests:** 7 tests (3 decision-dedup + 4 instinct-confidence)
**Total tests:** 537 passing

**Key accomplishments:**
- Success capture pipeline: chaos resilience and pattern synthesis events now enter learning-observations.json via memory-capture "success" — the first success-type entries in the memory pipeline
- Colony-prime RECENT ACTIVITY: builders now see the last 5 rolling-summary entries in their prompt, giving workers awareness of recent colony activity
- Midden write path expansion: all failure types (Builder, Chaos, Watcher, Gatekeeper, Auditor, approach-change) now write structured entries to midden.json — not just builder failures
- Intra-phase midden threshold: when 3+ failures share the same error category during a build wave, a REDIRECT pheromone emits mid-build (capped at 3, deduped via auto:error)
- Decision pheromone dedup alignment: context-update and continue-advance Step 2.1b now use matching format ("[decision] X" with auto:decision source) for reliable deduplication
- Recurrence-calibrated instinct confidence: learning-promote-auto computes confidence from observation_count using min(0.7 + (count-1)*0.05, 0.9) instead of fixed 0.6

**Delivered:** Complete integration wiring — colony learning loops now produce visible output. Decisions auto-convert to pheromones, learnings create calibrated instincts, midden captures all failure types with mid-build threshold checks, and success events enter the memory pipeline. All 7 v1.2 requirements satisfied.

**Known tech debt:**
- build-full.md (monolithic mirror) missing both MEM-01 success capture blocks from split playbooks
- REQUIREMENTS.md checkboxes not updated during execution (cosmetic)
- DEC-01 dedup runs at continue time only (intentional design)

---

