# Aether: Queen Ant Colony - Autonomous Multi-Agent System

## Quick Start

Aether is a phased autonomy system where the Queen (you) provides intention through pheromone signals, and the colony self-organizes to execute work. Unlike traditional multi-agent systems that require task-by-task supervision, Aether enables true emergence within structured phases.

### Installation

Aether is a standalone bash-based system with no external dependencies beyond standard Unix tools:

```bash
# Clone the repository
git clone <repo-url>
cd Aether

# The colony is ready - no npm/pip install required
ls .aether/utils/  # See available utilities
```

### First Colony Initialization

Start your first colony with a goal:

```bash
# Initialize colony with intention
.aether/utils/colony-setup.sh setup_test_colony "Build REST API for task management"

# Check colony status
cat .aether/data/COLONY_STATE.json | jq '.colony_status'

# Review active pheromones (signals guiding the colony)
cat .aether/data/pheromones.json | jq '.active_pheromones'
```

### Basic Commands

```
/ant:init <goal>         # Lay egg (new intention)
/ant:phase               # Show current phase status
/ant:plan                # Show upcoming phases
/ant:focus <area>        # Attract pheromone (prioritize)
/ant:redirect <pattern>  # Repel pheromone (avoid)
/ant:feedback <message>  # Guidance signal
/ant:status              # Colony state
/ant:memory              # Shared pheromone trails
/ant:errors              # Danger signals
```

---

## Architecture

### What Makes Aether Unique

**Traditional Multi-Agent Systems:**
- User provides task → Agent executes → User reviews → User provides next task
- Constant supervision required
- No shared context between agents
- Task-by-task orchestration

**Aether Queen Ant Model:**
- User provides intention → Colony self-organizes → Colony checks in at phase boundaries
- Supervision only at phase boundaries (not task-by-task)
- Shared memory and pheromone signals coordinate agents
- Emergent behavior within structured phases

The key insight: **Queen provides signals, not commands.**

### Core Components

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
```

### Pheromone Signal System

Pheromones are user signals that guide colony behavior without commands:

| Signal | Command | Effect | Duration |
|--------|---------|--------|----------|
| **Init** | `/ant:init "goal"` | Strong attract. Triggers planning. | Persists until phase complete |
| **Focus** | `/ant:focus "area"` | Medium attract. Guides attention. | Decays over 1 hour |
| **Redirect** | `/ant:redirect "pattern"` | Strong repel. Warns away. | Decays over 24 hours |
| **Feedback** | `/ant:feedback "msg"` | Variable. Adjusts behavior. | Decays over 6 hours |

**Signal Decay:**
```
Strength(t) = InitialStrength × e^(-t/HalfLife)

Example: /ant:focus "authentication" at t=0
  → t=0:  Strength 0.5
  → t=30m: Strength 0.25
  → t=1h: Strength 0.125
  → t=2h: Strength ~0 (gone)
```

### Worker Ant Castes

Six pre-defined castes with specific roles and capabilities:

| Caste | Role | Spawns |
|-------|------|--------|
| **Colonizer** | Colonize, index, understand codebase | Graph agents, search agents |
| **Route-setter** | Create structured phase plans | Estimators, risk assessors |
| **Builder** | Build code, implement | Language/framework specialists |
| **Watcher** | Watch, validate, QA | Test generators, security scanners |
| **Scout** | Scout for information, context | Search agents, crawlers |
| **Architect** | Architect memory, extract patterns | Analysis agents |

These castes are always available and spawn subagents as needed.

---

## Command Reference

### `/ant:init <goal>`

Lay egg - creates new intention pheromone that triggers colony mobilization.

```bash
/ant:init "Build a real-time chat application"
```

**What happens:**
1. Init pheromone released (strength 1.0)
2. Colonizer explores codebase
3. Route-setter creates phase structure
4. Colony awaits Queen review at phase boundary

### `/ant:phase`

Show current phase status - Queen's checkpoint view.

```bash
/ant:phase
```

**Output:**
```
Phase 2: Real-time Communication - IN PROGRESS
Duration: 47 minutes
Tasks: 5/8 completed
Agents spawned: 7

Key Learnings:
- WebSocket pooling reduces connections 40%
- Message queue prevents data loss

