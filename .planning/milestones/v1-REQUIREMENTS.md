# Requirements Archive: v1 Queen Ant Colony

**Archived:** 2026-02-02
**Status:** ✅ SHIPPED

This is the archived requirements specification for v1.
For current requirements, see `.planning/REQUIREMENTS.md` (created for next milestone).

---

# Requirements: Aether v2

**Defined:** 2026-02-01
**Core Value:** Autonomous Emergence - Worker Ants autonomously spawn other Worker Ants; Queen provides signals not commands

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Command System

- [x] **CMD-01**: User can initialize project with `/ant:init <goal>` — v1
- [x] **CMD-02**: User can view colony status with `/ant:status` — v1
- [x] **CMD-03**: User can view phase details with `/ant:phase [N]` — v1
- [x] **CMD-04**: User can execute phase with `/ant:execute <N>` — v1
- [x] **CMD-05**: User can emit focus pheromone with `/ant:focus <area>` — v1
- [x] **CMD-06**: User can emit redirect pheromone with `/ant:redirect <pattern>` — v1
- [x] **CMD-07**: User can emit feedback pheromone with `/ant:feedback <msg>` — v1

### Pheromone Signal System

- [x] **PH-01**: System emits INIT pheromone when user initializes project — v1
- [x] **PH-02**: INIT pheromone persists until phase complete — v1
- [x] **PH-03**: FOCUS pheromone decays over 1 hour — v1
- [x] **PH-04**: REDIRECT pheromone decays over 24 hours — v1
- [x] **PH-05**: FEEDBACK pheromone decays over 6 hours — v1
- [x] **PH-06**: Worker Ants detect pheromones based on sensitivity profile — v1
- [x] **PH-07**: Effective strength = signal strength × ant sensitivity — v1
- [x] **PH-08**: Colony responds to pheromone signal combinations — v1

### State Persistence

- [x] **STATE-01**: Colony state stored in `.aether/COLONY_STATE.json` — v1
- [x] **STATE-02**: Pheromone signals stored in `.aether/data/pheromones.json` — v1
- [x] **STATE-03**: Worker Ant states stored in `.aether/data/worker_ants.json` — v1
- [x] **STATE-04**: Memory stored in `.aether/data/memory.json` — v1
- [x] **STATE-05**: State persists across context refreshes — v1
- [x] **STATE-06**: File locking prevents race condition corruption — v1
- [x] **STATE-07**: Atomic writes prevent partial state corruption — v1

### Worker Ant Castes

- [x] **CASTE-01**: Colonizer Ant colonizes codebase and builds semantic index — v1
- [x] **CASTE-02**: Route-setter Ant creates phase structures and task breakdown — v1
- [x] **CASTE-03**: Builder Ant implements code and runs commands — v1
- [x] **CASTE-04**: Watcher Ant validates implementation and tests — v1
- [x] **CASTE-05**: Scout Ant gathers information and searches docs — v1
- [x] **CASTE-06**: Architect Ant compresses memory and extracts patterns — v1
- [x] **CASTE-07**: Each caste can spawn specialists based on capability gaps — v1

### Phase Execution

- [x] **PHASE-01**: Colony operates in phases with boundaries — v1
- [x] **PHASE-02**: Emergence occurs within phases (Queen does not intervene) — v1
- [x] **PHASE-03**: Phase boundaries trigger Queen check-in — v1
- [x] **PHASE-04**: Queen can review at phase boundaries via `/ant:phase` — v1
- [x] **PHASE-05**: Queen can adjust pheromones between phases — v1
- [x] **PHASE-06**: Next phase adapts based on previous phase learnings — v1

### Autonomous Agent Spawning

- [x] **SPAWN-01**: Worker Ants detect capability gaps autonomously — v1
- [x] **SPAWN-02**: System analyzes task requirements vs own capabilities — v1
- [x] **SPAWN-03**: System determines specialist type needed — v1
- [x] **SPAWN-04**: System spawns specialist via Task tool — v1
- [x] **SPAWN-05**: Spawned specialist inherits context from parent — v1
- [x] **SPAWN-06**: Resource budgets limit total spawns (max 10) — v1
- [x] **SPAWN-07**: Circuit breaker prevents infinite loops (depth limit 3) — v1
- [x] **SPAWN-08**: Spawn outcomes tracked for meta-learning — v1

### Triple-Layer Memory

- [x] **MEM-01**: Working Memory stores 200k tokens for current session — v1
- [x] **MEM-02**: Working Memory stores items with metadata and timestamps — v1
- [x] **MEM-03**: Short-term Memory stores 10 compressed sessions — v1
- [x] **MEM-04**: Short-term Memory uses DAST compression (2.5x ratio) — v1
- [x] **MEM-05**: Long-term Memory stores persistent patterns — v1
- [x] **MEM-06**: Long-term Memory uses maximum compression — v1
- [x] **MEM-07**: Associative links connect related items across layers — v1
- [x] **MEM-08**: Phase boundaries trigger compression (Working → Short-term) — v1
- [x] **MEM-09**: Pattern extraction triggers storage (Short-term → Long-term) — v1
- [x] **MEM-10**: LRU eviction when Short-term exceeds 10 sessions — v1
- [x] **MEM-11**: Search queries all layers and returns ranked results — v1

### Voting-Based Verification

