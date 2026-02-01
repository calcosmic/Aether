---
name: ant:execute
description: Execute a phase with pure emergence - colony self-organizes and completes tasks
---

<objective>
Execute a phase with pure emergence. Worker Ants self-organize, spawn subagents, and complete tasks. Progress shown in real-time.
</objective>

<reference>
# `/ant:execute` - Usage

## Command

```
/ant:execute <phase_id>
```

## Examples

```bash
/ant:execute 1    # Execute Phase 1
/ant:execute 2    # Execute Phase 2
```

## What Happens

1. Phase execution begins
2. Worker Ants self-organize
3. Subagents spawn for tasks
4. Tasks complete one by one
5. Phase completes
6. Summary shown

## During Execution

- Tasks complete sequentially
- Worker Ants spawn subagents
- Pheromones guide in real-time
- Progress updates shown

## After Completion

```
PHASE SUMMARY:
  ✓ 5/5 tasks completed
  ✓ 2 milestones reached
  ✓ 3 issues found and fixed

📋 NEXT STEPS:
  1. /ant:review 1          - Review completed work
  2. /ant:phase continue    - Continue to next phase

💡 COLONY RECOMMENDATION:
   Review work before continuing.

🔄 CONTEXT: REFRESH RECOMMENDED
   Phase execution used significant context.
   Refresh Claude with /ant:review 1 before continuing.
```

## Tips

- Consider adding /ant:focus before executing to guide colony
- Use /ant:status during execution to check progress
- Refresh context after completion

## Related Commands

```
/ant:phase    - Review phase before executing
/ant:focus    - Guide colony before execution
/ant:review   - Review completed work
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    if not args or not args[0].isdigit():
        return "❌ Usage: /ant:execute <phase_id>\n\nExample: /ant:execute 1"

    phase_id = int(args[0])
    return await commands.execute(phase_id)
</script>
