---
name: ant
description: Queen Ant Colony - phased autonomy where user provides intention via pheromones
---

<objective>
Display Queen Ant Colony system overview including how it works, key features, and getting started information.
</objective>

<reference>
# `/ant` - Aether Queen Ant Colony System

## What is Aether?

Aether is a **unique, standalone multi-agent system** built from first principles on ant colony intelligence. Unlike AutoGen, LangGraph, CrewAI, or any other framework, Aether implements:

- **True Autonomous Spawning** - Worker Ants spawn Worker Ants without human orchestration
- **Pheromone Communication** - Stigmergic signals guide colony behavior (not commands)
- **Phased Autonomy** - Structure at boundaries, pure emergence within
- **Queen-Based Control** - User provides intention via pheromones, colony self-organizes

## Aether vs Other Frameworks

| Aspect | AutoGen/LangGraph/CrewAI | Aether |
|--------|-------------------------|--------|
| Agent definition | Human predefines all agents | Colony creates agents autonomously |
| Workflow | Human defines orchestration | Colony self-organizes via pheromones |
| Spawning | Manual or predefined | Autonomous - Workers spawn Workers |
| Communication | Message passing | Stigmergic pheromone signals |
| Control flow | Predefined state machines | Emergent behavior within phases |

## Getting Started

### 1. Initialize Project

```bash
/ant:init "Build a real-time chat application"
```

Colony creates phase structure based on your goal.

### 2. Review Phases

```bash
/ant:plan
```

See all phases with tasks and milestones.

### 3. Guide Colony (Optional)

```bash
/ant:focus "WebSocket security"
/ant:focus "message reliability"
```

Guide colony attention to specific areas.

### 4. Execute Phase

```bash
/ant:phase 1      # Review Phase 1
/ant:execute 1    # Execute Phase 1
```

Colony self-organizes to complete tasks.

### 5. Review Work

```bash
/ant:review 1     # Review completed work
/ant:phase continue    # Continue to next phase
```

Review what was built, then continue.

## All Commands

### Core Workflow

| Command | What it does |
|---------|--------------|
| `/ant:init <goal>` | Initialize new project |
| `/ant:plan` | Show all phases |
| `/ant:phase [N]` | Show phase details |
| `/ant:execute <N>` | Execute a phase |
| `/ant:review <N>` | Review completed phase |
| `/ant:phase continue` | Continue to next phase |

### Guidance Commands

| Command | What it does |
|---------|--------------|
| `/ant:focus <area>` | Guide colony attention |
| `/ant:redirect <pattern>` | Warn colony away from approach |
| `/ant:feedback <message>` | Provide guidance |

### Status Commands

| Command | What it does |
|---------|--------------|
| `/ant:status` | Colony status |
| `/ant:memory` | Learned patterns |

## Key Features

- **6 Unique Worker Ant Castes**: Colonizer, Route-setter, Builder, Watcher, Scout, Architect (designed from first principles)
- **Pheromone Signal System**: Init, Focus, Redirect, Feedback (unique stigmergic communication)
- **Phased Autonomy**: Structure at boundaries, pure emergence within phases
- **Triple-Layer Memory**: Working → Short-term → Long-term with associative links
- **Voting-Based Verification**: Multi-perspective verification with belief calibration
- **Meta-Learning Loop**: Colony learns which specialists work best for which tasks

## Worker Ant Castes (Detailed)

| Caste | Function | Sensitivity | Spawns |
|-------|----------|-------------|--------|
| **Colonizer** | Codebase colonization, semantic indexing | INIT=1.0, FOCUS=0.7 | graph_builder, pattern_matcher |
| **Route-setter** | Goal decomposition, phase planning | INIT=1.0, REDIRECT=0.8 | estimator, dependency_analyzer |
| **Builder** | Code implementation, autonomous spawning | FOCUS=0.9, REDIRECT=0.9 | language_specialist, database_specialist |
| **Watcher** | Testing, validation, LLM-based test generation | FOCUS=0.8, FEEDBACK=0.9 | test_generator, security_scanner |
| **Scout** | Information gathering, research | FOCUS=0.9, INIT=0.7 | search_agent, documentation_reader |
| **Architect** | Memory compression, pattern extraction | FEEDBACK=0.6 | pattern_matcher, compression_agent |

## Pheromone Signals (Detailed)

| Signal | Strength | Duration | Effect | Learning |
|--------|----------|----------|--------|----------|
| **INIT** | 1.0 | Persists | Mobilize colony | - |
| **FOCUS** | 0.7 | 1hr half-life | Prioritize area | 3+ → Preference learned |
| **REDIRECT** | 0.7 | 24hr half-life | Avoid pattern | 3+ → Constraint created |
| **FEEDBACK** | 0.5-0.7 | 6hr half-life | Adjust behavior | Category-dependent |

## Autonomous Spawning System

### Capability Detection
- Analyzes task requirements vs own capabilities
- Uses semantic pattern matching for task categorization
- Spawns specialists based on capability gaps

### Resource Budgets
- Max subagents: 10 per phase
- Max spawn depth: 3 levels (parent → child → grandchild)
- Circuit breaker: 3 failed spawns → cooldown

### Specialist Mappings
- database/sql → database_specialist
- frontend (react/vue) → frontend_specialist
- api/websocket → api_specialist
- authentication/jwt → security_specialist
- testing → test_specialist
- performance → optimization_specialist

## Aether Architecture

Aether is **not** another framework wrapper. It's a complete standalone system with:

- **Unique caste system** - Each Worker Ant type has distinct behaviors and spawning capabilities
- **Stigmergic communication** - Environment (pheromones) as communication medium
- **Autonomous recruitment** - Workers detect capability gaps and spawn specialists automatically
- **Colony intelligence** - No central brain, distributed computation via emergence

This architecture is inspired by research on ant colonies, multi-agent systems, and stigmergic communication, but all implementations are uniquely Aether.

## Example Session

```bash
# Start new project
/ant:init "Build a REST API with JWT auth"

# Review phases
/ant:plan

# Guide colony
/ant:focus "security"
/ant:focus "test coverage"

# Execute Phase 1
/ant:execute 1

# Review completed work
/ant:review 1

# Continue to next phase
/ant:phase continue
```

## Tips

- **Be specific** with your goal, not how to achieve it
- **Review phases** before executing
- **Use focus** to guide colony attention
- **Provide feedback** to teach colony preferences
- **Refresh context** after phase execution

## Context Management

Each command tells you whether to continue or refresh:

- 🔄 **CONTEXT: Lightweight** - Safe to continue
- ⚠️ **CONTEXT: REFRESH RECOMMENDED** - Good checkpoint
- 🚨 **CONTEXT: REFRESH REQUIRED** - Memory intensive

Always follow context guidance for best results.
</reference>
