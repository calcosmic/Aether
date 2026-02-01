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
- [ ] **CMD-05**: User can emit focus pheromone with `/ant:focus <area>`
- [ ] **CMD-06**: User can emit redirect pheromone with `/ant:redirect <pattern>`
- [ ] **CMD-07**: User can emit feedback pheromone with `/ant:feedback <msg>`

### Pheromone Signal System

- [ ] **PH-01**: System emits INIT pheromone when user initializes project
- [ ] **PH-02**: INIT pheromone persists until phase complete
- [ ] **PH-03**: FOCUS pheromone decays over 1 hour
- [ ] **PH-04**: REDIRECT pheromone decays over 24 hours
- [ ] **PH-05**: FEEDBACK pheromone decays over 6 hours
- [ ] **PH-06**: Worker Ants detect pheromones based on sensitivity profile
- [ ] **PH-07**: Effective strength = signal strength × ant sensitivity
- [ ] **PH-08**: Colony responds to pheromone signal combinations

### State Persistence

- [ ] **STATE-01**: Colony state stored in `.aether/COLONY_STATE.json`
- [ ] **STATE-02**: Pheromone signals stored in `.aether/data/pheromones.json`
- [ ] **STATE-03**: Worker Ant states stored in `.aether/data/worker_ants.json`
- [ ] **STATE-04**: Memory stored in `.aether/data/memory.json`
- [ ] **STATE-05**: State persists across context refreshes
- [ ] **STATE-06**: File locking prevents race condition corruption
- [ ] **STATE-07**: Atomic writes prevent partial state corruption

### Worker Ant Castes

- [ ] **CASTE-01**: Mapper Ant explores codebase and builds semantic index
- [ ] **CASTE-02**: Planner Ant creates phase structures and task breakdown
- [ ] **CASTE-03**: Executor Ant implements code and runs commands
- [ ] **CASTE-04**: Verifier Ant validates implementation and tests
- [ ] **CASTE-05**: Researcher Ant gathers information and searches docs
- [ ] **CASTE-06**: Synthesizer Ant compresses memory and extracts patterns
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
- [ ] **VOTE-04**: Quality-focused verifier validates code quality aspects
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
| CMD-01 CMD-07 | Phase 1 | Pending |
| PH-01 PH-08 | Phase 3 | Pending |
| STATE-01 STATE-07 | Phase 1 | Pending |
| CASTE-01 CASTE-07 | Phase 2 | Pending |
| PHASE-01 PHASE-06 | Phase 4 | Pending |
| SPAWN-01 SPAWN-08 | Phase 4 | Pending |
| MEM-01 MEM-11 | Phase 3 | Pending |
| VOTE-01 VOTE-10 | Phase 5 | Pending |
| META-01 META-06 | Phase 6 | Pending |
| SM-01 SM-07 | Phase 4 | Pending |
| EVENT-01 EVENT-07 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 52 total
- Mapped to phases: Pending (roadmap not created yet)
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-01*
*Last updated: 2026-02-01 after initial definition*
