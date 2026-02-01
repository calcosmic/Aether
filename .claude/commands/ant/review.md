---
name: ant:review
description: Review completed phase - see what was built, key learnings, issues resolved
---

<objective>
Review a completed phase to see what was built, files created/modified, features implemented, key learnings, and issues resolved.
</objective>

<reference>
# `/ant:review` - Usage

## Command

```
/ant:review <phase_id>
```

## Examples

```bash
/ant:review 1    # Review Phase 1
```

## What It Shows

- Files created/modified
- Features implemented
- Key learnings
- Issues resolved
- Queen feedback options

## Output Example

```
PHASE 1: Foundation - COMPLETE

WHAT WAS BUILT:
  Files created/modified:
    • project/setup.py
    • project/config.py
    • database/schema.sql
    • websocket/server.py
    • routing/handlers.py

FEATURES IMPLEMENTED:
  ✓ Project structure with modular architecture
  ✓ Development environment configuration
  ✓ PostgreSQL database with connection pooling
  ✓ WebSocket server with connection pooling
  ✓ Basic message routing between clients

KEY LEARNINGS:
  • Connection pooling reduces overhead by 40%
  • Modular structure enables parallel development

ISSUES RESOLVED:
  • WebSocket timeout issue (fixed with heartbeat)
  • Database connection leak (fixed with pool limits)
```

## After Review

```
📋 NEXT STEPS:
  1. /ant:phase continue    - Continue to Phase 2
  2. /ant:focus <area>      - Set focus for next phase

💡 COLONY RECOMMENDATION:
   Ready for next phase.

🔄 CONTEXT: REFRESH RECOMMENDED
   This is a clean checkpoint - safe to refresh Claude
   and continue with /ant:phase continue
```

## When to Use

- After phase execution completes
- Before starting next phase
- To see what was actually built

## Tips

- This is a good checkpoint to refresh context
- Provide feedback via /ant:feedback
- Use /ant:focus to set priorities for next phase

## Related Commands

```
/ant:execute  - Execute a phase
/ant:phase    - Show phase status
/ant:focus    - Set focus for next phase
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    if not args or not args[0].isdigit():
        return "❌ Usage: /ant:review <phase_id>\n\nExample: /ant:review 1"

    phase_id = int(args[0])
    return await commands.review(phase_id)
</script>
