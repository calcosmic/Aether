# Roadmap: Aether Colony Wiring

## Overview

Aether has all the infrastructure for a self-improving colony -- learnings capture, instinct creation, pheromone signaling, wisdom promotion, context assembly -- but the pieces are disconnected. This roadmap wires them together through five vertical pipeline phases, each delivering a complete data flow from capture to builder injection. Every phase is independently verifiable: create the data, then confirm it reaches workers.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Instinct Pipeline** - High-confidence patterns auto-create instincts that reach builders (completed 2026-03-06)
- [x] **Phase 2: Learnings Injection** - Phase learnings from previous phases flow into builder prompts (completed 2026-03-06)
- [x] **Phase 3: Context Expansion** - CONTEXT.md decisions and blocker flags reach builders (completed 2026-03-06)
- [x] **Phase 4: Pheromone Auto-Emission** - Decisions, errors, and success patterns auto-emit pheromones (completed 2026-03-06)
- [ ] **Phase 5: Wisdom Promotion** - Observations auto-promote to QUEEN.md and reach builders

## Phase Details

### Phase 1: Instinct Pipeline
**Goal**: Patterns validated with high confidence during continue automatically become instincts that builders receive in their prompts
**Depends on**: Nothing (first phase)
**Requirements**: LEARN-02, LEARN-03
**Success Criteria** (what must be TRUE):
  1. Running /ant:continue on a phase with patterns at confidence >= 0.7 creates instinct entries in COLONY_STATE.json
  2. Running /ant:build after instincts exist shows instinct guidance in the builder's prompt context
  3. Instincts created during continue include the source pattern, confidence score, and actionable guidance text
  4. colony-prime output includes an "Instincts" section when instincts exist, and omits it when none exist
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md — Fix instinct-read bug, tighten confidence thresholds, add midden/success pattern sourcing
- [x] 01-02-PLAN.md — Add domain-grouped instinct formatting to pheromone-prime, verify build visibility
- [x] 01-03-PLAN.md — Integration tests for instinct pipeline end-to-end

### Phase 2: Learnings Injection
**Goal**: Builders automatically see what was learned in previous phases, so the colony doesn't repeat mistakes or rediscover solutions
**Depends on**: Phase 1
**Requirements**: LEARN-01, LEARN-04
**Success Criteria** (what must be TRUE):
  1. Running /ant:build on phase 3 shows validated learnings from phases 1 and 2 in the builder's prompt context
  2. Learnings appear as actionable guidance (not raw JSON) in the builder prompt
  3. Only validated learnings (not rejected or pending) are injected
  4. colony-prime output includes a "Phase Learnings" section when previous phase learnings exist
**Plans**: 2 plans

Plans:
- [ ] 02-01-PLAN.md — Wire phase learnings extraction and formatting into colony-prime
- [ ] 02-02-PLAN.md — Integration tests for learnings injection end-to-end

### Phase 3: Context Expansion
**Goal**: Key decisions recorded in CONTEXT.md and escalated blocker flags automatically reach builders, closing the last context gaps
**Depends on**: Phase 1
**Requirements**: CTX-01, CTX-02
**Success Criteria** (what must be TRUE):
  1. Running /ant:build after decisions are recorded in CONTEXT.md shows those decisions in the builder's prompt context
  2. Escalated blocker flags appear as REDIRECT-priority warnings in builder prompts
  3. colony-prime extracts only key decisions (not the entire CONTEXT.md file) to keep prompt size manageable
  4. Blocker-originated REDIRECT warnings are distinguishable from user-set REDIRECT pheromones
**Plans**: 2 plans

Plans:
- [ ] 03-01-PLAN.md — Wire CONTEXT.md decision extraction and blocker flag injection into colony-prime
- [ ] 03-02-PLAN.md — Integration tests for context expansion end-to-end

### Phase 4: Pheromone Auto-Emission
**Goal**: The colony automatically emits pheromone signals from decisions, recurring errors, and success patterns -- no manual /ant:focus or /ant:feedback needed for routine signals
**Depends on**: Phase 1
**Requirements**: PHER-01, PHER-02, PHER-03
**Success Criteria** (what must be TRUE):
  1. Running /ant:continue after recording key decisions creates FEEDBACK pheromones for those decisions
  2. When an error pattern occurs 3+ times in midden, running /ant:continue auto-emits a REDIRECT pheromone warning about that pattern
  3. Success criteria patterns that recur across phases auto-emit FEEDBACK pheromones on recurrence
  4. Auto-emitted pheromones are marked with their source (decision/error/success) so users can distinguish them from manual pheromones
  5. Auto-emitted pheromones appear in the next /ant:build via the existing pheromone-prime pipeline
**Plans**: 2 plans

Plans:
- [ ] 04-01-PLAN.md — Wire all three auto-emission blocks (decision, error, success) into continue-advance and continue-full playbooks
- [ ] 04-02-PLAN.md — Integration tests for pheromone auto-emission from all three sources

### Phase 5: Wisdom Promotion
**Goal**: Learning observations that cross promotion thresholds automatically graduate to QUEEN.md wisdom, and that wisdom reaches builders -- completing the full learning lifecycle
**Depends on**: Phase 2, Phase 4
**Requirements**: QUEEN-01, QUEEN-02, QUEEN-03
**Success Criteria** (what must be TRUE):
  1. Running /ant:continue on a colony with observations meeting promotion thresholds creates entries in QUEEN.md
  2. Running /ant:seal on a completed colony promotes all qualifying observations to QUEEN.md
  3. Running /ant:build after QUEEN.md has entries shows queen wisdom in the builder's prompt context
  4. colony-prime output includes a "Colony Wisdom" section when QUEEN.md has entries, and omits it when empty
**Plans**: 2 plans

Plans:
- [ ] 05-01-PLAN.md — Wire batch wisdom auto-promotion into continue-finalize and seal playbooks
- [ ] 05-02-PLAN.md — Integration tests for wisdom promotion and injection end-to-end

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Instinct Pipeline | 3/3 | Complete    | 2026-03-06 |
| 2. Learnings Injection | 0/2 | Complete    | 2026-03-06 |
| 3. Context Expansion | 0/2 | Complete    | 2026-03-06 |
| 4. Pheromone Auto-Emission | 0/2 | Complete    | 2026-03-06 |
| 5. Wisdom Promotion | 0/2 | Not started | - |
