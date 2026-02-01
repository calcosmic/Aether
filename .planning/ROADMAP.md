# Roadmap: Aether v2 - Claude-Native Queen Ant Colony

## Overview

Aether is a **unique, standalone multi-agent system** built from first principles on ant colony intelligence. Unlike AutoGen, LangGraph, CrewAI, or any other framework, Aether implements true autonomous emergence where Worker Ants spawn Worker Ants without human orchestration. This is a Claude-native architecture where prompt files define Worker Ant behaviors and JSON persists colony state. The system follows pheromone-based philosophy: Queen provides intention (not commands), colony self-organizes with emergence within phases, and stigmergic communication replaces orchestration. Each phase delivers observable capabilities while preventing critical pitfalls: JSON corruption, context rot, infinite spawning loops, and hallucination cascades.

## Aether's Unique Philosophy

**Unlike traditional systems:**
- Traditional: Human → Orchestrator → Agents (predefined)
- **Aether: Queen signals → Colony → Workers spawn Workers → Emergence → Complete**

**Core principles:**
1. **Queen provides intention, not commands** - Pheromone signals guide colony
2. **Structure at boundaries, emergence within** - Phase checkpoints, pure emergence during execution
3. **Stigmergic communication** - Environment (pheromones) as communication medium
4. **Autonomous recruitment** - Workers spawn Workers based on capability gaps
5. **Colony IS the intelligence** - No central brain, distributed computation

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Colony Foundation** - JSON state persistence and pheromone signal layer
- [ ] **Phase 2: Worker Ant Castes** - Six Worker Ant prompt behaviors with Task tool spawning (Colonizer, Route-setter, Builder, Watcher, Scout, Architect)
- [ ] **Phase 3: Pheromone Communication** - Stigmergic signals (INIT, FOCUS, REDIRECT, FEEDBACK) with caste sensitivity
- [ ] **Phase 4: Triple-Layer Memory** - Working → Short-term (DAST 2.5x) → Long-term with associative links
- [x] **Phase 5: Phase Boundaries** - State machine with Queen check-ins and colony checkpoints
- [ ] **Phase 6: Autonomous Emergence** - Capability gap detection with Worker-spawns-Workers and safeguards
- [ ] **Phase 7: Colony Verification** - Multi-perspective verification with weighted voting and belief calibration
- [ ] **Phase 8: Colony Learning** - Meta-learning loop with Bayesian confidence scoring for specialist selection
- [ ] **Phase 9: Stigmergic Events** - Event bus for colony-wide pub/sub communication
- [ ] **Phase 10: Colony Maturity** - End-to-end testing, pattern extraction, production readiness

## Phase Details

### Phase 1: Colony Foundation

**Goal**: Colony state persists safely across context refreshes with corruption-proof JSON storage and pheromone signal system

**Aether Philosophy**: Establish the pheromone layer as the communication medium for colony coordination

**Depends on**: Nothing (first phase)

**Requirements**: CMD-01, CMD-02, STATE-01, STATE-02, STATE-03, STATE-04, STATE-05, STATE-06, STATE-07

**Success Criteria** (observable colony behaviors):
1. Queen can initialize colony with `/ant:init "Build a REST API"` and see COLONY_STATE.json created in `.aether/`
2. Queen can run `/ant:status` and see colony state, active pheromones, and Worker Ant counts
3. Colony state persists across Claude context refreshes (file survives, data intact)
4. Multiple Worker Ants can read/write state simultaneously without corruption (file locking prevents race conditions)
5. Atomic writes prevent partial state corruption (crash during write leaves valid previous state)

**Colony Tasks** (Aether-unique terminology):
- [ ] 01-01: Create colony state schema (COLONY_STATE.json) with goal, status, phases, Worker Ants, pheromones, memory
- [ ] 01-02: Create pheromone signal schema (pheromones.json) with type, strength, created_at, decay_rate
- [ ] 01-03: Create Worker Ant state schema (worker_ants.json) with caste, status, current_task, spawns
- [ ] 01-04: Create memory schema (memory.json) with working, short-term, long-term memory
- [ ] 01-05: Implement file locking mechanism for concurrent colony access prevention
- [ ] 01-06: Implement atomic write pattern (temp file + rename) for colony state corruption safety
- [ ] 01-07: Create `/ant:init` command prompt (Queen sets intention, colony mobilizes)
- [ ] 01-08: Create `/ant:status` command prompt (Queen observes colony state)