Queen Action:
→ /ant:feedback "Great work"
→ /ant:focus "Next: user authentication"
→ /ant:phase continue
```

### `/ant:focus <area>`

Attract pheromone - guides colony attention to specific areas.

```bash
/ant:focus "WebSocket security"
/ant:focus "authentication"
/ant:focus "performance optimization"
```

**Effect:** Colony prioritizes work in focused areas. Signal decays over 1 hour.

### `/ant:redirect <pattern>`

Repel pheromone - warns colony away from anti-patterns.

```bash
/ant:redirect "Don't use string concatenation for SQL"
/ant:redirect "Avoid callback hell - use promises"
```

**Effect:** Colony avoids specified patterns. Signal decays over 24 hours.

### `/ant:feedback <message>`

Guidance signal - adjusts colony behavior based on Queen observations.

```bash
/ant:feedback "Too slow - optimize database queries"
/ant:feedback "Great progress on WebSocket handling"
```

**Effect:** Colony adjusts approach. Signal decays over 6 hours.

### `/ant:status`

Show colony state - comprehensive view of all castes and current activity.

```bash
/ant:status
```

### `/ant:memory`

Show shared memory - pheromone trails and accumulated knowledge.

```bash
/ant:memory
```

### `/ant:errors`

Show error prevention - tracked mistakes and auto-flagged issues.

```bash
/ant:errors
```

---

## Caste Behaviors

### Colonizer Ant

**Purpose:** Colonize codebases by building semantic understanding

**Capabilities:**
- Codebase Analysis: Understand project structure, patterns, conventions
- Semantic Indexing: Build mental maps of code relationships
- Pattern Detection: Find recurring patterns and anti-patterns
- Dependency Mapping: Trace component dependencies

**Responds to:**
- `/ant:init` (new project → start colonizing)
- `/ant:focus "module X"` (colonize specific area)

**Outputs:**
- Semantic index
- Dependency graphs
- Code relationship maps

**Spawns:** Graph builders, search agents, pattern matchers

---

### Route-setter Ant

**Purpose:** Create structured phase plans

**Capabilities:**
- Goal decomposition
- Phase boundary identification
- Milestone definition
- Dependency analysis

**Responds to:**
- `/ant:init` (needs structure)
- `/ant:feedback "needs more phases"` (adjust granularity)

**Outputs:**
- Phase plans
- Task breakdowns
- Milestones

**Spawns:** Estimator agents, dependency analyzers, risk assessors

---

### Builder Ant

**Purpose:** Write code, implement changes

**Capabilities:**
- Code generation
- File manipulation
- Refactoring
- Implementation

**Responds to:**
- `/ant:focus "feature X"` (prioritize this work)
- `/ant:redirect "don't use Y"` (avoid pattern)

**Outputs:**
- Code files
- Implementations
- Refactored code

**Spawns:** Language specialists (Python, JS, etc.), framework specialists, database specialists

---

### Watcher Ant

**Purpose:** Watch, validate, QA

**Capabilities:**
- Test generation
- Validation
- Quality checks
- Bug detection

**Responds to:**
- `/ant:focus "quality"` (increase scrutiny)
- `/ant:feedback "too many bugs"` (intensify)

**Outputs:**
- Test suites
- Validation reports
- Bug findings

**Spawns:** Test generators, lint agents, security scanners, performance testers

---

### Scout Ant

**Purpose:** Scout for information, context

**Capabilities:**
- Web search
- Documentation lookup
- Reference finding
- Context gathering

**Responds to:**
- `/ant:focus "how to X"` (research specific topic)
- `/ant:init "new domain"` (learn new area)

**Outputs:**
- Research findings
- Documentation summaries
- Context information

**Spawns:** Search agents, crawlers, documentation readers, API explorers

---

### Architect Ant

**Purpose:** Architect memory, extract patterns

**Capabilities:**
- Memory compression
- Pattern extraction
- Anti-pattern detection
- Knowledge synthesis

**Responds to:**
- Memory capacity alerts (>60%)
- `/ant:phase` boundaries (compress completed phase)

**Outputs:**
- Compressed memory
- Pattern libraries
- Best practices
- Anti-patterns

**Spawns:** Analysis agents, pattern matchers, compression agents

---

## Examples

### Example 1: Basic Workflow

Initialize colony, observe emergence, complete goal.

```bash
# 1. Initialize colony with intention
/ant:init "Build a REST API for task management"

