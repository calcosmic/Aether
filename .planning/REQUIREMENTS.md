# Requirements: Aether v2

**Defined:** 2026-02-01
**Core Value:** Autonomous Emergence - Worker Ants autonomously spawn other Worker Ants; Queen provides signals not commands

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Command System

- [ ] **CMD-01**: User can initialize project with `/ant:init <goal>`
- [ ] **CMD-02**: User can view colony status with `/ant:status`
- [ ] **CMD-03**: User can view phase details with `/ant:phase [N]`
- [ ] **CMD-04**: User can execute phase with `/ant:execute <N>`
- [x] **CMD-05**: User can emit focus pheromone with `/ant:focus <area>`
- [x] **CMD-06**: User can emit redirect pheromone with `/ant:redirect <pattern>`
- [x] **CMD-07**: User can emit feedback pheromone with `/ant:feedback <msg>`

### Pheromone Signal System

- [x] **PH-01**: System emits INIT pheromone when user initializes project
- [x] **PH-02**: INIT pheromone persists until phase complete
- [x] **PH-03**: FOCUS pheromone decays over 1 hour
- [x] **PH-04**: REDIRECT pheromone decays over 24 hours
- [x] **PH-05**: FEEDBACK pheromone decays over 6 hours
- [x] **PH-06**: Worker Ants detect pheromones based on sensitivity profile
- [x] **PH-07**: Effective strength = signal strength × ant sensitivity
- [x] **PH-08**: Colony responds to pheromone signal combinations

### State Persistence

- [ ] **STATE-01**: Colony state stored in `.aether/COLONY_STATE.json`
- [ ] **STATE-02**: Pheromone signals stored in `.aether/data/pheromones.json`
- [ ] **STATE-03**: Worker Ant states stored in `.aether/data/worker_ants.json`
- [ ] **STATE-04**: Memory stored in `.aether/data/memory.json`
- [ ] **STATE-05**: State persists across context refreshes
- [ ] **STATE-06**: File locking prevents race condition corruption
- [ ] **STATE-07**: Atomic writes prevent partial state corruption

### Worker Ant Castes

- [ ] **CASTE-01**: Colonizer Ant colonizes codebase and builds semantic index
- [ ] **CASTE-02**: Route-setter Ant creates phase structures and task breakdown
- [ ] **CASTE-03**: Builder Ant implements code and runs commands
- [ ] **CASTE-04**: Watcher Ant validates implementation and tests
- [ ] **CASTE-05**: Scout Ant gathers information and searches docs
- [ ] **CASTE-06**: Architect Ant compresses memory and extracts patterns
- [ ] **CASTE-07**: Each caste can spawn specialists based on capability gaps

### Phase Execution

- [ ] **PHASE-01**: Colony operates in phases with boundaries
- [ ] **PHASE-02**: Emergence occurs within phases (Queen does not intervene)
- [ ] **PHASE-03**: Phase boundaries trigger Queen check-in
- [ ] **PHASE-04**: Queen can review at phase boundaries via `/ant:phase`
- [ ] **PHASE-05**: Queen can adjust pheromones between phases
- [ ] **PHASE-06**: Next phase adapts based on previous phase learnings

### Autonomous Agent Spawning

- [ ] **SPAWN-01**: Worker Ants detect capability gaps autonomously
- [ ] **SPAWN-02**: System analyzes task requirements vs own capabilities
- [ ] **SPAWN-03**: System determines specialist type needed
- [ ] **SPAWN-04**: System spawns specialist via Task tool
- [ ] **SPAWN-05**: Spawned specialist inherits context from parent
- [ ] **SPAWN-06**: Resource budgets limit total spawns (max 10)
- [ ] **SPAWN-07**: Circuit breaker prevents infinite loops (depth limit 3)
- [ ] **SPAWN-08**: Spawn outcomes tracked for meta-learning

### Triple-Layer Memory

- [ ] **MEM-01**: Working Memory stores 200k tokens for current session
- [ ] **MEM-02**: Working Memory stores items with metadata and timestamps
- [ ] **MEM-03**: Short-term Memory stores 10 compressed sessions
- [ ] **MEM-04**: Short-term Memory uses DAST compression (2.5x ratio)
- [ ] **MEM-05**: Long-term Memory stores persistent patterns
- [ ] **MEM-06**: Long-term Memory uses maximum compression
- [ ] **MEM-07**: Associative links connect related items across layers
- [ ] **MEM-08**: Phase boundaries trigger compression (Working → Short-term)
- [ ] **MEM-09**: Pattern extraction triggers storage (Short-term → Long-term)
- [ ] **MEM-10**: LRU eviction when Short-term exceeds 10 sessions
- [ ] **MEM-11**: Search queries all layers and returns ranked results