### Phase 2: Worker Ant Castes

**Goal**: Six Worker Ant castes exist as prompt files and can spawn via Task tool with inherited context

**Aether Philosophy**: Worker Ants are autonomous specialists that respond to pheromone signals and can spawn specialists

**Depends on**: Phase 1 (colony state infrastructure)

**Requirements**: CASTE-01, CASTE-02, CASTE-03, CASTE-04, CASTE-05, CASTE-06, CASTE-07, CMD-03, CMD-04

**Success Criteria** (observable colony behaviors):
1. Each Worker Ant caste (Colonizer, Planner, Executor, Verifier, Researcher, Synthesizer) has a prompt file defining its behavior
2. Route-setter Ant can spawn Builder Ant via Task tool with pheromone-inherited context
3. Spawned specialists receive parent's context (goal, pheromones, relevant colony memory)
4. Queen can run `/ant:phase 1` and see phase details including assigned caste
5. Queen can run `/ant:execute 1` and phase begins with appropriate caste mobilized

**Colony Tasks**:
- [ ] 02-01: Create Colonizer Ant prompt (codebase colonization, semantic indexing, pattern detection)
- [ ] 02-02: Create Route-setter Ant prompt (phase structure, task breakdown, dependency analysis)
- [ ] 02-03: Create Builder Ant prompt (code implementation, command execution, file manipulation)
- [ ] 02-04: Create Watcher Ant prompt (validation, testing, quality checks)
- [ ] 02-05: Create Scout Ant prompt (information gathering, documentation search, context retrieval)
- [ ] 02-06: Create Architect Ant prompt (memory compression, pattern extraction, knowledge synthesis)
- [ ] 02-07: Implement Task tool spawning pattern in Worker Ant prompts (context inheritance)
- [ ] 02-08: Create `/ant:phase` command prompt (Queen reviews phase status)
- [  ] 02-09: Create `/ant:execute` command prompt (Queen triggers phase execution)

### Phase 3: Pheromone Communication

**Goal**: Colony coordinates through stigmergic pheromone signals with time-based decay and caste-specific sensitivity

**Aether Philosophy**: Pheromones are the communication medium - environment holds signals, ants respond locally

**Depends on**: Phase 1 (colony state), Phase 2 (Worker Ants respond to signals)

**Requirements**: PH-01, PH-02, PH-03, PH-04, PH-05, PH-06, PH-07, PH-08, CMD-05, CMD-06, CMD-07

**Success Criteria** (observable colony behaviors):
1. Queen can emit FOCUS pheromone with `/ant:focus "authentication module"` and see it in pheromones.json
2. FOCUS pheromone decays over 1 hour (strength drops to 50% after 30 minutes)
3. REDIRECT pheromone emitted via `/ant:redirect "avoid synchronous patterns"` lasts 24 hours then disappears
4. FEEDBACK pheromone via `/ant:feedback "great progress on API"` decays over 6 hours
5. Worker Ants respond to pheromone combinations (e.g., FOCUS + FEEDBACK increases sensitivity)
6. INIT pheromone persists until phase complete (no decay)
7. Effective strength = signal strength × ant sensitivity (different castes have different sensitivities)

**Colony Tasks**:
- [x] 03-01: Create `/ant:focus` command prompt (FOCUS pheromone, 1-hour decay)
- [x] 03-02: Create `/ant:redirect` command prompt (REDIRECT pheromone, 24-hour decay)
- [x] 03-03: Create `/ant:feedback` command prompt (FEEDBACK pheromone, 6-hour decay)
- [x] 03-04: Update Colonizer, Route-setter, Builder Ant prompts (pheromone reading)
- [x] 03-05: Update Watcher, Scout, Architect Ant prompts (pheromone reading)
- [x] 03-06: Verification and human checkpoint (pheromone communication system verified)

