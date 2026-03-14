# Roadmap: Aether

## Milestones

- v1.0 **Aether Colony Wiring** -- Phases 1-5 (shipped 2026-03-07)
- v1.1 **Oracle Deep Research** -- Phases 6-11 (shipped 2026-03-13)
- v1.2 **Integration Gaps** -- Phases 12-14 (in progress)

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

<details>
<summary>v1.1 Oracle Deep Research (Phases 6-11) -- SHIPPED 2026-03-13</summary>

- [x] Phase 6: State Architecture Foundation (2/2 plans) -- completed 2026-03-13
- [x] Phase 7: Iteration Prompt Engineering (2/2 plans) -- completed 2026-03-13
- [x] Phase 8: Orchestrator Upgrade (2/2 plans) -- completed 2026-03-13
- [x] Phase 9: Source Tracking and Trust Layer (2/2 plans) -- completed 2026-03-13
- [x] Phase 10: Steering Integration (2/2 plans) -- completed 2026-03-13
- [x] Phase 11: Colony Knowledge Integration (3/3 plans) -- completed 2026-03-13

Full details: `.planning/milestones/v1.1-ROADMAP.md`

</details>

### v1.2 Integration Gaps (In Progress)

**Milestone Goal:** Colony learning loops produce visible output -- decisions, instincts, midden entries, and auto-pheromones accumulate naturally during build/continue cycles.

- [x] **Phase 12: Success Capture and Colony-Prime Enrichment** - Wire memory-capture success events and rolling-summary into the colony knowledge pipeline (completed 2026-03-14)
- [x] **Phase 13: Midden Write Path Expansion** - All failure types write to midden with intra-phase threshold checks (completed 2026-03-14)
- [x] **Phase 14: Decision-Pheromone and Learning-Instinct Verification** - Verify and fix the decision-to-pheromone dedup and recurrence-calibrated instinct confidence (completed 2026-03-14)

## Phase Details

### Phase 12: Success Capture and Colony-Prime Enrichment
**Goal**: Workers gain awareness of recent colony activity, and success events enter the memory pipeline for the first time
**Depends on**: Nothing (first v1.2 phase -- purely additive, cannot break existing behavior)
**Requirements**: MEM-01, MEM-02
**Success Criteria** (what must be TRUE):
  1. After a build where chaos reports strong resilience, learning-observations.json contains a new success-type entry from build-verify
  2. After a build completes with pattern synthesis, learning-observations.json contains a new success-type entry from build-complete
  3. Colony-prime output includes the last 5 rolling-summary entries so builders see recent colony activity in their prompt
  4. Existing failure-path memory-capture calls still fire unchanged (no regression)
**Plans**: 2 plans

Plans:
- [x] 12-01-PLAN.md -- Wire success capture at build-verify (chaos resilience) and build-complete (pattern synthesis)
- [x] 12-02-PLAN.md -- Add rolling-summary section to colony-prime output

### Phase 13: Midden Write Path Expansion
**Goal**: Midden data reflects actual colony failure patterns across all agent types, not just builder failures
**Depends on**: Nothing (can run in parallel with Phase 14 -- edits different playbook files)
**Requirements**: MID-01, MID-02, MID-03
**Success Criteria** (what must be TRUE):
  1. Watcher failures, Chaos failures, verification failures, Gatekeeper findings, and Auditor findings all produce entries in midden.json via midden-write
  2. When a builder abandons an approach during a build, the approach change is captured in both midden.json and learning-observations.json
  3. During a build wave, if 3+ midden entries share the same error category, a REDIRECT pheromone is emitted mid-build (not deferred to continue)
  4. Existing builder failure midden-write calls in build-wave.md still fire unchanged (no regression)
**Plans**: 2 plans

Plans:
- [x] 13-01-PLAN.md -- Wire midden-write at all failure points and add approach-change capture
- [x] 13-02-PLAN.md -- Add intra-phase midden threshold check for mid-build REDIRECT emission

### Phase 14: Decision-Pheromone and Learning-Instinct Verification
**Goal**: Decision pheromones emit reliably after continue, and instinct confidence reflects actual recurrence evidence
**Depends on**: Nothing (can run in parallel with Phase 13 -- edits different playbook files)
**Requirements**: DEC-01, LRN-01
**Success Criteria** (what must be TRUE):
  1. After a continue run that logged decisions, pheromones.json contains new FEEDBACK pheromones corresponding to those decisions (deduplication works correctly)
  2. When an instinct is created from a learning observed multiple times, its confidence score is higher than one observed only once (recurrence-calibrated, not fixed 0.7)
  3. Instinct confidence for a learning with observation_count=1 starts at 0.7, and increases with each additional observation up to a cap of 0.9
  4. The deduplication format alignment in continue-advance Step 2.1b correctly prevents duplicate pheromones for already-emitted decisions
**Plans**: 2 plans

Plans:
- [ ] 14-01-PLAN.md -- Align decision pheromone format between context-update and Step 2.1b for reliable dedup
- [ ] 14-02-PLAN.md -- Add recurrence-calibrated instinct confidence to learning-promote-auto

## Progress

**Execution Order:**
Phases 13 and 14 can be parallelized after Phase 12 completes (they edit different playbook files with no shared call sites).

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Instinct Pipeline | v1.0 | 3/3 | Complete | 2026-03-06 |
| 2. Learnings Injection | v1.0 | 2/2 | Complete | 2026-03-06 |
| 3. Context Expansion | v1.0 | 2/2 | Complete | 2026-03-06 |
| 4. Pheromone Auto-Emission | v1.0 | 2/2 | Complete | 2026-03-06 |
| 5. Wisdom Promotion | v1.0 | 2/2 | Complete | 2026-03-07 |
| 6. State Architecture Foundation | v1.1 | 2/2 | Complete | 2026-03-13 |
| 7. Iteration Prompt Engineering | v1.1 | 2/2 | Complete | 2026-03-13 |
| 8. Orchestrator Upgrade | v1.1 | 2/2 | Complete | 2026-03-13 |
| 9. Source Tracking and Trust Layer | v1.1 | 2/2 | Complete | 2026-03-13 |
| 10. Steering Integration | v1.1 | 2/2 | Complete | 2026-03-13 |
| 11. Colony Knowledge Integration | v1.1 | 3/3 | Complete | 2026-03-13 |
| 12. Success Capture and Colony-Prime Enrichment | v1.2 | 2/2 | Complete | 2026-03-14 |
| 13. Midden Write Path Expansion | v1.2 | 2/2 | Complete | 2026-03-14 |
| 14. Decision-Pheromone and Learning-Instinct Verification | v1.2 | Complete    | 2026-03-14 | - |