# Colony response: Colonizer explores, Route-setter creates phases
# Output: "5 phases created with 30 total tasks"

# 2. Review phase plan
/ant:phase

# 3. (Optional) Guide colony attention
/ant:focus "database schema design"

# 4. Colony executes (pure emergence)
# - Worker Ants self-organize
# - Subagents spawn as needed
# - No Queen intervention needed

# 5. At phase boundary: Review results
/ant:phase

# 6. Provide feedback (optional)
/ant:feedback "Great work on the API endpoints"

# 7. Continue to next phase
/ant:phase continue
```

**What you see:**
- Phase 1 complete
- 8/8 tasks done
- 12 subagents spawned
- Key learnings extracted

### Example 2: Pheromone Guidance

Use FOCUS to guide colony, REDIRECT to avoid anti-patterns.

```bash
# Initialize
/ant:init "Build real-time chat application"

# Colony starts working...

# Guide attention to specific area
/ant:focus "WebSocket security"

# Colony response: Prioritizes WebSocket security work
# - Builder spawns WebSocket-Specialist
# - Watcher spawns Security-Tester
# - Scout researches WebSocket best practices

# Warn away from anti-pattern
/ant:redirect "Don't use string concatenation for SQL queries"

# Colony response: Avoids pattern, uses parameterized queries
# - Logged in ERROR_LEDGER
# - After 3 occurrences: CONSTRAINT created
# - Future attempts automatically blocked
```

### Example 3: Recovery from Checkpoint

Simulate crash, recover from checkpoint, resume work.

```bash
# Colony is working on Phase 3...
# (Simulate crash: kill process, lose state)

# Recover from checkpoint
/ant:recover

# Output:
# "Restored from checkpoint: phase_3_checkpoint_20260202.json"
# "Phase 3: User Authentication - RESUMED"
# "Tasks: 3/5 completed, resuming from task 4"

# Colony continues from where it left off
# - Working memory restored
# - Pheromone signals restored
# - Worker Ant state restored
```

### Example 4: Memory Query

Search compressed memory, retrieve key information.

```bash
# Query memory for patterns
/ant:memory --query "WebSocket connection pooling"

# Output:
# "Found 3 related memory items:"
#
# 1. [Phase 2] WebSocket pooling reduces connections 40%
#    - Created: 2026-02-01T14:23:00Z
#    - Relevance: 0.92
#    - Summary: Connection pooling dramatically reduces resource...
#
# 2. [Phase 2] Redis pub/sub scales better than direct messages
#    - Created: 2026-02-01T15:10:00Z
#    - Relevance: 0.87
#    - Association: connection_pooling (0.81)
#
# 3. [Long-term] Pattern: Connection pooling essential for scalability
#    - Created: 2026-01-28T10:05:00Z
#    - Relevance: 0.78
#    - Confidence: 0.94 (based on 7 occurrences)
```

---

## Troubleshooting

### Colony Not Responding

**Problem:** Colony seems stuck, no progress.

**Diagnosis:**
```bash
# Check colony state
/ant:status

# Check for circuit breaker trips
cat .aether/data/COLONY_STATE.json | jq '.resource_budgets.circuit_breaker_trips'

# Check pheromone signals
cat .aether/data/pheromones.json | jq '.active_pheromones'
```

**Solutions:**
- If circuit breaker tripped: Wait for cooldown or reset manually
- If no active pheromones: Add `/ant:focus` to guide attention
- If stuck in phase: Try `/ant:phase continue` to advance

### Memory Overflow

**Problem:** Working memory at capacity, compression not triggered.

**Diagnosis:**
```bash
# Check memory usage
cat .aether/data/memory.json | jq '.working_memory'
```

**Solutions:**
```bash
# Manual compression trigger
.aether/utils/memory-compress.sh trigger_phase_boundary_compression 1 '{"summary": "..."}'

# Clear working memory (emergency)
.aether/utils/memory-ops.sh clear_working_memory
```

### Spawning Failures

**Problem:** Subagent spawns failing repeatedly.

**Diagnosis:**
```bash
# Check spawn tracking
cat .aether/data/COLONY_STATE.json | jq '.spawn_tracking'

