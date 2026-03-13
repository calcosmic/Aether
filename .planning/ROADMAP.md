# Roadmap: Aether

## Milestones

- v1.0 **Aether Colony Wiring** -- Phases 1-5 (shipped 2026-03-07)
- v1.1 **Oracle Deep Research** -- Phases 6-11 (in progress)

## Phases

<details>
<summary>v1.0 Aether Colony Wiring (Phases 1-5) -- SHIPPED 2026-03-07</summary>

- [x] Phase 1: Instinct Pipeline (3/3 plans) -- completed 2026-03-06
- [x] Phase 2: Learnings Injection (2/2 plans) -- completed 2026-03-06
- [x] Phase 3: Context Expansion (2/2 plans) -- completed 2026-03-06
- [x] Phase 4: Pheromone Auto-Emission (2/2 plans) -- completed 2026-03-06
- [x] Phase 5: Wisdom Promotion (2/2 plans) -- completed 2026-03-07

Full details: `.planning/milestones/v1.0-ROADMAP.md`

</details>

### v1.1 Oracle Deep Research (In Progress)

**Milestone Goal:** Rebuild the oracle as a proper Ralph-loop-based deep research engine that produces thorough, source-verified, actionable research -- rebranded for the Aether colony.

- [ ] **Phase 6: State Architecture Foundation** - Structured JSON state files that bridge context between stateless oracle iterations
- [x] **Phase 7: Iteration Prompt Engineering** - Phase-aware prompts that drive gap-targeted, deepening research across iterations (completed 2026-03-13)
- [ ] **Phase 8: Orchestrator Upgrade** - Multi-signal convergence detection and intelligent loop control in oracle.sh
- [ ] **Phase 9: Source Tracking and Trust Layer** - Citation tracking and multi-source verification for every factual claim
- [x] **Phase 10: Steering Integration** - Mid-session research control via pheromone signals and configurable strategy (completed 2026-03-13)
- [ ] **Phase 11: Colony Knowledge Integration and Output Polish** - Research findings promote to colony knowledge; adaptive structured reports

## Phase Details

### Phase 6: State Architecture Foundation
**Goal**: Oracle iterations communicate through structured, machine-readable state files instead of flat markdown append
**Depends on**: Phase 5 (v1.0 complete)
**Requirements**: LOOP-01, INTL-01, INTL-04
**Success Criteria** (what must be TRUE):
  1. Oracle creates and maintains state.json, plan.json, gaps.md, and synthesis.md files in the oracle working directory
  2. A research topic decomposes into 3-8 tracked sub-questions with status (open/partial/answered) visible in plan.json
  3. research-plan.md is generated from plan.json showing questions, status, confidence, and next steps -- user can read it at any time to see what oracle is doing
  4. State files pass jq validation after creation and after simulated iteration updates
**Plans:** 2 plans

Plans:
- [ ] 06-01-PLAN.md -- State infrastructure: validate-oracle-state subcommand, session file updates, oracle.sh orchestrator, oracle.md prompt rewrite
- [ ] 06-02-PLAN.md -- Wizard commands and tests: update oracle wizard (Claude + OpenCode), ava unit tests, bash integration tests

### Phase 7: Iteration Prompt Engineering
**Goal**: Each oracle iteration reads structured state, targets the highest-priority knowledge gap, and writes valid state updates -- deepening research rather than appending
**Depends on**: Phase 6
**Requirements**: LOOP-02, LOOP-03, INTL-02, INTL-03
**Success Criteria** (what must be TRUE):
  1. Each iteration reads state files first, then targets the highest-priority knowledge gap (not a random or repeated topic)
  2. Oracle uses phase-aware prompts (survey / investigate / synthesize / verify) that change behavior based on research lifecycle stage
  3. After each iteration, gaps.md reflects updated unknowns and contradictions -- remaining gaps shrink or refine over successive iterations
  4. Per-question confidence scoring (0-100%) drives which areas get researched next -- lowest-confidence open questions are prioritized
  5. Running 3+ iterations on a real topic produces measurably deeper findings (not restatements of earlier iterations)
**Plans:** 2/2 plans complete

Plans:
- [ ] 07-01-PLAN.md -- Phase-aware prompt engineering: oracle.sh phase transitions, iteration counter, prompt construction; oracle.md rewrite with confidence rubric and depth enforcement
- [ ] 07-02-PLAN.md -- Phase transition and iteration tests: ava unit tests for determine_phase and build_oracle_prompt, bash integration tests for iteration counter and transitions