### Phase 4: Triple-Layer Memory

**Goal**: Colony memory compresses across three layers (Working → Short-term → Long-term) preventing context rot and enabling retrieval

**Aether Philosophy**: Memory mirrors human cognition - working (immediate), short-term (recent sessions), long-term (persistent patterns)

**Depends on**: Phase 1 (colony state), Phase 3 (pheromone signals trigger compression at boundaries)

**Requirements**: MEM-01, MEM-02, MEM-03, MEM-04, MEM-05, MEM-06, MEM-07, MEM-08, MEM-09, MEM-10, MEM-11

**Success Criteria** (observable colony behaviors):
1. Working Memory stores current session items with metadata (type, timestamp, relevance_score)
2. When Working Memory exceeds 150k tokens, oldest items are evicted (LRU policy)
3. At phase boundary, Working Memory compresses to Short-term (2.5x ratio) via DAST algorithm
4. Short-term Memory stores maximum 10 compressed sessions (11th triggers LRU eviction)
5. Pattern extraction moves high-value items from Short-term to Long-term (associative links)
6. Queen can query memory and get ranked results from all three layers
7. Context window never exceeds 200k tokens (compression triggers prevent overflow)

**Plans:** 5 plans in 5 waves

**Colony Tasks**:
- [ ] 04-01-PLAN.md — Working Memory operations (add, get, update, list) with LRU eviction at 80% capacity
- [ ] 04-02-PLAN.md — DAST compression prompt enhancement and Short-term Memory schema verification
- [ ] 04-03-PLAN.md — Short-term LRU eviction (max 10 sessions) and Long-term pattern extraction
- [ ] 04-04-PLAN.md — Phase boundary compression trigger and pattern extraction trigger
- [ ] 04-05-PLAN.md — Cross-layer search with relevance ranking and /ant:memory command

### Phase 5: Phase Boundaries

**Goal**: Colony operates through explicit state machine with phase boundaries, checkpoints, and recovery capability

**Aether Philosophy**: Phase boundaries are Queen check-ins - structure at boundaries, pure emergence within

**Depends on**: Phase 2 (Worker Ants mobilize), Phase 3 (pheromone signals), Phase 4 (memory triggers at boundaries)

**Requirements**: SM-01, SM-02, SM-03, SM-04, SM-05, SM-06, SM-07, PHASE-01, PHASE-02, PHASE-03, PHASE-04, PHASE-05, PHASE-06

**Success Criteria** (observable colony behaviors):
1. Colony has explicit states (IDLE, INIT, PLANNING, EXECUTING, VERIFYING, COMPLETED, FAILED)
2. State transitions triggered by pheromone signals (e.g., phase complete → VERIFYING)
3. Checkpoint saved before each state transition (colony can recover to previous state)
4. Checkpoint saved after each state transition (rollback capability)
5. Colony can recover from checkpoint after crash (restart from last known good state)
6. State history tracked in COLONY_STATE.json (debugging capability)
7. At phase boundaries, Queen check-in occurs (Queen can review via `/ant:phase`)
8. Next phase adapts based on previous phase learnings (colony memory influences planning)
9. Emergence occurs within phases (Worker Ants work autonomously, Queen doesn't intervene)

**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)
**Plans:** 9 plans in 4 waves

**Colony Tasks:**
- [ ] 05-01-PLAN.md — State machine schema and transition validation
- [ ] 05-02-PLAN.md — Pheromone-triggered state transitions with file locking
- [ ] 05-03-PLAN.md — Checkpoint system with save/load/rotate functions
- [ ] 05-04-PLAN.md — Pre/post-transition checkpoint integration and recovery
- [ ] 05-05-PLAN.md — Crash detection and /ant:recover command
- [ ] 05-06-PLAN.md — State history logging with archival to memory
- [ ] 05-07-PLAN.md — Phase boundary Queen check-in with CHECKIN pheromone
- [ ] 05-08-PLAN.md — Next phase adaptation from previous phase memory
- [ ] 05-09-PLAN.md — Emergence guard (no Queen intervention during EXECUTING)

