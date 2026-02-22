# Aether Colony System

> This repo uses the Aether colony system for multi-agent development.
> These rules are auto-distributed by `aether update` — do not edit directly.

## Session Recovery

On the first message of a new conversation, check if `.aether/data/session.json` exists. If it does:

1. Read the file briefly to check for `colony_goal`
2. If a goal exists, display:
   ```
   Previous colony session detected: "{goal}"
   Run /ant:resume to restore context, or continue with a new topic.
   ```
3. Do NOT auto-restore — wait for the user to explicitly run /ant:resume

This only applies to genuinely new conversations, not after /clear.

## Available Commands

### Getting Started
| Command | Purpose |
|---------|---------|
| `/ant:init "<goal>"` | Set colony intention and initialize |
| `/ant:colonize` | Analyze existing codebase |
| `/ant:plan` | Generate project phases |
| `/ant:build <phase>` | Execute a phase with parallel workers |
| `/ant:continue` | Verify work, extract learnings, advance |

### Pheromone Signals
| Command | Priority | Purpose |
|---------|----------|---------|
| `/ant:focus "<area>"` | normal | Guide colony attention |
| `/ant:redirect "<pattern>"` | high | Hard constraint — avoid this |
| `/ant:feedback "<note>"` | low | Gentle adjustment |

### Status & Monitoring
| Command | Purpose |
|---------|---------|
| `/ant:status` | Colony dashboard |
| `/ant:phase [N]` | View phase details |
| `/ant:flags` | List active flags |
| `/ant:flag "<title>"` | Create a flag |
| `/ant:history` | Browse colony events |
| `/ant:watch` | Live tmux monitoring |

### Session Management
| Command | Purpose |
|---------|---------|
| `/ant:pause-colony` | Save state and create handoff |
| `/ant:resume-colony` | Restore from pause |
| `/ant:resume` | Quick session restore |

### Lifecycle
| Command | Purpose |
|---------|---------|
| `/ant:seal` | Seal colony (Crowned Anthill) |
| `/ant:entomb` | Archive completed colony |
| `/ant:maturity` | View colony maturity journey |
| `/ant:update` | Update system files from hub |

### Advanced
| Command | Purpose |
|---------|---------|
| `/ant:swarm "<bug>"` | Parallel bug investigation |
| `/ant:oracle` | Deep research (RALF loop) |
| `/ant:dream` | Philosophical observation |
| `/ant:interpret` | Review dreams, discuss actions |
| `/ant:chaos` | Resilience testing |
| `/ant:archaeology` | Git history analysis |
| `/ant:organize` | Codebase hygiene report |
| `/ant:council` | Intent clarification |

## Typical Workflow

```
/ant:init "Build feature X"    → Set colony goal
/ant:colonize                  → Understand existing code (optional)
/ant:plan                      → Generate phases
/ant:focus "security"          → Steer attention (optional)
/ant:build 1                   → Execute phase 1
/ant:continue                  → Verify, learn, advance
/ant:build 2                   → Execute phase 2
...repeat until complete...
/ant:seal                      → Seal completed colony
```

After `/clear` or session break: `/ant:resume-colony` to restore context.

## Worker Castes

Workers are assigned to castes based on task type:

| Caste | Role |
|-------|------|
| builder | Implementation work |
| watcher | Monitoring, quality checks |
| scout | Research, discovery |
| chaos | Edge case testing |
| oracle | Deep research (RALF loop) |
| architect | Planning, design |
| colonizer | Codebase exploration |
| route_setter | Phase planning |
| archaeologist | Git history analysis |

## Protected Paths

**Never modify these programmatically:**

| Path | Reason |
|------|--------|
| `.aether/data/` | Colony state (COLONY_STATE.json, session files) |
| `.aether/dreams/` | Dream journal entries |
| `.aether/checkpoints/` | Session checkpoints |
| `.aether/locks/` | File locks |

## Colony State

State is stored in `.aether/data/COLONY_STATE.json` and includes:
- Colony goal and current phase
- Task breakdown and completion status
- Instincts (learned patterns with confidence scores)
- Pheromone signals (FOCUS/REDIRECT/FEEDBACK)
- Event history

## Pheromone System

Signals guide colony behavior without hard-coding instructions:
- **FOCUS** — attracts attention to an area (expires at phase end)
- **REDIRECT** — repels workers from a pattern (high priority, hard constraint)
- **FEEDBACK** — calibrates behavior based on observation (low priority)

Use FOCUS + REDIRECT before builds to steer. Use FEEDBACK after builds to adjust.
