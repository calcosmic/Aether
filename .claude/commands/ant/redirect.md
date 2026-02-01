---
name: ant:redirect
description: Emit redirect pheromone - warn colony away from approach or pattern
---

<objective>
Emit a redirect pheromone (strong repel signal) to warn the colony away from a specific approach, pattern, or technology.
</objective>

<reference>
# `/ant:redirect` - Usage

## Command

```
/ant:redirect "<pattern>"
```

## Examples

```bash
/ant:redirect "Don't use string concatenation for SQL"
/ant:redirect "Avoid callbacks, use async/await"
/ant:redirect "Don't use MongoDB for this"
```

## What Happens

- Redirect pheromone emitted (strength 0.7, lasts 24 hours)
- Executor avoids this pattern
- Planner avoids in future plans
- After 3 occurrences: becomes permanent constraint

## Output

```
🐜 Queen Ant Colony - Redirect Pheromone Emitted

"Don't use string concatenation for SQL"

Signal: REDIRECT (strength: 0.7)
Duration: 24 hours

COLONY RESPONDING:
  ✓ Executor avoiding string concatenation
  ✓ Planner using parameterized queries
  ✓ Verifier validating SQL before execution

OCCURRENCES: 1/3 (will become constraint after 3)
```

## Learning

- Occurrence 1: Logged in ERROR_LEDGER
- Occurrence 2: Pattern detected
- Occurrence 3: **FLAGGED_ISSUE created, constraint created**

After 3 occurrences, the pattern becomes a permanent constraint that validates BEFORE execution.

## Related Commands

```
/ant:focus     - Guide colony attention
/ant:memory     - View learned patterns
/ant:status     - Check colony response
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    if not args:
        return "❌ Usage: /ant:redirect \"<pattern>\"\n\nExample: /ant:redirect \"Don't use string concatenation for SQL\""

    pattern = " ".join(args)
    return await commands.redirect(pattern)
</script>
