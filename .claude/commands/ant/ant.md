---
name: ant
description: Queen Ant Colony - phased autonomy where user provides intention via pheromones
---

<objective>
Display Queen Ant Colony system overview including how it works, key features, and getting started information.
</objective>

<reference>
# `/ant` - Queen Ant Colony System

## What is the Queen Ant System?

A phased autonomy system where:
- **Queen (User)** provides intention via pheromones (signals, not commands)
- **Colony** self-organizes within phases (pure emergence)
- **Phase boundaries** provide checkpoints for Queen review
- **Pheromones** guide behavior with strength/decay (signals, not orders)

## Key Difference from Original AETHER

| Aspect | Original AETHER | Queen Ant Model |
|--------|-----------------|-----------------|
| User role | Provide goal, wait | Signal provider, observer |
| Structure | Pure emergence | Phased autonomy |
| Visibility | Limited | Phase checkpoints |
| Planning | Autonomous | Queen reviews at boundaries |
| Feedback | After completion | Continuous via pheromones |

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

- **6 Worker Ant Castes**: Mapper, Planner, Executor, Verifier, Researcher, Synthesizer
- **Pheromone Signals**: Init, Focus, Redirect, Feedback
- **Phased Autonomy**: Structure at boundaries, emergence within
- **Learning System**: Colony learns from your patterns over time

## Research Foundation

Built on **25 research documents** (383K words, 758 references):
- Phase 1: Context Engine Foundation
- Phase 3: Semantic Codebase Understanding
- Phase 4: Predictive & Anticipatory Systems
- Phase 5: Advanced Verification & Quality
- Phase 6: Integration & Synthesis
- Phase 7: Implementation Planning

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