### Phase 6: Autonomous Emergence

**Goal**: Worker Ants detect capability gaps and spawn specialists automatically with safeguards against infinite loops

**Aether Philosophy**: Workers spawn Workers autonomously - no Queen approval needed, colony self-organizes

**Depends on**: Phase 2 (Worker Ants exist), Phase 4 (memory for spawn tracking), Phase 5 (orchestration limits scope)

**Requirements**: SPAWN-01, SPAWN-02, SPAWN-03, SPAWN-04, SPAWN-05, SPAWN-06, SPAWN-07, SPAWN-08

**Success Criteria** (observable colony behaviors):
1. Worker Ant detects capability gap (e.g., Route-setter Ant needs database expertise)
2. System analyzes task requirements vs own capabilities (gap detection logic)
3. System determines specialist type needed (maps gap to caste)
4. System spawns specialist via Task tool (autonomous spawning, no Queen approval)
5. Spawned specialist inherits parent context (goal, pheromones, colony memory)
6. Resource budget limits total spawns (max 10 per phase, tracked in state)
7. Circuit breaker prevents infinite loops (depth limit 3, same-specialist cache)
8. Spawn outcomes tracked for meta-learning (success/failure recorded)
**Colony Tasks**:
- [ ] 06-01-PLAN.md — Capability gap detection and specialist mapping (Wave 1)
- [ ] 06-02-PLAN.md — Task tool spawning with context inheritance and budget tracking (Wave 2)
- [ ] 06-03-PLAN.md — Spawn depth limit and circuit breaker (Wave 3)
- [ ] 06-04-PLAN.md — Spawn outcome tracking and safeguard testing (Wave 4)

**Plans:** 4 plans in 4 waves

**Wave Structure:**
- Wave 1: Capability detection + Specialist mapping (06-01)
- Wave 2: Spawning infrastructure + Budget tracking (06-02)
- Wave 3: Depth limit + Circuit breaker (06-03)
- Wave 4: Meta-learning + Testing (06-04)

### Phase 7: Colony Verification

**Goal**: Multiple verifier perspectives validate outputs with weighted voting and belief calibration for improved accuracy

**Aether Philosophy**: Colony learns from verification - successful votes increase reliability weights, improving future decisions

**Depends on**: Phase 2 (Watcher caste), Phase 6 (Worker Ants spawn watchers in parallel)

**Requirements**: VOTE-01, VOTE-02, VOTE-03, VOTE-04, VOTE-05, VOTE-06, VOTE-07, VOTE-08, VOTE-09, VOTE-10

**Success Criteria** (observable colony behaviors):
1. Colony spawns 4 watcher perspectives in parallel (Security, Performance, Quality, Test-coverage)
2. Security-focused watcher checks for vulnerabilities, auth issues, input validation
3. Performance-focused watcher checks for complexity, bottlenecks, resource usage
4. Quality-focused watcher checks for maintainability, readability, conventions
5. Test-coverage watcher checks for test completeness, edge cases, assertions
6. Each watcher casts weighted vote (APPROVE/REJECT) based on historical reliability
7. Weight based on belief calibration (reliable watchers have higher weight)
8. Supermajority (67%) required for approval (3/4 or 4/4 must approve)
9. Colony aggregates issues from all watchers (unified issue report)
10. Colony records vote for meta-learning (update reliability scores)

**Colony Tasks**:
- [ ] 07-01: Create Security Watcher prompt (vulnerabilities, auth, input validation)
- [ ] 07-02: Create Performance Watcher prompt (complexity, bottlenecks, resources)
- [ ] 07-03: Create Quality Watcher prompt (maintainability, readability, conventions)
- [ ] 07-04: Create Test-Coverage Watcher prompt (completeness, edge cases, assertions)
- [ ] 07-05: Implement parallel watcher spawning via Task tool
- [ ] 07-06: Implement weighted voting schema (belief calibration scores)
- [ ] 07-07: Implement supermajority calculation (67% threshold)
- [ ] 07-08: Implement issue aggregation (combine issues from all watchers)
- [ ] 07-09: Implement vote recording (track for meta-learning)
- [ ] 07-10: Test voting system (verify supermajority logic, check weight updates)