### Phase 8: Orchestrator Upgrade
**Goal**: oracle.sh uses structural convergence metrics to decide when research is complete, and produces useful partial results on interruption
**Depends on**: Phase 7
**Requirements**: LOOP-04, INTL-05, OUTP-02
**Success Criteria** (what must be TRUE):
  1. Convergence detection uses gap resolution rate, novelty rate, and coverage completeness -- not self-assessed confidence alone
  2. Oracle detects diminishing returns (e.g., 3 iterations with minimal new findings) and triggers strategy changes or synthesis
  3. On stop signal or max-iterations, oracle runs a synthesis pass that produces a useful structured partial report from whatever state exists
  4. State validation runs after each iteration -- malformed JSON triggers recovery, not silent corruption
**Plans:** 2 plans

Plans:
- [ ] 08-01-PLAN.md -- Convergence detection, diminishing returns, synthesis pass, signal handling, and JSON recovery in oracle.sh
- [ ] 08-02-PLAN.md -- Ava unit tests and bash integration tests for all convergence and recovery functions

### Phase 9: Source Tracking and Trust Layer
**Goal**: Every factual claim in oracle output tracks its source, and single-source claims are flagged as low confidence
**Depends on**: Phase 8
**Requirements**: TRST-01, TRST-02, TRST-03
**Success Criteria** (what must be TRUE):
  1. Every claim in the research output tracks its source (URL + title + date accessed)
  2. Single-source claims are flagged as low confidence; key claims require 2+ independent sources to be marked high confidence
  3. Final report includes a dedicated sources section with inline citations linking findings to their origins
**Plans:** 2 plans

Plans:
- [ ] 09-01-PLAN.md -- Source tracking implementation: compute_trust_scores in oracle.sh, source tracking prompt in oracle.md, updated synthesis/research-plan, backward-compatible validation, wizard v1.1 template
- [ ] 09-02-PLAN.md -- Trust layer tests: Ava unit tests and bash integration tests for trust scoring, validation, and backward compatibility

### Phase 10: Steering Integration
**Goal**: Users can steer oracle research mid-session via pheromone signals and configure research strategy without restarting
**Depends on**: Phase 8
**Requirements**: STRC-01, STRC-02, STRC-03
**Success Criteria** (what must be TRUE):
  1. User can emit FOCUS/REDIRECT/FEEDBACK pheromone signals that the oracle reads between iterations and acts on in the next iteration
  2. User can configure search strategy (breadth-first, depth-first, or adaptive) in the oracle wizard before research begins
  3. User can set focus areas to prioritize certain aspects of the research topic, and oracle visibly prioritizes those areas in subsequent iterations
**Plans:** 2/2 plans complete

Plans:
- [ ] 10-01-PLAN.md -- Steering implementation: read_steering_signals in oracle.sh, build_oracle_prompt strategy modifier, wizard strategy/focus questions, validate-oracle-state extensions
- [ ] 10-02-PLAN.md -- Steering tests: Ava unit tests and bash integration tests for steering signal reading, strategy handling, and validation

### Phase 11: Colony Knowledge Integration and Output Polish
**Goal**: High-confidence research findings promote to colony instincts and learnings; final output adapts its structure to the specific research topic
**Depends on**: Phase 9, Phase 10
**Requirements**: COLN-01, COLN-02, OUTP-01, OUTP-03
**Success Criteria** (what must be TRUE):
  1. After oracle completion, high-confidence findings can be promoted to colony instincts and learnings via a deliberate user-triggered step
  2. Pre-built research strategy templates exist for common patterns (tech evaluation, architecture review, bug investigation, best practices) and are selectable in the wizard
  3. Final output is a structured, synthesized report with executive summary, sections organized by sub-question, and findings grouped by confidence level
  4. Output structure adapts to the specific research topic -- a tech evaluation looks different from a bug investigation
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 6 -> 7 -> 8 -> 9 -> 10 -> 11
Note: Phase 10 depends on Phase 8 (not Phase 9), so Phases 9 and 10 could execute in parallel.

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Instinct Pipeline | v1.0 | 3/3 | Complete | 2026-03-06 |
| 2. Learnings Injection | v1.0 | 2/2 | Complete | 2026-03-06 |
| 3. Context Expansion | v1.0 | 2/2 | Complete | 2026-03-06 |
| 4. Pheromone Auto-Emission | v1.0 | 2/2 | Complete | 2026-03-06 |
| 5. Wisdom Promotion | v1.0 | 2/2 | Complete | 2026-03-07 |
| 6. State Architecture Foundation | v1.1 | 0/2 | Not started | - |
| 7. Iteration Prompt Engineering | v1.1 | Complete    | 2026-03-13 | - |
| 8. Orchestrator Upgrade | v1.1 | 0/2 | Not started | - |
| 9. Source Tracking and Trust Layer | v1.1 | 0/2 | Not started | - |
| 10. Steering Integration | v1.1 | Complete    | 2026-03-13 | - |
| 11. Colony Knowledge Integration and Output Polish | v1.1 | 0/TBD | Not started | - |
