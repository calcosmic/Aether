# Requirements: Aether v1.23 Daily Driver Reliability

**Defined:** 2026-05-20
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1 Requirements

Requirements for v1.23 Daily Driver Reliability milestone. Each maps to roadmap phases.

### Silent Pipeline (PIPE)

- [x] **PIPE-01**: All playbook CLI invocations produce honest errors -- no `2>/dev/null || true` suppression on data-persistence calls (learning, midden, pheromone, memory, spawn tracking)
- [x] **PIPE-02**: All state writes use `state-mutate` with targeted jq -- no raw "Write COLONY_STATE.json" from LLM context
- [x] **PIPE-03**: Pending decisions are scoped by session across all readers -- not just discuss.go
- [x] **PIPE-04**: Session cleanup on `/ant-init` clears stale session.json from prior colonies
- [x] **PIPE-05**: Event array cap enforced at the storage layer, not dependent on every caller

### Command Classification (CATALOG)

- [ ] **CATALOG-01**: Command reliability matrix built from `audit-catalog`, current docs, and historical tags v1.10-v1.21 plus v5.4.0
- [ ] **CATALOG-02**: Every command classified as public lifecycle, public utility, internal runtime, alias, or deprecated
- [ ] **CATALOG-03**: Classification integrated into `audit-catalog` output for ongoing verification

### Critical Test Coverage (TEST)

- [ ] **TEST-01**: Test files added for 12 HIGH-severity untested source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
- [ ] **TEST-02**: Every public lifecycle command has at least one smoke or fixture test with executable evidence
- [ ] **TEST-03**: Test matrix documents which commands are covered and how

### Learning and Workflow Restoration (WORKFLOW)

- [ ] **WORKFLOW-01**: Learning extraction lifecycle verified or restored -- hypothesis/validated/disproven tracking in continue path
- [ ] **WORKFLOW-02**: Worker context injection verified -- workers receive pheromones, skills, survey data, colony goal, and phase description
- [ ] **WORKFLOW-03**: Oracle promote-to-colony pipeline verified end-to-end (findings -> instincts -> learnings -> QUEEN.md -> hive)
- [ ] **WORKFLOW-04**: Build workflow produces correct state mutations, ceremony output, and spawn records
- [ ] **WORKFLOW-05**: Continue workflow verifies work, extracts learnings, and advances phase correctly
- [ ] **WORKFLOW-06**: Plan workflow produces grounded plans referencing real files
- [ ] **WORKFLOW-07**: Colonize workflow produces complete territory survey
- [ ] **WORKFLOW-08**: Autopilot (run) implements all 10 Classic pause conditions
- [ ] **WORKFLOW-09**: Seal ceremony completes lifecycle with wisdom promotion
- [ ] **WORKFLOW-10**: Entomb archives colony to chambers correctly
- [ ] **WORKFLOW-11**: Swarm 4-scout cross-comparison and confidence ranking verified

### Queen Execution Policy (QUEEN)

- [ ] **QUEEN-01**: Execution policy documentation aligned with Go runtime -- CLAUDE.md matches actual VerificationDepth behavior
- [ ] **QUEEN-02**: Playbook spawning instructions match Go caste relevance system -- no contradictions
- [ ] **QUEEN-03**: Deterministic commands (display, signal, session, admin) verified to have zero agent spawning
- [ ] **QUEEN-04**: Continue gate spawning minimized -- no over-spawning at standard depth

### Runtime Safety (RUNTIME)

- [ ] **RUNTIME-01**: Worker artifact loss on interrupt has a recovery path -- completed work is not silently lost
- [ ] **RUNTIME-02**: Worktree merge-back runs at build-complete, not only at continue-advance
- [ ] **RUNTIME-03**: Provider API errors classified correctly before JSON parsing -- auth failures reported as auth failures
- [ ] **RUNTIME-04**: Pre-build check detects orphaned worktree branches from prior interrupted builds
- [ ] **RUNTIME-05**: Status command reports unreconciled worker changes

