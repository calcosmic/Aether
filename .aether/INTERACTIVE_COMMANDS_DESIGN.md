# Queen Ant Interactive Commands Design

## CDS-Like Interactive Flow

### Stages (Clear Progression)

```
1. /ant:init <goal>
   ↓ Colony creates phase structure
   ↓
2. /ant:plan
   ↓ Queen reviews phases
   ↓
3. /ant:phase 1 (review)
   ↓
4. /ant:focus "area" (optional guidance)
   ↓
5. /ant:execute 1
   ↓ Colony executes (pure emergence)
   ↓
6. /ant:review 1
   ↓ Queen reviews completed work
   ↓
7. /ant:phase continue (next phase)
   ↓ Loop 3-7 until complete
```

---

## Command Interface

### `/ant:init <goal>`

**What it does:**
- Queen sets intention
- Mapper explores codebase
- Planner creates phase structure
- Displays summary
- **Prompts for next actions**

**Output:**
```
🐜 Queen Ant Colony - Initialize Project

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Goal: "Build a real-time chat application"

COLONY RESPONSE:
  ✓ Mapper explored codebase
  ✓ Planner created phase structure

PHASES CREATED: 5
  Phase 1: Foundation (5 tasks)
  Phase 2: Real-time Communication (8 tasks)
  Phase 3: User Authentication (5 tasks)
  Phase 4: Message Features (7 tasks)
  Phase 5: Testing & Deployment (5 tasks)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:plan              - Review all phases in detail
  2. /ant:phase 1           - Review Phase 1 before starting
  3. /ant:focus <area>      - Guide colony attention (optional)

💡 RECOMMENDATION: Run /ant:plan to see the full roadmap
```

---

### `/ant:plan`

**What it does:**
- Display all phases with tasks
- Show current status
- **Prompts for next actions**

**Output:**
```
🐜 Queen Ant Colony - Phase Plan

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GOAL: Build a real-time chat application

PHASE 1: Foundation [PENDING]
  Tasks: 5
  • Setup project structure
  • Configure development environment
  • Initialize database schema
  • Setup WebSocket server
  • Implement basic message routing
  Milestones: WebSocket running, Database connected

PHASE 2: Real-time Communication [PENDING]
  Tasks: 8
  • Implement WebSocket connection handling
  • Create message queue system
  • Configure Redis pub/sub
  • Add connection pooling
  • Implement message delivery
  • Add message persistence
  • Create offline message handling
  • Add message acknowledgment
  Milestones: Real-time delivery, Message persistence

[... more phases ...]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:phase 1           - Review Phase 1 details
  2. /ant:execute 1         - Start executing Phase 1
  3. /ant:focus <area>      - Add focus guidance (optional)

💡 RECOMMENDATION: Review Phase 1 with /ant:phase 1 before executing

🔄 CONTEXT: This command is lightweight - safe to continue
```

---

### `/ant:phase [N]`

**What it does:**
- Show current phase or specific phase
- Show tasks, progress, ants working
- **Prompts for next actions based on phase state**

**Output (phase not started):**
```
🐜 Queen Ant Colony - Phase 1: Foundation

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

STATUS: PENDING
TASKS: 5

TASKS:
  ⏳ Setup project structure
  ⏳ Configure development environment
  ⏳ Initialize database schema
  ⏳ Setup WebSocket server
  ⏳ Implement basic message routing

MILESTONES:
  • WebSocket server running
  • Database connected

ESTIMATED DURATION: 45 minutes

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:execute 1         - Start executing this phase
  2. /ant:focus <area>      - Guide colony before execution (optional)
  3. /ant:plan              - Back to full plan

💡 COLONY RECOMMENDATION:
   Consider focusing on: "WebSocket setup" or "Database schema"
   Use /ant:focus to guide attention

🔄 CONTEXT: This command is lightweight - safe to continue
```

**Output (phase in progress):**
```
🐜 Queen Ant Colony - Phase 1: Foundation

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

STATUS: IN PROGRESS (60% complete)
STARTED: 23 minutes ago

TASKS:
  ✓ Setup project structure
  ✓ Configure development environment
  ✓ Initialize database schema
  → Setup WebSocket server (in progress)
  ⏳ Implement basic message routing

ACTIVE WORKER ANTS:
  EXECUTOR: Implementing WebSocket server
    → Spawned: python_specialist, websocket_specialist
  VERIFIER: Testing database connections
    → Spawned: test_generator

ACTIVE PHEROMONES:
  [FOCUS] WebSocket security (strength: 0.7)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:status            - Check detailed colony status
  2. /ant:focus <area>      - Add additional focus
  3. /ant:feedback <msg>     - Provide guidance to colony
  4. /ant:review 1          - Review completed work

💡 COLONY RECOMMENDATION:
   Phase progressing well. Consider: /ant:focus "message routing"

⚠️ CONTEXT: Phase execution is memory-intensive.
   Consider /ant:review after completion before continuing.
```

