# Queen Ant Colony Architecture

**Phased Autonomy with User as Pheromone Source**

---

## Core Principles

1. **Queen provides intention, not commands**
2. **Colony self-organizes within phases**
3. **Pheromones = user signals that guide behavior**
4. **Phase boundaries = checkpoints with Queen**
5. **Pure emergence within structured phases**

---

## The Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        QUEEN (User)                         │
│  Provides intention, pheromones, feedback, observation      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Signals (not commands)
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   /ant:init  │  │ /ant:focus   │  │/ant:redirect │
│   Lay egg    │  │  Attract     │  │   Repel      │
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                         ▼
        ┌─────────────────────────────────────────────┐
        │          PHEROMONE SIGNAL LAYER             │
        │  Signal strength, decay, propagation        │
        └────────────────┬────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    WORKER ANT COLONY                        │
│  Self-organizing castes that respond to pheromones         │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ COLONIZER    │  │ROUTE-SETTER │  │   BUILDER    │
│  Colonize    │  │  Structure   │  │  Build       │
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                │
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    SCOUT     │  │  ARCHITECT   │  │   WATCHER    │
│  Scouting    │  │  Memory      │  │   Quality    │
└──────────────┘  └──────────────┘  └──────────────┘
                         │
                         │ Spawn subagents
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      SUBAGENTS                              │
│  Short-lived, task-specific, emerge as needed              │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ Peer-to-peer coordination
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              MEMORY & LEARNING LAYER                        │
│  Working Memory | Short-Term | Long-Term | Associations    │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ Learning flows back
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    ERROR PREVENTION                         │
│  Track mistakes, auto-flag, create constraints             │
└─────────────────────────────────────────────────────────────┘
```

---

## Queen's Role (User Interface)

The Queen is NOT a commander. The Queen is a **signal source**.

### What the Queen Does

| Action | Command | Meaning |
|--------|---------|---------|
| Initiate | `/ant:init "Build X"` | Lays egg. Creates intention pheromone. |
| Prioritize | `/ant:focus "authentication"` | Attract pheromone. Guides colony attention. |
| Redirect | `/ant:redirect "approach Y"` | Repel pheromone. Warns colony away. |
| Feedback | `/ant:feedback "too slow"` | Guidance pheromone. Adjusts behavior. |
| Observe | `/ant:status`, `/ant:phase` | Watch colony. No signal. |
| Review | `/ant:memory`, `/ant:errors` | Check shared state. |

### What the Queen Does NOT Do

- ❌ "Builder, write this function" (direct command)
- ❌ "Route-setter, create a plan" (assignment)
- ❌ "Watcher, test this" (task assignment)
- ❌ "Stop working on X" (direct intervention)

### The Queen's Workflow

```
1. /ant:init "Build a real-time chat app"
   → Intention pheromone released

2. Colony detects pheromone
   → Colonizer explores codebase
   → Route-setter creates phase structure

3. /ant:phase
   → Queen reviews phase plan
   → /ant:focus "prioritize WebSocket security"
   → Guidance pheromone added

4. Colony executes (pure emergence)
   → Worker Ants self-organize
   → Subagents spawn as needed
   → Respond to focus pheromone

5. Phase boundary
   → Colony checks in
   → /ant:phase shows results
   → Queen reviews, adjusts if needed

6. Next phase
   → Adapts based on feedback
   → Cycle continues
