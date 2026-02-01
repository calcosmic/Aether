---
name: ant:init
description: Initialize new project - Queen sets intention, colony creates phase structure
---

<objective>
Initialize a new project by emitting an init pheromone. The colony will create a structured phase plan based on the Queen's intention.

This is the first command to run when starting a new project.
</objective>

<reference>
# `/ant:init` - Usage

## Command

```
/ant:init "<your goal here>"
```

## Examples

```bash
/ant:init "Build a real-time chat application"
/ant:init "Add authentication system"
/ant:init "Create REST API with user management"
```

## What Happens

1. Queen sets intention via init pheromone
2. Colony mobilizes:
   - Mapper explores codebase
   - Planner creates phase structure
3. Phase plan is created and displayed
4. Next steps are clearly shown

## After Running

```
📋 NEXT STEPS:

  1. /ant:plan              - Review all phases in detail
  2. /ant:phase 1           - Review Phase 1 before starting
  3. /ant:focus <area>      - Guide colony attention (optional)

💡 RECOMMENDATION: Run /ant:plan to see the full roadmap

🔄 CONTEXT: This command is lightweight - safe to continue
```

## Tips

- Be specific about your goal
- Include what you're building, not how
- Colony will figure out the "how"
- Use /ant:plan to review before executing

## Related Commands

```
/ant:plan     - Review all phases
/ant:phase    - Review specific phase
/ant:focus    - Guide colony attention
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()
    goal = " ".join(args) if args else None

    if not goal:
        return "❌ Usage: /ant:init \"<goal>\"\n\nExample: /ant:init \"Build a real-time chat application\""

    commands.started = True
    return await commands.init(goal)
</script>