### Phase 8: Colony Learning

**Goal**: Colony learns which specialists work best for which tasks using Bayesian confidence scoring

**Aether Philosophy**: Colony intelligence emerges from learning - successful spawns increase confidence, failed spawns decrease it

**Depends on**: Phase 6 (spawn outcomes tracked), Phase 7 (vote outcomes recorded)

**Requirements**: META-01, META-02, META-03, META-04, META-05, META-06

**Success Criteria** (observable colony behaviors):
1. Colony tracks spawn outcomes (success/failure) in meta_learning.json
2. Colony updates specialist type confidence based on outcomes (Beta distribution)
3. Colony uses Bayesian distribution for confidence scoring (prevents overconfidence)
4. Colony recommends specialists based on historical success (highest confidence for task type)
5. Colony adapts recommendations over time (continuous learning as outcomes accumulate)
6. Beta distribution prevents overconfidence from small samples (alpha=1, beta=1 prior)

**Colony Tasks**:
- [ ] 08-01: Implement meta_learning.json schema (specialist types, success/failure counts, confidence scores)
- [ ] 08-02: Implement spawn outcome tracking (record success/failure after each spawn)
- [ ] 08-03: Implement Beta distribution confidence scoring (alpha = successes + 1, beta = failures + 1)
- [ ] 08-04: Implement specialist recommendation logic (select highest confidence for task type)
- [ ] 08-05: Implement confidence score updates (increment alpha/beta based on outcomes)
- [ ] 08-06: Test meta-learning loop (verify confidence updates, check recommendation accuracy)

### Phase 9: Stigmergic Events

**Goal**: Pub/sub event bus enables colony-wide asynchronous coordination between Worker Ants

**Aether Philosophy**: Event bus extends stigmergic communication - pheromones for guidance, events for coordination

**Depends on**: Phase 3 (pheromone signals), Phase 5 (orchestration engine)

**Requirements**: EVENT-01, EVENT-02, EVENT-03, EVENT-04, EVENT-05, EVENT-06, EVENT-07

**Success Criteria** (observable colony behaviors):
1. Event bus enables pub/sub communication (Worker Ants can publish/subscribe)
2. Worker Ants can publish events (task started, completed, failed, discovered issue)
3. Worker Ants can subscribe to event topics (phase_complete, error, spawn_request)
4. Event filtering prevents irrelevant messages (Worker Ants only receive relevant events)
5. Event logging enables debugging and replay (events logged to events.json)
6. Async non-blocking event delivery (publish returns immediately, no waiting)
7. Event metrics track performance (publish rate, subscribe count, delivery latency)

**Colony Tasks**:
- [ ] 09-01: Implement event bus schema in events.json (topics, subscriptions, logs)
- [ ] 09-02: Implement publish operation (Worker Ants can emit events to topics)
- [ ] 09-03: Implement subscribe operation (Worker Ants can register interest in topics)
- [ ] 09-04: Implement event filtering (Worker Ants only receive matching events)
- [ ] 09-05: Implement event logging (all events logged with timestamps)
- [ ] 09-06: Implement async event delivery (non-blocking publish)
- [ ] 09-07: Implement event metrics (track publish rate, subscriptions, latency)

### Phase 10: Colony Maturity

**Goal**: End-to-end colony validation with comprehensive testing and production readiness

**Aether Philosophy**: Colony maturity means emergence works reliably - Queen provides intention, colony self-organizes, results emerge

**Depends on**: All previous phases (integration requires all colony components)

**Requirements**: (All v1 requirements validated end-to-end)

**Success Criteria** (observable colony behaviors):
1. Queen can run full workflow: `/ant:init` → phases execute → colony completes goal
2. Autonomous spawning works (Worker Ants spawn specialists without Queen intervention)
3. Memory compression works (no context rot after extended session)
4. Voting verification works (outputs validated by multiple perspectives)
5. Meta-learning improves recommendations (confidence scores adjust over time)
6. Event-driven communication scales (multiple Worker Ants coordinate without bottleneck)
7. State corruption prevented (file locking + atomic writes survive concurrent access)
8. Infinite loops prevented (circuit breakers trigger, depth limits enforced)
9. All critical pitfalls from research addressed (no regressions)

