# Architecture Patterns: CLI-Based Multi-Agent Systems

**Domain:** Multi-agent CLI orchestration for AI coding assistants
**Researched:** 2026-02-13
**Confidence:** HIGH

This document analyzes the Aether Colony System architecture and identifies patterns that work well for CLI-based multi-agent systems.

---

## Recommended Architecture: Queen-Worker Hierarchy

The Aether Colony System uses a proven hierarchical pattern that balances autonomy with control:

```
┌─────────────────────────────────────────────────────────────────┐
│                        QUEEN (User)                             │
│  Provides goal, constraints (focus/avoid), observation          │
│  Never directly executes work — only orchestrates               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                    Signals (not commands)
                         │
                         ▼
        ┌─────────────────────────────────────────────┐
        │            CONSTRAINTS LAYER                 │
        │  Focus areas + Avoid patterns (declarative)  │
        │  Workers read at spawn time, not Queen-pushed│
        └────────────────┬──────────────────────────────┘
                         │
                         ▼
        ┌─────────────────────────────────────────────┐
        │              PHASE PIPELINE                 │
        │  /ant:plan → /ant:build N → /ant:continue  │
        └────────────────┬──────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    PRIME WORKDepth 1)ER (                   │
│  Coordinator — spawns up to 4 specialists                   │
│  Owns phase completion, synthesizes results                  │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   BUILDER    │  │   WATCHER    │  │    SCOUT     │
│   Depth 2    │  │   Depth 2    │  │   Depth 2   │
│(can spawn 2)│  │(can spawn 2) │  │(can spawn 2)│
└──────────────┘──┘   └──────────── └──────────────┘
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│Sub-Builder   │  │Sub-Watcher   │  │  Sub-Scout   │
│   Depth 3    │  │   Depth 3    │  │   Depth 3    │
│ (no spawn)  │  │ (no spawn)   │  │ (no spawn)   │
└──────────────┘  └──────────────┘  └──────────────┘
```

---

## Component Boundaries

### 1. Queen Layer (Orchestration)

| Component | Responsibility | Boundaries |
|-----------|---------------|------------|
| `/ant:init` | Goal initialization, state creation | Creates COLONY_STATE.json only |
| `/ant:plan` | Iterative research/planning loop | Outputs plan to COLONY_STATE.json |
| `/ant:build` | Spawns Prime Worker | Cannot execute work directly |
| `/ant:continue` | Verification gates, phase advance | Reads state, writes events |
| Constraint commands | Signal injection | Modifies constraints.json only |

**Key principle:** Queen never executes code. It only spawns workers and modifies state.

### 2. Constraints Layer (Signaling)

| Component | Responsibility | Boundaries |
|-----------|---------------|------------|
| `constraints.json` | Declarative focus/avoid rules | No execution logic |
| `/ant:focus` | Add attention directive | Max 5 focus areas |
| `/ant:redirect` | Add avoidance directive | Max 10 avoid patterns |
| `/ant:council` | Interactive multi-signal injection | Translates to constraints |

**Key principle:** Constraints are read by workers at spawn time, not pushed during execution.

### 3. Worker Layer (Execution)

| Caste | Depth | Spawns | Responsibility |
|-------|-------|--------|----------------|
| Prime Worker | 1 | 4 max | Coordinate phase, delegate, synthesize |
| Builder | 2 | 2 max (if surprised) | Implement code, run commands |
| Watcher | 2 | 2 max (if surprised) | Test, validate, quality gates |
| Scout | 2 | 2 max (if surprised) | Research, gather context |
| Sub-worker | 3 | 0 | Complete work inline |

**Key principle:** Workers spawn workers directly. Queen does not mediate spawns after initial Prime Worker creation.

### 4. Utility Layer (Infrastructure)

| Component | Purpose | Location |
|-----------|---------|----------|
| `aether-utils.sh` | Single entry point for all deterministic ops | `.aether/aether-utils.sh` |
| File locking | Prevents race conditions | `.aether/utils/file-lock.sh` |
| Atomic writes | Prevents corruption | `.aether/utils/atomic-write.sh` |
| Activity logging | Observable worker actions | `.aether/data/activity.log` |

---

## Data Flow

### State Mutation Flow

```
1. User invokes command
       │
       ▼
2. Command validates prerequisites
       │
       ▼
3. Command calls aether-utils.sh subcommand
       │
       ▼
4. Utils reads/writes JSON state files
       │
       ▼
5. Output JSON to stdout, errors to stderr
```

### Worker Spawn Flow

```
1. /ant:build N triggers
       │
       ▼
2. Queen spawns Prime Worker (depth 1)
       │
       ▼
3. Prime Worker reads constraints.json
       │
       ▼
4. Prime Worker decides to spawn specialists
       │
       ▼
5. Specialist spawns sub-specialist (depth 2→3)
       │
       ▼
6. Depth 3 completes work inline
```

### Constraint Propagation Flow

```
1. User: /ant:focus "security"
       │
       ▼
2. /ant:focus writes to constraints.json
       │
       ▼
3. Next spawned worker reads constraints.json
       │
       ▼
4. Worker incorporates into prompt
```

---