### Voting-Based Verification

- [ ] **VOTE-01**: System spawns 4 verifier perspectives in parallel
- [ ] **VOTE-02**: Security-focused verifier validates security aspects
- [ ] **VOTE-03**: Performance-focused verifier validates performance aspects
- [ ] **VOTE-04**: Quality-focused validator validates code quality aspects
- [ ] **VOTE-05**: Test-coverage verifier validates test completeness
- [ ] **VOTE-06**: Each verifier casts weighted vote (APPROVE/REJECT)
- [ ] **VOTE-07**: Weight based on historical reliability (belief calibration)
- [ ] **VOTE-08**: Supermajority (67%) required for approval
- [ ] **VOTE-09**: System aggregates issues from all verifiers
- [ ] **VOTE-10**: System records vote for learning and reliability updates

### Meta-Learning Loop

- [ ] **META-01**: System tracks spawn outcomes (success/failure)
- [ ] **META-02**: System updates specialist type confidence based on outcomes
- [ ] **META-03**: System uses Bayesian distribution for confidence scoring
- [ ] **META-04**: System recommends specialists based on historical success
- [ ] **META-05**: System adapts recommendations over time
- [ ] **META-06**: Beta distribution prevents overconfidence from small samples

### State Machine Orchestration

- [ ] **SM-01**: Colony has explicit states (IDLE, INIT, PLANNING, EXECUTING, VERIFYING, COMPLETED, FAILED)
- [ ] **SM-02**: State transitions triggered by events
- [ ] **SM-03**: Checkpoint saved before each state transition
- [ ] **SM-04**: Checkpoint saved after each state transition
- [ ] **SM-05**: System can recover from checkpoint on failure
- [ ] **SM-06**: State history tracked for debugging
- [ ] **SM-07**: Observable state transitions for monitoring

### Event-Driven Communication

- [ ] **EVENT-01**: Event bus enables pub/sub communication
- [ ] **EVENT-02**: Worker Ants can publish events
- [ ] **EVENT-03**: Worker Ants can subscribe to event topics
- [ ] **EVENT-04**: Event filtering prevents irrelevant messages
- [ ] **EVENT-05**: Event logging enables debugging and replay
- [ ] **EVENT-06**: Async non-blocking event delivery
- [ ] **EVENT-07**: Event metrics track performance

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
| CMD-01, CMD-02 | Phase 1 | Pending |
| CMD-03, CMD-04 | Phase 2 | Pending |
| CMD-05, CMD-06, CMD-07 | Phase 3 | Pending |
| PH-01, PH-02 | Phase 3 | Pending |
| PH-03, PH-04, PH-05 | Phase 3 | Pending |
| PH-06, PH-07, PH-08 | Phase 3 | Pending |
| STATE-01, STATE-02, STATE-03, STATE-04 | Phase 1 | Pending |
| STATE-05, STATE-06, STATE-07 | Phase 1 | Pending |
| CASTE-01, CASTE-02, CASTE-03, CASTE-04 | Phase 2 | Pending |
| CASTE-05, CASTE-06, CASTE-07 | Phase 2 | Pending |
| PHASE-01, PHASE-02, PHASE-03, PHASE-04 | Phase 5 | Pending |
| PHASE-05, PHASE-06 | Phase 5 | Pending |
| SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04 | Phase 6 | Pending |
| SPAWN-05, SPAWN-06, SPAWN-07, SPAWN-08 | Phase 6 | Pending |
| MEM-01, MEM-02, MEM-03, MEM-04, MEM-05 | Phase 4 | Pending |
| MEM-06, MEM-07, MEM-08, MEM-09, MEM-10, MEM-11 | Phase 4 | Pending |
| VOTE-01, VOTE-02, VOTE-03, VOTE-04, VOTE-05 | Phase 7 | Pending |
| VOTE-06, VOTE-07, VOTE-08, VOTE-09, VOTE-10 | Phase 7 | Pending |
| META-01, META-02, META-03, META-04, META-05, META-06 | Phase 8 | Pending |
| SM-01, SM-02, SM-03, SM-04, SM-05, SM-06, SM-07 | Phase 5 | Pending |
| EVENT-01, EVENT-02, EVENT-03, EVENT-04, EVENT-05 | Phase 9 | Pending |
| EVENT-06, EVENT-07 | Phase 9 | Pending |

**Coverage:**
- v1 requirements: 52 total
- Mapped to phases: 52/52 ✓
- Unmapped: 0

---
*Requirements defined: 2026-02-01*
*Last updated: 2026-02-01 after roadmap creation*