```

---

## Worker Ant Castes

Six pre-defined castes with specific roles and capabilities.

### 1. Colonizer Ant

**Purpose**: Colonize, index, understand codebase

**Capabilities**:
- Semantic codebase exploration
- Dependency graph mapping
- Code relationship identification
- Pattern detection

**Spawns**:
- Graph builders
- Search agents
- Pattern matchers

**Responds to**:
- `/ant:init` (new project → start colonizing)
- `/ant:focus "module X"` (colonize specific area)

**Outputs**:
- Semantic index
- Dependency graphs
- Code relationship maps

---

### 2. Route-setter Ant

**Purpose**: Create structured phase plans

**Capabilities**:
- Goal decomposition
- Phase boundary identification
- Milestone definition
- Dependency analysis

**Spawns**:
- Estimator agents
- Dependency analyzers
- Risk assessors

**Responds to**:
- `/ant:init` (needs structure)
- `/ant:feedback "needs more phases"` (adjust granularity)

**Outputs**:
- Phase plans
- Task breakdowns
- Milestones

---

### 3. Builder Ant

**Purpose**: Write code, implement changes

**Capabilities**:
- Code generation
- File manipulation
- Refactoring
- Implementation

**Spawns**:
- Language specialists (Python, JS, etc.)
- Framework specialists (React, FastAPI, etc.)
- Database specialists

**Responds to**:
- `/ant:focus "feature X"` (prioritize this work)
- `/ant:redirect "don't use Y"` (avoid pattern)

**Outputs**:
- Code files
- Implementations
- Refactored code

---

### 4. Watcher Ant

**Purpose**: Watch, validate, QA

**Capabilities**:
- Test generation
- Validation
- Quality checks
- Bug detection

**Spawns**:
- Test generators
- Lint agents
- Security scanners
- Performance testers

**Responds to**:
- `/ant:focus "quality"` (increase scrutiny)
- `/ant:feedback "too many bugs"` (intensify)

**Outputs**:
- Test suites
- Validation reports
- Bug findings

---

### 5. Scout Ant

**Purpose**: Scout for information, context

**Capabilities**:
- Web search
- Documentation lookup
- Reference finding
- Context gathering

**Spawns**:
- Search agents
- Crawlers
- Documentation readers
- API explorers

**Responds to**:
- `/ant:focus "how to X"` (research specific topic)
- `/ant:init "new domain"` (learn new area)

**Outputs**:
- Research findings
- Documentation summaries
- Context information

---

### 6. Architect Ant

**Purpose**: Architect memory, extract patterns

**Capabilities**:
- Memory compression
- Pattern extraction
- Anti-pattern detection
- Knowledge synthesis

**Spawns**:
- Analysis agents
- Pattern matchers
- Compression agents

**Responds to**:
- Memory capacity alerts (>60%)
- `/ant:phase` boundaries (compress completed phase)

**Outputs**:
- Compressed memory
- Pattern libraries
- Best practices
- Anti-patterns

---

## Pheromone Signal System

Pheromones are **user signals that guide colony behavior** without commands.

### Signal Types

| Signal | Command | Effect | Duration |
|--------|---------|--------|----------|
| **Init** | `/ant:init "goal"` | Strong attract. Triggers planning. | Persists until phase complete |
| **Focus** | `/ant:focus "area"` | Medium attract. Guides attention. | Decays over 1 hour |
| **Redirect** | `/ant:redirect "pattern"` | Strong repel. Warns away. | Decays over 24 hours |
| **Feedback** | `/ant:feedback "msg"` | Variable. Adjusts behavior. | Decays over 6 hours |

### Signal Strength

```
Strength 1.0 (Init)      → Colony mobilizes
Strength 0.7 (Redirect)  → Colony avoids
Strength 0.5 (Focus)     → Colony prioritizes
Strength 0.3 (Feedback)  → Colony adjusts
Strength 0.0 (None)      → Colony operates normally
```

### Signal Decay

Pheromones decay over time. Recent signals are stronger.

```
Strength(t) = InitialStrength × e^(-t/HalfLife)

Example:
/ant:focus "authentication" at t=0
  → t=0:  Strength 0.5
  → t=30m: Strength 0.25
  → t=1h:  Strength 0.125
  → t=2h: Strength ~0 (gone)
```

### Signal Response

Each Worker Ant has a **sensitivity profile**:

```python
Mapper.sensitivity = {
    "init": 1.0,      # Always responds to init
    "focus": 0.7,     # Responds to focus on areas
    "redirect": 0.3,  # Less affected by redirect
}

Planner.sensitivity = {
    "init": 1.0,      # Triggers planning
    "focus": 0.5,     # Adjusts priorities
    "redirect": 0.8,  # Avoids redirected approaches
}