## Build Order for New Projects

Based on the Aether architecture, the recommended implementation order:

### Phase 1: Core Infrastructure
- State management (COLONY_STATE.json schema)
- Basic command dispatch (aether-utils.sh)
- File locking and atomic writes

### Phase 2: Queen Commands
- `/ant:init` — Initialize state
- `/ant:status` — Read state
- `/ant:focus`, `/ant:redirect` — Constraint management

### Phase 3: Worker System
- Worker spawn protocol
- Depth-based behavior enforcement
- Worker prompt templates

### Phase 4: Execution Pipeline
- `/ant:plan` — Iterative planning loop
- `/ant:build` — Phase execution
- `/ant:continue` — Verification gates

### Phase 5: Advanced Features
- `/ant:swarm` — Parallel investigation
- `/ant:council` — Interactive clarification
- `/ant:watch` — tmux live monitoring

---

## Patterns That Work

### Pattern 1: Declarative Constraints Over Directives

**What:** Replace command-style orchestration with declarative constraint files.

**Why:** Workers can read constraints at spawn time without complex state machine logic.

**Example:**
```bash
# Instead of: QUEEN → "do X then Y"
# Use: constraints.json → ["focus": "security", "avoid": "eval"]
```

### Pattern 2: Workers Spawn Workers

**What:** Prime Workers use Task tool to spawn specialists directly.

**Why:** True emergence — structure comes from work, not orchestration.

**Implementation:**
- Depth 1: Prime Worker spawns up to 4 specialists
- Depth 2: Specialists spawn up to 2 (only if surprised)
- Depth 3: No spawning, complete inline

### Pattern 3: State File as Source of Truth

**What:** All colony state in JSON files, not in-memory.

**Why:** Survives context resets, enables debugging, supports handoff.

**Files:**
- `COLONY_STATE.json` — Goal, plan, phase, memory
- `constraints.json` — Focus/avoid rules
- `flags.json` — Issue tracking
- `activity.log` — Event stream

### Pattern 4: Single Entry Point for Deterministic Ops

**What:** All state mutations go through `aether-utils.sh`.

**Why:** Testable, consistent, single source of truth.

**Example:**
```bash
# Any command that modifies state:
bash .aether/aether-utils.sh activity-log "ACTION" "caste" "description"
```

### Pattern 5: Depth-Based Behavior Limits

**What:** Worker capabilities vary by spawn depth.

**Why:** Prevents infinite spawn loops, creates natural termination.

| Depth | Max Spawns | Typical Role |
|-------|------------|--------------|
| 1 | 4 | Coordinator |
| 2 | 2 | Specialist |
| 3 | 0 | Worker |

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Queen as Worker

**What:** Queen executes code or directly manages workers.

**Why:** Violates separation of concerns, makes state management complex.

**Instead:** Queen only spawns Prime Worker; worker manages sub-spawns.

### Anti-Pattern 2: Real-Time Signal Pushing

**What:** Queen pushes pheromones to in-flight workers.

**Why:** Race conditions, worker context already loaded.

**Instead:** Constraints read at spawn time only. New workers pick up changes.

### Anti-Pattern 3: In-Memory State Only

**What:** Colony state stored only in shell variables.

**Why:** Lost on context reset, no debugging, no handoff.

**Instead:** JSON files in `.aether/data/`, loaded at command start.

### Anti-Pattern 4: Unbounded Spawning

**What:** Workers can spawn unlimited sub-workers.

**Why:** Resource exhaustion, infinite loops, no termination.

**Instead:** Depth-based limits (1→4→2→0), total cap (10 workers/phase).

### Anti-Pattern 5: Complex Pheromone Math

**What:** Signal strength calculations, decay functions, sensitivity profiles.

**Why:** Hard to debug, unpredictable behavior, over-engineered.

**Instead:** Simple declarative constraints with no decay.

---

## Scalability Considerations

| Scale | Concern | Approach |
|-------|---------|----------|
| 10 workers | Basic spawning | In-memory spawn tracking |
| 100 workers | Resource limits | Hard caps per phase |
| 1000 workers | Coordination | Separate colonies per project |
| Cross-session | State persistence | JSON files in `.aether/data/` |
| Parallel execution | Race conditions | File locking utilities |

---

## Verification Protocol

Each phase should pass before advancing:

1. **Build gate** — Code compiles/installs
2. **Types gate** — Type checking passes
3. **Lint gate** — Code style compliant
4. **Tests gate** — Tests pass
5. **Security gate** — No vulnerabilities
6. **Diff gate** — Human review of changes

---

## Sources

- Aether Colony System implementation (`/Users/callumcowie/repos/Aether/runtime/QUEEN_ANT_ARCHITECTURE.md`)
- Aether utilities layer (`/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`)
- Aether README (`/Users/callumcowie/repos/Aether/README.md`)

---

## Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Component boundaries | HIGH | Based on existing working implementation |
| Data flow | HIGH | Verified from aether-utils.sh |
| Build order | MEDIUM | Inferred from architecture dependencies |
| Patterns to follow | HIGH | Proven in production |
| Anti-patterns | HIGH | Lessons learned from v2.0 |