# Check failed specialist types
cat .aether/data/COLONY_STATE.json | jq '.spawn_tracking.failed_specialist_types'
```

**Solutions:**
- Circuit breaker trips automatically after 3 failures
- Wait for cooldown (10 minutes)
- Check ERROR_LEDGER for root causes

---

## FAQ

### Philosophy

**Q: Why pheromones instead of commands?**

A: Commands require constant supervision. Pheromones enable emergence. The colony self-organizes based on signal strength and decay, just like biological ant colonies. This scales better and requires less human oversight.

**Q: What's the difference between Aether and other multi-agent systems?**

A: Most systems are task-oriented (human assigns each task). Aether is intention-oriented (human sets goal, colony figures out tasks). Also, Aether uses phased autonomy with checkpoints, not pure emergence or pure control.

**Q: Can I use Aether for production work?**

A: See Production Readiness Checklist below.

### Comparison

**Q: How does this compare to AutoGPT?**

A: AutoGPT uses autonomous agents with goal-chasing loops. Aether uses phased autonomy with explicit checkpoints. AutoGPT can run forever; Aether checks in at phase boundaries for course correction.

**Q: How does this compare to CrewAI?**

A: CrewAI uses hierarchical crews with defined roles. Aether uses pheromone signals for coordination. CrewAI requires explicit task assignment; Aether allows agents to self-organize based on signals.

### Usage

**Q: Do I need to monitor the colony constantly?**

A: No. That's the point. Set intention with `/ant:init`, check in at phase boundaries with `/ant:phase`. Use `/ant:focus` or `/ant:redirect` if you want to guide attention between checkpoints.

**Q: What if the colony goes off the rails?**

A: Phase boundaries are safety checkpoints. Review work at each boundary, provide corrective feedback via `/ant:feedback`, and the next phase adapts. Circuit breakers prevent infinite loops.

**Q: Can I run multiple colonies simultaneously?**

A: Yes, each in its own directory. Colonies don't interfere because all state is in `.aether/data/`.

---

## Performance Tuning

### How to Measure Baselines

Run performance baseline test to establish operation timing:

```bash
bash .planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/performance/timing-baseline.test.sh
```

**Output:**
```
ok 1 - Colony init: 0.020s (min: 0.019s, max: 0.021s)
ok 2 - Pheromone emit: 0.012s (min: 0.011s, max: 0.013s)
...
Baseline saved to: baseline-20260202.json
```

### How to Identify Bottlenecks

Generate comparison report to find slow operations:

```bash
source .planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/performance/metrics-tracking.sh
generate_report baseline-20260201.json baseline-20260202.json
```

**Output shows:**
- Delta (improvement or regression)
- Percent change
- Color-coded: ✓ for improvement, ✗ for regression
- Bottlenecks (slowest 3 operations)

### How to Compare Before/After

After optimization, compare new baseline to old:

```bash
# Before optimization
bash tests/performance/timing-baseline.test.sh  # → baseline-before.json

# ... make optimizations ...

# After optimization
bash tests/performance/timing-baseline.test.sh  # → baseline-after.json

# Compare
generate_report baseline-before.json baseline-after.json
```

---

## Production Readiness Checklist

Before using Aether for production work:

- [ ] **End-to-end tests passing**: Run `bash tests/test-orchestrator.sh`
- [ ] **Stress tests passing**: Run tests in `tests/stress/`
- [ ] **Performance baselines established**: Run timing-baseline.test.sh
- [ ] **Checkpoint recovery verified**: Test `/ant:recover` workflow
- [ ] **Circuit breakers tested**: Verify spawn limits work

---

## Next Steps

1. **Read the Architecture:** See [QUEEN_ANT_ARCHITECTURE.md](/.aether/QUEEN_ANT_ARCHITECTURE.md) for complete design details

2. **Run the Tests:** Execute `bash .planning/phases/10-colony-maturity**---end-to-end-testing,-pattern-extraction,-production-readiness/tests/test-orchestrator.sh` to verify colony emergence

3. **Establish Baselines:** Run performance tests to measure your system's capabilities

4. **Initialize Your First Colony:** Use `/ant:init` to start your first project

5. **Learn by Doing:** Experiment with pheromone signals to guide colony behavior

---

**The Queen provides intention. The colony provides intelligence.** 🐜