**Output (phase complete, awaiting review):**
```
🐜 Queen Ant Colony - Phase 1: Foundation

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

STATUS: COMPLETE ✓
DURATION: 47 minutes
TASKS: 5/5 completed

KEY LEARNINGS:
  • WebSocket pooling reduces connections by 40%
  • Database connection pool improves performance

ISSUES FOUND & FIXED:
  • 3 bugs (all resolved)

SPAWNED AGENTS: 8
MESSAGES EXCHANGED: 23

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:

  1. /ant:review 1          - Review completed work (recommended)
  2. /ant:phase continue    - Continue to next phase
  3. /ant:focus <area>      - Set focus for next phase

💡 COLONY RECOMMENDATION:
   Review completed work before continuing.
   Use /ant:review 1 to see what was built.

🔄 CONTEXT: Phase complete - good time to refresh context
   After /ant:review, use /ant:phase continue
```

---

### `/ant:execute <N>`

**What it does:**
- Execute a phase with pure emergence
- Colony self-organizes
- **Updates progress in real-time**
- **Prompts for interaction during execution**

**Output:**
```
🐜 Queen Ant Colony - Executing Phase 1

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Starting Phase 1: Foundation
Tasks: 5
Estimated: 45 minutes

[COLONY SELF-ORGANIZING]

Task 1/5: Setup project structure
  → Executor spawned: filesystem_specialist
  → Complete (2 minutes)

Task 2/5: Configure development environment
  → Executor spawned: config_specialist
  → Complete (3 minutes)

Task 3/5: Initialize database schema
  → Executor spawned: database_specialist
  → Complete (8 minutes)

Task 4/5: Setup WebSocket server
  → Executor spawned: websocket_specialist
  → Verifier spawned: security_scanner
  → Complete (12 minutes)

Task 5/5: Implement basic message routing
  → Executor spawned: routing_specialist
  → Verifier testing message flow
  → Complete (15 minutes)

[PHASE COMPLETE]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE SUMMARY:
  ✓ 5/5 tasks completed
  ✓ 2 milestones reached
  ✓ 3 issues found and fixed
  ⏱️ Total time: 40 minutes

📋 NEXT STEPS:

  1. /ant:review 1          - Review completed work
  2. /ant:phase continue    - Continue to next phase

💡 COLONY RECOMMENDATION:
   Review work before continuing.

🔄 CONTEXT: REFRESH RECOMMENDED
   Phase execution used significant context.
   Refresh Claude with /ant:review 1 before continuing.
```

---

### `/ant:review <N>`

**What it does:**
- Review completed phase
- Show what was built
- **Prompts for next actions**
- **Context refresh recommendation**

**Output:**
```
🐜 Queen Ant Colony - Phase 1 Review

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PHASE 1: Foundation - COMPLETE

WHAT WAS BUILT:
  Files created/modified:
    • project/setup.py
    • project/config.py
    • database/schema.sql
    • websocket/server.py
    • routing/handlers.py

FEATURES IMPLEMENTED:
  ✓ Project structure with modular architecture
  ✓ Development environment configuration
  ✓ PostgreSQL database with connection pooling
  ✓ WebSocket server with connection pooling
  ✓ Basic message routing between clients

KEY LEARNINGS:
  • Connection pooling reduces overhead by 40%
  • Modular structure enables parallel development

ISSUES RESOLVED:
  • WebSocket timeout issue (fixed with heartbeat)
  • Database connection leak (fixed with pool limits)
  • Routing race condition (fixed with queue)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

QUEEN FEEDBACK:
  /ant:feedback "Great work on connection pooling"
  /ant:feedback "Need better error handling in routing"

📋 NEXT STEPS:

  1. /ant:phase continue    - Continue to Phase 2
  2. /ant:focus <area>      - Set focus for next phase
  3. /ant:status            - Check overall status

💡 COLONY RECOMMENDATION:
   Ready for Phase 2: Real-time Communication
   Consider: /ant:focus "WebSocket security"

🔄 CONTEXT: REFRESH RECOMMENDED
   This is a clean checkpoint - safe to refresh Claude
   and continue with /ant:phase continue
```

---

## Context Management

### When to Refresh Context

| Situation | Action |
|-----------|--------|
| After `/ant:plan` | Continue - lightweight |
| After `/ant:phase` | Continue - lightweight |
| **After `/ant:execute`** | **REFRESH - then review** |
| After `/ant:review` | Continue - checkpoint |
| When memory > 60% | REFRESH |

### Clear Prompts

Each command tells you:
- 🔄 **CONTEXT** section: Whether to continue or refresh
- 📋 **NEXT STEPS**: Clear options
- 💡 **RECOMMENDATION**: What the colony suggests

---

## Implementation Priority

1. **Update `/ant:init` command** - Interactive with prompts
2. **Update `/ant:plan` command** - Show all phases
3. **Update `/ant:phase` command** - State-aware prompts
4. **Add `/ant:execute` command** - Interactive execution
5. **Add `/ant:review` command** - Phase review
6. **Add context management** - Track context usage

This makes it work like CDS - clear stages, always know what's next, colony recommendations, context guidance.
