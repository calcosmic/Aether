---
name: ant:plan
description: Show all phases with tasks, milestones, and status
---

<objective>
Display the complete phase plan showing all phases, their tasks, milestones, and current status.
</objective>

<reference>
# `/ant:plan` - Usage

## Command

```
/ant:plan
```

## What It Shows

- All phases created by the colony
- Tasks in each phase
- Milestones to achieve
- Current status of each phase

## Output Example

```
PHASE 1: Foundation [PENDING]
  Tasks: 5
  ⏳ Setup project structure
  ⏳ Configure development environment
  ⏳ Initialize database schema
  ⏳ Setup WebSocket server
  ⏳ Implement basic message routing
  Milestones:
    • WebSocket server running
    • Database connected

PHASE 2: Real-time Communication [PENDING]
  Tasks: 8
  ⏳ Implement WebSocket connection handling
  ⏳ Create message queue system
  ...
```

## After Running

```
📋 NEXT STEPS:

  1. /ant:phase 1           - Review Phase 1 details
  2. /ant:execute 1         - Start executing Phase 1
  3. /ant:focus <area>      - Add focus guidance (optional)

💡 RECOMMENDATION: Review Phase 1 with /ant:phase 1 before executing

🔄 CONTEXT: This command is lightweight - safe to continue
```

## Related Commands

```
/ant:init     - Initialize new project
/ant:phase    - Review specific phase
/ant:execute  - Start executing a phase
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    return await commands.plan()
</script>