- [x] **VOTE-01**: System spawns 4 verifier perspectives in parallel — v1
- [x] **VOTE-02**: Security-focused verifier validates security aspects — v1
- [x] **VOTE-03**: Performance-focused verifier validates performance aspects — v1
- [x] **VOTE-04**: Quality-focused validator validates code quality aspects — v1
- [x] **VOTE-05**: Test-coverage verifier validates test completeness — v1
- [x] **VOTE-06**: Each verifier casts weighted vote (APPROVE/REJECT) — v1
- [x] **VOTE-07**: Weight based on historical reliability (belief calibration) — v1
- [x] **VOTE-08**: Supermajority (67%) required for approval — v1
- [x] **VOTE-09**: System aggregates issues from all verifiers — v1
- [x] **VOTE-10**: System records vote for learning and reliability updates — v1

### Meta-Learning Loop

- [x] **META-01**: System tracks spawn outcomes (success/failure) — v1
- [x] **META-02**: System updates specialist type confidence based on outcomes — v1
- [x] **META-03**: System uses Bayesian distribution for confidence scoring — v1
- [x] **META-04**: System recommends specialists based on historical success — v1
- [x] **META-05**: System adapts recommendations over time — v1
- [x] **META-06**: Beta distribution prevents overconfidence from small samples — v1

### State Machine Orchestration

- [x] **SM-01**: Colony has explicit states (IDLE, INIT, PLANNING, EXECUTING, VERIFYING, COMPLETED, FAILED) — v1
- [x] **SM-02**: State transitions triggered by events — v1
- [x] **SM-03**: Checkpoint saved before each state transition — v1
- [x] **SM-04**: Checkpoint saved after each state transition — v1
- [x] **SM-05**: System can recover from checkpoint on failure — v1
- [x] **SM-06**: State history tracked for debugging — v1
- [x] **SM-07**: Observable state transitions for monitoring — v1

### Event-Driven Communication

- [x] **EVENT-01**: Event bus enables pub/sub communication — v1
- [x] **EVENT-02**: Worker Ants can publish events — v1
- [x] **EVENT-03**: Worker Ants can subscribe to event topics — v1
- [x] **EVENT-04**: Event filtering prevents irrelevant messages — v1
- [x] **EVENT-05**: Event logging enables debugging and replay — v1
- [x] **EVENT-06**: Async non-blocking event delivery — v1
- [x] **EVENT-07**: Event metrics track performance — v1

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Memory

- **MEM-V2-01**: Predictive context loading (anticipate what's needed next)
- **MEM-V2-02**: Forgetting mechanisms (strategic memory pruning)
- **MEM-V2-03**: Memory consolidation during idle periods

### Multi-Colony Support

- **COL-V2-01**: Multiple colonies can run simultaneously
- **COL-V2-02**: Colonies can communicate via pheromone bridges
- **COL-V2-03**: Colony federation for large-scale coordination

### Advanced Verification

- **VOTE-V2-01**: Multi-phase verification (static → dynamic → semantic)
- **VOTE-V2-02**: Cross-agent validation between colonies

## Out of Scope

| Feature | Reason |
|---------|--------|
| **Python CLI/REPL interfaces** | Replaced by Claude-native prompt commands |
| **Async/await implementation** | Claude handles concurrency via Task tool |
| **External vector databases** | Using Claude's native semantic understanding |
| **Embedding services** | No external dependencies allowed |
| **Docker/Kubernetes deployment** | Claude Code has built-in sandboxing |
| **Redis/PostgreSQL** | JSON sufficient for prototype scope |
| **Real-time dashboard UI** | Async event logging sufficient |
| **Predefined workflows** | Defeats emergence; use phased autonomy instead |
| **Direct command patterns** | Use pheromone signals instead |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CMD-01, CMD-02 | Phase 1 | Complete |
| CMD-03, CMD-04 | Phase 2 | Complete |
| CMD-05, CMD-06, CMD-07 | Phase 3 | Complete |
| PH-01, PH-02 | Phase 3 | Complete |
| PH-03, PH-04, PH-05 | Phase 3 | Complete |
| PH-06, PH-07, PH-08 | Phase 3 | Complete |
| STATE-01, STATE-02, STATE-03, STATE-04 | Phase 1 | Complete |
| STATE-05, STATE-06, STATE-07 | Phase 1 | Complete |
| CASTE-01, CASTE-02, CASTE-03, CASTE-04 | Phase 2 | Complete |
| CASTE-05, CASTE-06, CASTE-07 | Phase 2 | Complete |
| PHASE-01, PHASE-02, PHASE-03, PHASE-04 | Phase 5 | Complete |
| PHASE-05, PHASE-06 | Phase 5 | Complete |
| SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04 | Phase 6 | Complete |
| SPAWN-05, SPAWN-06, SPAWN-07, SPAWN-08 | Phase 6 | Complete |
| MEM-01, MEM-02, MEM-03, MEM-04, MEM-05 | Phase 4 | Complete |
| MEM-06, MEM-07, MEM-08, MEM-09, MEM-10, MEM-11 | Phase 4 | Complete |
| VOTE-01, VOTE-02, VOTE-03, VOTE-04, VOTE-05 | Phase 7 | Complete |
| VOTE-06, VOTE-07, VOTE-08, VOTE-09, VOTE-10 | Phase 7 | Complete |
| META-01, META-02, META-03, META-04, META-05, META-06 | Phase 8 | Complete |
| SM-01, SM-02, SM-03, SM-04, SM-05, SM-06, SM-07 | Phase 5 | Complete |
| EVENT-01, EVENT-02, EVENT-03, EVENT-04, EVENT-05 | Phase 9 | Complete |
| EVENT-06, EVENT-07 | Phase 9 | Complete |

**Coverage:**
- v1 requirements: 52 total
- Mapped to phases: 52/52 ✓
- Unmapped: 0

---

## Milestone Summary

**Shipped:** 52 of 52 v1 requirements (100%)
**Adjusted:** None - All requirements implemented as specified
**Dropped:** None

---

*Archived: 2026-02-02 as part of v1 milestone completion*