Executor.sensitivity = {
    "init": 0.5,      # Awaits planning
    "focus": 0.9,     # Highly responsive to focus
    "redirect": 0.9,  # Strongly avoids redirected patterns
}
```

### Signal Propagation

Signals propagate through the colony:

```
Queen: /ant:focus "WebSocket security"
  ↓
Pheromone layer: Signal registered
  ↓
Executor Ant: Detects signal
  → Spawns WebSocket-Specialist subagent
  → Prioritizes security-focused implementation
  ↓
Verifier Ant: Detects signal
  → Increases testing intensity on WebSocket code
  → Spawns security-tester subagent
  ↓
Researcher Ant: Detects signal
  → Searches for WebSocket security best practices
  → Feeds findings to Executor
```

---

## Phased Autonomy Flow

The colony operates in **phases** with **autonomy within**.

### Phase Lifecycle

```
1. PHASE INITIATION
   Queen: /ant:init "Build a real-time chat app"
   ↓
   Colony detects init pheromone (strength 1.0)
   ↓
   Mapper: Explores codebase
   Planner: Creates phase structure
   ↓
   Queen reviews: /ant:phase

2. PHASE EXECUTION (Pure Emergence)
   Worker Ants self-organize
   ↓
   Spawn subagents as needed
   ↓
   Respond to pheromones in real-time
   ↓
   Coordinate peer-to-peer
   ↓
   No Queen intervention

3. PHASE BOUNDARY
   Colony: "Phase 1 complete"
   ↓
   Synthesizer: Compresses phase memory
   ↓
   Queen reviews: /ant:phase
   ↓
   Queen adjusts: /ant:focus, /ant:redirect, /ant:feedback
   ↓
   Next phase adapts based on feedback

4. NEXT PHASE
   Cycle continues
   ↓
   Each phase learns from previous
```

### Within a Phase (Pure Emergence)

```
Phase 2: Real-time Communication
  ↓
Queen sets initial pheromones:
  /ant:focus "WebSocket performance"
  /ant:focus "message reliability"
  ↓
Executor Ant detects signals
  → Spawns WebSocket-Specialist
  → Spawns Message-Queue-Specialist
  → They coordinate peer-to-peer
  ↓
Verifier Ant detects signals
  → Spawns Load-Tester
  → Spawns Reliability-Tester
  → They test independently
  ↓
Executor and Verifier coordinate
  → "Load tester found bottleneck"
  → "Optimizer spawned"
  ↓
No Queen direction. Pure emergence.
```

---

## Phase Boundary Checkpoints

At phase boundaries, the colony **checks in** with the Queen.

### What Happens

1. **Phase completion detected**
   - Synthesizer: All tasks complete
   - Memory: Compressed to key learnings

2. **Check-in**
   ```
   /ant:phase

   Phase 2: Real-time Communication - COMPLETE
   Duration: 47 minutes
   Tasks: 8/8 completed
   Agents spawned: 12

   Key Learnings:
   - WebSocket pooling reduces connections 40%
   - Message queue prevents data loss
   - Redis pub/sub scales better than direct

   Issues Found:
   - 3 bugs (all fixed)
   - 1 performance issue (optimized)

   Queen Action:
   → /ant:feedback "Great work"
   → /ant:focus "Next: user authentication"
   → /ant:phase continue
   ```

3. **Queen review**
   - Approve next phase
   - Adjust direction
   - Add new pheromones

4. **Next phase adapts**
   - Incorporates learnings
   - Responds to new pheromones
   - Continues cycle

---

## Memory & Learning Integration

The colony learns from every phase.

### Memory Flow

```
During Phase:
  → Working Memory: Active context, messages, facts
  ↓
Phase Boundary:
  → Synthesizer compresses
  ↓
Short-Term Memory:
  → 10 compressed sessions
  ↓
Long-Term Memory:
  → Persistent patterns, best practices, anti-patterns
```

### Learning from Pheromones

User feedback patterns become learned constraints:

```
/ant:redirect "Don't use string concatenation for SQL"
  → Logged in ERROR_LEDGER
  ↓
