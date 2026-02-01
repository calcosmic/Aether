# Queen Ant System - Context Handoff

**If context collapses, start here.**

---

## What We're Building

**Queen Ant Colony**: A phased autonomy system where:
- User (Queen) provides intention via pheromones (signals)
- Colony self-organizes within phases
- Pure emergence, guided by user feedback
- Phase boundaries = checkpoints with Queen

**Key difference from original AETHER**:
- Original: Goal → Agents figure it out (AI synthesis)
- Queen Ant: Queen signals → Colony self-organizes (your vision)

---

## Core Architecture Document

`/Users/callumcowie/repos/cosmic-dev-system/.aether/QUEEN_ANT_ARCHITECTURE.md`

This is the **complete architecture**. Read it first.

---

## The Commands (All /ant:)

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

**No command verbs. No assignments. Just signals.**

---

## Worker Ant Castes (6 Pre-Defined)

| Caste | Role | Spawns |
|-------|------|--------|
| **Colonizer** | Colonize, index, understand codebase | Graph agents, search agents |
| **Route-setter** | Create structured phase plans | Estimators, risk assessors |
| **Builder** | Build code, implement | Language/framework specialists |
| **Watcher** | Watch, validate, QA | Test generators, security scanners |
| **Scout** | Scout for information, context | Search agents, crawlers |
| **Architect** | Architect memory, extract patterns | Analysis agents |

These are **always available**. They don't emerge - they're the colony's structure.

---

## How It Works

```
1. /ant:init "Build a chat app"
   → Intention pheromone released

2. Colony detects pheromone
   → Colonizer explores codebase
   → Route-setter creates phase structure

3. /ant:phase
   → Queen reviews phase plan
   → /ant:focus "prioritize WebSocket security"

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
```

---

## Key Principles

1. **Queen provides intention, not commands**
2. **Colony self-organizes within phases**
3. **Pheromones = user signals that guide behavior**
4. **Phase boundaries = checkpoints with Queen**
5. **Pure emergence within structured phases**

---

## Pheromone System

| Signal | Command | Effect | Duration |
|--------|---------|--------|----------|
| **Init** | `/ant:init` | Strong attract. Triggers planning. | Persists |
| **Focus** | `/ant:focus` | Medium attract. Guides attention. | 1 hour |
| **Redirect** | `/ant:redirect` | Strong repel. Warns away. | 24 hours |
| **Feedback** | `/ant:feedback` | Variable. Adjusts behavior. | 6 hours |

Signals decay over time. Recent signals are stronger.

---

## Current Task List

```
#5 [completed] Design Queen Ant colony architecture
#6 [pending] Implement Worker Ant castes
#7 [pending] Build /ant command interface
#8 [pending] Implement pheromone signal system
#9 [pending] Build phase execution engine
```

**Next task**: Implement Worker Ant castes (#6)

---

## File Locations

```
/Users/callumcowie/repos/cosmic-dev-system/.aether/
├── QUEEN_ANT_ARCHITECTURE.md    # Complete architecture (READ THIS)
├── HANDOFF.md                   # This file
├── worker_ants.py               # [TO BUILD] Worker Ant implementations
├── pheromone_system.py          # [TO BUILD] Signal layer
├── phase_engine.py              # [TO BUILD] Phase execution
└── commands/
    └── ant/                     # [TO UPDATE] /ant: commands
```

---

## Research Foundation

All design grounded in 25 research documents (383K words, 758 refs):

```
/Users/callumcowie/repos/cosmic-dev-system/.ralph/research/
```

Key research supporting this architecture:
- **Phase 1**: Semantic protocols, memory hierarchy
- **Phase 3**: Semantic codebase understanding
- **Phase 4**: Anticipatory systems, feedback loops
- **Phase 5**: Verification, quality assurance
- **Phase 6**: Multi-agent coordination patterns
- **Phase 7**: Implementation planning

---

## Resume Here

If context collapses:

1. **Read**: `QUEEN_ANT_ARCHITECTURE.md` (complete design)
2. **Check**: Task list via `TaskList` tool
3. **Continue**: Next pending task (implementing Worker Ant castes)
4. **Remember**: Queen signals, colony emerges. No commands.

---

## Quick Mental Model

```
Queen (User)
  ↓ signals (pheromones)
Worker Ants (pre-defined castes)
  ↓ spawn (subagents emerge)
Colony Intelligence (pure emergence)
```

**Your vision, their execution.**
