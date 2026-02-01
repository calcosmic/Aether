---
name: ant:phase
description: Show current phase or specific phase details with state-aware prompts
---

<objective>
Display phase status with different output based on phase state (pending/in-progress/complete).
Shows tasks, Worker Ant activity, active pheromones, and next steps.
</objective>

<reference>
# `/ant:phase` - Usage

## Command

```
/ant:phase              # Show current phase
/ant:phase 1           # Show specific phase
```

## State-Aware Output

### Pending Phase
Shows tasks, milestones, and execution options.

### In-Progress Phase
Shows progress percentage, active Worker Ants, subagents, and active pheromones.

### Complete Phase
Shows summary, key learnings, issues resolved, and review prompt.

## After Running

```
📋 NEXT STEPS:

  1. /ant:execute 1         - Start executing this phase
  2. /ant:focus <area>      - Guide colony attention

💡 COLONY RECOMMENDATION:
   Consider focusing on: "WebSocket setup"

🔄 CONTEXT: This command is lightweight - safe to continue
```

## Related Commands

```
/ant:plan     - Show all phases
/ant:execute  - Execute a phase
/ant:review   - Review completed phase
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    phase_id = int(args[0]) if args and args[0].isdigit() else None
    return await commands.phase(phase_id)
</script>