### End-to-End Proof (PROOF)

- [ ] **PROOF-01**: Full lifecycle smoke test in a separate downstream repo proves Aether works without modifying itself during the run
- [ ] **PROOF-02**: End-to-end tests through TS host for 5 lifecycle commands (build, continue, seal, plan, colonize)
- [ ] **PROOF-03**: Public utility commands (9 remaining) have test files with executable evidence

## v2 Requirements

Deferred to future milestone.

### Deferred Ceremony

- **CEREMONY-01**: Optional Sage analytics and Chronicler audit at seal ceremony
- **CEREMONY-02**: Oracle wizard 7-question setup with template-specific synthesis
- **CEREMONY-03**: Init charter ceremony with approval flow and governance detection

### Deferred Platform

- **PLATFORM-01**: Codex runtime-native testing improvements
- **PLATFORM-02**: OpenCode parity gap resolution
- **PLATFORM-03**: Cross-platform visual output comparison tests

## Out of Scope

| Feature | Reason |
|---------|--------|
| New features or capabilities | Hard constraint: no new features until existing surface is reliable |
| Pheromone markets / reputation exchange | Explicitly deferred in PROJECT.md |
| Swarm memory beyond hive/wisdom path | Explicitly deferred in PROJECT.md |
| Federation / inter-colony coordination | Explicitly deferred in PROJECT.md |
| Self-mutating agents / evolution engine | Explicitly deferred in PROJECT.md |
| Cross-colony ledger sharing | Code-specific paths go stale across repos |
| Ledger web UI | CLI-only for now |
| Web dashboard | Future consideration |
| Reverting Go runtime to shell | Go architecture is correct -- restore behavior within it |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PIPE-01 | Phase 145 | Complete |
| PIPE-02 | Phase 145 | Complete |
| PIPE-03 | Phase 145 | Complete |
| PIPE-04 | Phase 145 | Complete |
| PIPE-05 | Phase 145 | Complete |
| CATALOG-01 | Phase 146 | Pending |
| CATALOG-02 | Phase 146 | Pending |
| CATALOG-03 | Phase 146 | Pending |
| TEST-01 | Phase 147 | Pending |
| TEST-02 | Phase 147 | Pending |
| TEST-03 | Phase 147 | Pending |
| WORKFLOW-01 | Phase 148 | Pending |
| WORKFLOW-02 | Phase 148 | Pending |
| WORKFLOW-03 | Phase 148 | Pending |
| WORKFLOW-04 | Phase 148 | Pending |
| WORKFLOW-05 | Phase 148 | Pending |
| WORKFLOW-06 | Phase 148 | Pending |
| WORKFLOW-07 | Phase 148 | Pending |
| WORKFLOW-08 | Phase 148 | Pending |
| WORKFLOW-09 | Phase 148 | Pending |
| WORKFLOW-10 | Phase 148 | Pending |
| WORKFLOW-11 | Phase 148 | Pending |
| QUEEN-01 | Phase 149 | Pending |
| QUEEN-02 | Phase 149 | Pending |
| QUEEN-03 | Phase 149 | Pending |
| QUEEN-04 | Phase 149 | Pending |
| RUNTIME-01 | Phase 150 | Pending |
| RUNTIME-02 | Phase 150 | Pending |
| RUNTIME-03 | Phase 150 | Pending |
| RUNTIME-04 | Phase 150 | Pending |
| RUNTIME-05 | Phase 150 | Pending |
| PROOF-01 | Phase 151 | Pending |
| PROOF-02 | Phase 151 | Pending |
| PROOF-03 | Phase 151 | Pending |

**Coverage:**
- v1 requirements: 30 total
- Mapped to phases: 30
- Unmapped: 0

---
*Requirements defined: 2026-05-20*
*Last updated: 2026-05-20 after roadmap creation*
