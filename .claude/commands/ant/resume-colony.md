---
name: ant:resume-colony
description: Resume colony from saved session - restores all state
---

<objective>
Resume colony work from saved handoff document. Restores goal, pheromones, phase progress, Worker Ant states, and memory. Allows continuing in a new Claude session with clean context.
</objective>

<reference>
# `/ant:resume-colony` - Usage

## Command

```
/ant:resume-colony
```

## What It Does

Restores colony state from `.aether/PAUSED_SESSION.json`:
- Goal and intention
- All pheromone signals
- Phase progress and task states
- Worker Ant states
- Memory and learned patterns

## When to Use

- Starting a new Claude session after pausing
- After closing and reopening Claude Code
- When you want to continue where you left off

## Output

```
🐜 Queen Ant Colony - Resume Session

SESSION RESTORED:

  Goal: "Build a real-time chat application"
  Paused at: 2025-02-01T15:30:00

RESTORED PHASE: Phase 2 - Real-time Communication
  Status: IN_PROGRESS
  Tasks: 5/8 completed

STATE RESTORED:
  ✓ Goal and pheromones
  ✓ Phase progress
  ✓ Worker Ant states
  ✓ Memory and learned patterns

ACTIVE PHEROMONES:
  [INIT] Build chat app (strength: 100%)
  [FOCUS] WebSocket security (strength: 0.65)
  [FOCUS] message reliability (strength: 0.45)

✅ COLONY READY TO CONTINUE

You can now:
  • Continue where you left off
  • Use all /ant: commands normally
  • Colony remembers everything

📋 NEXT STEPS:
  1. /ant:status            - Check colony status
  2. /ant:phase             - Continue with phase
  3. /ant:focus <area>      - Add guidance if needed

💡 RECOMMENDATION:
   Review what was happening before pausing, then continue.

🔄 CONTEXT: REFRESHED
   You're in a new session with clean context.
   Colony state fully restored.
```

## How It Works

1. Loads handoff from `.aether/PAUSED_SESSION.json`
2. Restores goal and pheromones
3. Restores phase and task states
4. Restores Worker Ant states
5. Restores memory and learned patterns
6. Colony ready to continue

## Benefits

- Resume in new session with clean context
- No work is lost
- Colony remembers all your signals
- Seamless continuation

## Tips

- Always use `/ant:pause-colony` before closing Claude
- Resume in fresh Claude session for best results
- Colony state is fully restored

## Related Commands

```
/ant:pause-colony   - Pause and save session
/ant:status          - Check colony status
/ant:phase          - Continue with phase
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    return await commands.resume_colony()
</script>