**Colony Tasks**:
- [ ] 10-01: End-to-end integration test (run full workflow from init to completion)
- [ ] 10-02: Test autonomous spawning (verify capability gap detection and specialist selection)
- [ ] 10-03: Test memory compression (verify no context rot after extended session)
- [ ] 10-04: Test voting verification (verify supermajority logic and issue aggregation)
- [ ] 10-05: Test meta-learning (verify confidence updates and recommendation improvements)
- [ ] 10-06: Test event-driven communication (verify pub/sub scalability and filtering)
- [ ] 10-07: Test state corruption prevention (verify file locking and atomic writes under load)
- [ ] 10-08: Test infinite loop prevention (verify circuit breakers and depth limits)
- [ ] 10-09: Create colony documentation (README with quickstart and examples)
- [ ] 10-10: Performance optimization (identify and address bottlenecks)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10

| Phase | Plans | Status | Completed |
|-------|-------|--------|-----------|
| 1. Colony Foundation | 8/8 | Complete | 2026-02-01 |
| 2. Worker Ant Castes | 9/9 | Complete | 2026-02-01 |
| 3. Pheromone Communication | 6/6 | Complete | 2026-02-01 |
| 4. Triple-Layer Memory | 5/5 | Complete | 2026-02-01 |
| 5. Phase Boundaries | 9/9 | Complete | 2026-02-01 |
| 6. Autonomous Emergence | 0/8 | Not started | - |
| 7. Colony Verification | 0/10 | Not started | - |
| 8. Colony Learning | 0/6 | Not started | - |
| 9. Stigmergic Events | 0/7 | Not started | - |
| 10. Colony Maturity | 0/10 | Not started | - |

## Aether's Unique Approach

This roadmap is **NOT** a generic software development plan. It follows Aether's pheromone-based philosophy:

**Traditional vs Aether:**

| Aspect | Traditional Software | Aether Colony |
|--------|-------------------|---------------|
| **Control** | Human orchestrator, predefined agents | Queen signals, colony self-organizes |
| **Communication** | Direct commands, message passing | Pheromone signals, stigmergic coordination |
| **Planning** | Human-defined workflows | Queen sets intention, colony creates structure |
| **Execution** | Sequential task lists | Emergent execution within phases |
| **Verification** | Single verifier or tests | Multi-perspective voting with belief calibration |
| **Learning** | Manual analysis | Meta-learning loop with Bayesian confidence |

**Aether-Unique Terminology:**

- **Queen**: User who provides intention via pheromones (not "user", not "orchestrator")
- **Colony**: Multi-agent system that self-organizes (not "system", not "framework")
- **Worker Ants**: Autonomous specialists (Colonizer, Route-setter, Builder, Watcher, Scout, Architect)
- **Pheromones**: Signals that guide colony behavior (INIT, FOCUS, REDIRECT, FEEDBACK)
- **Emergence**: Colony self-organization without central direction
- **Phase Boundaries**: Queen check-ins where structure meets emergence
- **Stigmergy**: Communication through environment (pheromone layer)

This roadmap builds a **completely standalone system** - Aether v2 is not dependent on CDS, Ralph, or any external framework. It's its own unique multi-agent system based on ant colony intelligence principles, with:

- **Unique Worker Ant architecture** - Caste system designed from first principles for autonomous emergence
- **Unique pheromone communication** - Stigmergic signaling system unlike any other framework
- **Unique phase structure** - Phased autonomy with boundaries, not copied from any other system
- **Claude-native implementation** - Built as prompt commands from the ground up, not ported from Python

Aether draws inspiration from research on ant colonies, multi-agent systems, and stigmergic communication, but all architectures, patterns, and implementations are uniquely Aether.

---

*Aether v2: Queen Ant Colony - Autonomous Emergence in Claude-Native Form*
