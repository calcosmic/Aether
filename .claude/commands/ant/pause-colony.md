---
name: ant:pause-colony
description: Pause colony work and create handoff document for resuming later
---

<objective>
Pause colony work mid-phase and create a handoff document. Saves all state including goal, pheromones, phase progress, Worker Ant states, and memory. Allows you to resume work later in a new Claude session.
</objective>

<reference>
# `/ant:pause-colony` - Usage

## Command

```
/ant:pause-colony
```

## What It Does

Saves current colony state to `.aether/PAUSED_SESSION.json`:
- Current goal and pheromones
- Phase progress and task states
- Worker Ant activity
- Memory and learned patterns

## When to Use

- When you need to stop working mid-phase
- Before closing Claude Code
- When context is getting full
- When you need to take a break

## Output

```
🐜 Queen Ant Colony - Pause & Save Session

SAVED PHASE: Phase 2 - Real-time Communication
STATUS: IN_PROGRESS
TASKS: 8 total
PROGRESS: 5/8 tasks completed

SAVED STATE:
  ✓ Current goal and pheromones
  ✓ Worker Ant states
  ✓ Phase progress
  ✓ Memory and learned patterns

HANDOFF FILE:
  → .aether/PAUSED_SESSION.json

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 NEXT STEPS:
  1. /ant:resume-colony    - Resume from saved session
  2. Start new Claude session → Then run resume command

💡 TIP:
   Use pause when you need to stop mid-phase.
   Colony will be ready to continue when you resume.

🔄 CONTEXT: PERFECT CHECKPOINT
   Refreshing Claude is recommended after pause.
   Resume in new session with clean context.
```

## Resume Later

In a new Claude session:
```
/ant:resume-colony
```

Colony restores all state and you can continue where you left off.

## Benefits

- Save progress mid-phase
- Resume in new session with clean context
- Colony remembers everything
- No work is lost

## Related Commands

```
/ant:resume-colony   - Resume from paused session
/ant:status          - Check colony status
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    return await commands.pause_colony()
</script>