After 3 occurrences:
  → FLAGGED_ISSUES created
  ↓
CONSTRAINT created:
  → validate_sql_before_execution
  ↓
Executor learns:
  → Always use parameterized queries
  → Validates before execution
```

### Associative Links

Memory items form associations:

```
"WebSocket security" ←→ "authentication token validation" (0.92)
"message reliability" ←→ "Redis pub/sub" (0.87)
"real-time performance" ←→ "connection pooling" (0.81)
```

Strong associations activate together when either is accessed.

---

## Error Prevention Integration

Errors are tracked, flagged, and prevented.

### Error Lifecycle

```
1. Error occurs
   → Executor: "SQL injection vulnerability"
   ↓
2. Logged
   → ERROR_LEDGER.md updated
   ↓
3. Root cause analyzed
   → "String concatenation in query"
   ↓
4. Prevention created
   → "Use parameterized queries"
   ↓
5. Constraint created (after 3 occurrences)
   → validate_sql_before_execution
   ↓
6. Validates before action
   → Blocks unsafe operations
```

### Error Prevention in Colony

```
Executor attempts:
  → query = f"SELECT * FROM users WHERE id={user_id}"

Constraint validates:
  → String concatenation detected
  → BLOCKED

Executor learns:
  → Use parameterized: "SELECT * FROM users WHERE id=?"
  → Passes validation
```

---

## Coordination Patterns

Worker Ants coordinate using **six patterns** from Phase 6 research.

### 1. Sequential Pattern

```
Planner → creates plan → Executor → implements → Verifier → tests
```

### 2. Parallel Pattern

```
Executor-A ─┐
            ├→ Implement different modules simultaneously
Executor-B ─┘
```

### 3. Router Pattern

```
Router Ant receives task
  → Detects: "Database query needed"
  → Routes to: Database-Specialist
```

### 4. Supervisor Pattern

```
Planner supervises phase execution
  → Monitors progress
  → Adjusts allocations
```

### 5. Swarm Pattern

```
Multiple Executor Ants
  → Detect large task
  → Self-organize
  → Divide and conquer
```

### 6. Group Chat Pattern

```
All Worker Ants
  → Share context
  → Discuss approach
  → Vote on decision
```

---

## Research Alignment

This architecture is grounded in the research corpus.

### Phase 1: Context Engine
- Semantic protocols for communication
- Memory hierarchy (working, short-term, long-term)
- Peer-to-peer coordination reduces overhead

### Phase 3: Semantic Codebase Understanding
- Mapper Ant uses semantic indexing
- Code relationship detection
- Repository-scale understanding

### Phase 4: Predictive & Anticipatory Systems
- Anticipatory next-action prediction
- Adaptive personalization from feedback
- Proactive assistance patterns

### Phase 5: Advanced Verification & Quality
- Verifier Ant generates tests automatically
- Multi-perspective verification
- Feedback loops for learning

### Phase 6: Integration & Synthesis
- Six coordination patterns
- Multi-agent integration protocols
- Component synthesis

### Phase 7: Implementation Planning
- Phased roadmap (18-24 months)
- Production-ready infrastructure
- Progressive deployment

---

## Key Differences from Original AETHER

| Aspect | Original AETHER | Queen Ant Model |
|--------|-----------------|-----------------|
| User role | Provide goal, wait | Signal provider, observer |
| Structure | Pure emergence | Phased autonomy |
| Visibility | Limited | Phase checkpoints |
| Planning | Autonomous | Queen reviews at boundaries |
| Feedback | After completion | Continuous via pheromones |

---

## Implementation Priority

1. **Core architecture** (this document)
2. **Worker Ant castes** (6 implementations)
3. **Pheromone system** (signal layer)
4. **Phase execution** (autonomy engine)
5. **Commands** (user interface)
6. **Memory integration** (learning system)
7. **Error prevention** (constraint system)

---

**This architecture represents a true hybrid: Your vision through pheromones, colony intelligence through emergence.**
