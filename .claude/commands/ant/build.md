---
name: ant:build
description: Execute a goal with autonomous agent spawning - watch agents spawn agents to build what you want
---

<objective>
Execute a high-level goal using AETHER's autonomous agent spawning system.

User provides a goal description (e.g., "Build a blog with comments"), AETHER:
1. Decomposes the goal into subtasks autonomously
2. Spawns specialist agents as needed
3. Coordinates execution without human direction
4. Shows real-time agent hierarchy as agents spawn
5. Completes the goal autonomously

This is the PRIMARY interface to AETHER's revolutionary agent spawning capability.
</objective>

<reference>
# `/ant:build` - Execute Goals with Autonomous Agent Spawning

## What It Does

Takes a high-level goal and autonomously builds it using agents that spawn other agents.

**Example:**
```
/ant:build "Build a blog with markdown support, comments, and admin panel"
```

AETHER will:
1. **Decompose** into subtasks (database, API, frontend, etc.)
2. **Spawn specialists** for each capability gap
3. **Coordinate** execution autonomously
4. **Learn** from any mistakes
5. **Report** results when complete

## What Makes This Revolutionary

**Traditional systems** (AutoGen, LangGraph, CDS):
```
Human → Orchestrator → Predefined Agents → Result
```

**AETHER**:
```
Goal → Agent → Detects Gap → Spawns Specialist → Coordinates → Completes
```

**No human orchestration required.** This has never existed before.

## Usage

### Basic Usage

```
/ant:build "Build a REST API with JWT authentication"
```

### Complex Goals

```
/ant:build "Create a real-time chat app with WebSockets, user profiles, and message history"
```

### Code-Specific Goals

```
/ant:build "Add OAuth login to this project"
```

## What You'll See

### Real-Time Agent Spawning

```
🐜 AETHER: Executing Goal
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎯 Goal: "Build authentication system"

📋 Decomposed into 5 subtasks

Task 1/5: Plan architecture
  🔄 SPAWNING: Orchestrator → Architecture-Specialist

Task 2/5: Design database schema
  🔄 SPAWNING: Orchestrator → Database-Specialist

Task 3/5: Build authentication API
  🔄 SPAWNING: API-Specialist (already exists!)

Task 4/5: Create login UI
  🔄 SPAWNING: Orchestrator → Frontend-Specialist

Task 5/5: Test implementation
  🔄 SPAWNING: Orchestrator → QA-Specialist

✅ All tasks completed! Agents terminating...
```

### System Statistics

After execution, you'll see:
- **Agents spawned**: How many agents were created
- **Messages exchanged**: Semantic communication between agents
- **Memory operations**: Working/short-term/long-term memory stats
- **Error prevention**: Validations performed, errors logged

## Agent Hierarchy Example

```
Build authentication system (Goal Agent)
├── Architecture-Specialist
│   └── Database-Specialist
├── API-Specialist
│   └── Security-Specialist
├── Frontend-Specialist
└── QA-Specialist
```

## Memory Integration

All spawned agents share **triple-layer memory**:

- **Working Memory**: Current task context (shared across all agents)
- **Short-Term Memory**: Recent session compressed (10 sessions)
- **Long-Term Memory**: Persistent patterns and learnings

Agents can access:
```
/ant:memory "authentication patterns"  # Query long-term memory
```

## Error Prevention

AETHER learns from every execution:

- **Logs** every error with symptom/root cause/fix/prevention
- **Auto-flags** after 3 occurrences of same error category
- **Validates** actions BEFORE execution (guardrails)
- **Creates constraints** to prevent recurring mistakes

## Goal Examples

### Web Development

```
/ant:build "Build a blog with markdown support and comments"
/ant:build "Create a todo app with user authentication"
/ant:build "Build an e-commerce site with shopping cart"
```

### Features

```
/ant:build "Add real-time notifications with WebSockets"
/ant:build "Implement search with Elasticsearch"
/ant:build "Create admin panel with role-based access"
```

### Analysis

```
/ant:build "Analyze this codebase for security vulnerabilities"
/ant:build "Refactor database schema for better performance"
/ant:build "Generate API documentation from code"
```

## Success Indicators

When execution completes, you'll see:
- ✅ All subtasks completed
- ✅ Agents terminated cleanly
- ✅ Memory compressed to short-term
- ✅ Any errors logged (if failures occurred)

## Error Handling

If something goes wrong:
- **Error logged** to error ledger with full details
- **Specialist spawned** to handle the error (if possible)
- **Auto-flag** created if error category reaches threshold
- **Continue** with remaining tasks (resilient execution)

## Tips for Best Results

### Be Specific

```
✅ Good: "Build a blog with markdown support"
❌ Vague: "Build something cool"
```

### Include Tech Stack (Optional)

```
/ant:build "Build a Python FastAPI blog with PostgreSQL"
```

### Use Context

```
/ant:build "Add OAuth login to existing authentication system"
```

AETHER will remember previous work and adapt accordingly.

## System Requirements

AETHER requires:
- Python 3.8+
- The AETHER core system (`.aether/aether.py`)
- Memory system modules (`.aether/memory_system.py`)
- Error prevention modules (`.aether/error_prevention.py`)

All components are pre-built and tested.

## Output Format

```
✅ Goal Execution Complete

Results:
  [task_name]: [completion status]
  ...

Agents Spawned: X
Messages Exchanged: Y
Memory Operations: Z

Time Elapsed: X seconds

Agent Hierarchy:
  → AgentName (state)
    → ChildAgent (state)
      ...
```

## Related Commands

```
/ant                    # Show system overview
/ant:status            # Show detailed system status
/ant:memory            # View memory system
/ant:errors            # View error ledger
```

## Philosophical Note

🐜 **Why "ant"?**

Ant colonies demonstrate emergent intelligence without central control:
- No single ant directs the colony
- Each ant acts autonomously based on local cues
- Complex behavior emerges from simple rules
- Collective intelligence exceeds individual capability

**AETHER brings this to AI development.**

Traditional: Human → Orchestrator → Agents
AETHER: Goal → Agent → Spawn → Coordinate → Complete

This is **paradigm-shifting technology.**

"The whole is greater than the sum of its parts." - Aristotle (applies to both ant colonies and AETHER) 🐜
</reference>
