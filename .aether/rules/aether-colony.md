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

### Setup & Getting Started
| Command | Purpose |
|---------|---------|
| `/ant:lay-eggs` | Set up Aether in this repo (one-time, creates .aether/) |
| `/ant:init "<goal>"` | Start a colony with a goal |
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
First time in a repo:
0. /ant:lay-eggs                           (set up Aether in this repo)

Starting a colony:
1. /ant:init "Build feature X"             (start colony with a goal)
2. /ant:colonize                           (if existing code)
3. /ant:plan                               (generates phases)
4. /ant:focus "security"                   (optional guidance)
5. /ant:build 1                            (workers execute phase 1)
6. /ant:continue                           (verify, learn, advance)
7. /ant:build 2                            (repeat until complete)

After /clear or session break:
8. /ant:resume-colony                      (restore full context)
9. /ant:status                             (see where you left off)

After completing a colony:
10. /ant:seal                              (mark as complete)
11. /ant:entomb                            (archive to chambers)
12. /ant:init "next project goal"          (start fresh colony)
```

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
