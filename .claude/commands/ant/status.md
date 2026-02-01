---
name: ant:status
description: Show Queen Ant Colony status - Worker Ants, pheromones, phase progress
---

<objective>
Display comprehensive colony status including Worker Ant activity, active pheromones, phase progress, and colony health.
</objective>

<reference>
# `/ant:status` - Usage

## Command

```
/ant:status
```

## What It Shows

- Current goal and phase
- Worker Ant activity (6 castes)
- Active subagents
- Active pheromones
- Phase progress

## Output Example

```
🐜 QUEEN ANT COLONY STATUS

GOAL: Build a real-time chat application
PHASE: Phase 2 (Real-time Communication)

WORKER ANTS:
  MAPPER [IDLE]: None
  PLANNER [IDLE]: None
  EXECUTOR [ACTIVE]: Implementing message persistence
    → 3 subagents active
  VERIFIER [ACTIVE]: Testing message delivery
    → 2 subagents active
  RESEARCHER [IDLE]: None
  SYNTHESIZER [IDLE]: None

ACTIVE PHEROMONES: 3
  [INIT] Build chat app (strength: 100%)
  [FOCUS] WebSocket security (strength: 65%)
  [FOCUS] message reliability (strength: 45%)

PHASE PROGRESS:
  Completed: 1
  In Progress: 1
  Pending: 3
```

## When to Use

- Check colony activity during execution
- See what Worker Ants are doing
- Check active pheromones
- Monitor phase progress

## Tips

- Use during phase execution to monitor
- Check pheromone strength before deciding whether to refresh
- See which Worker Ants are active

## Related Commands

```
/ant:phase     - Show current phase details
/ant:memory     - View learned patterns
/ant:focus     - Add focus pheromone
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    return await commands.status()
</script>
