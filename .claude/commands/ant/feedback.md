---
name: ant:feedback
description: Emit feedback pheromone - provide guidance to colony
---

<objective>
Emit a feedback pheromone to guide colony behavior based on Queen's observations, preferences, or corrections.
</objective>

<reference>
# `/ant:feedback` - Usage

## Command

```
/ant:feedback "<message>"
```

## Examples

```bash
/ant:feedback "Great progress on WebSocket layer"
/ant:feedback "Too slow, need to speed up"
/ant:feedback "This approach is wrong"
/ant:feedback "Need more test coverage"
```

## What Happens

- Feedback pheromone emitted
- Categorized (quality/speed/direction/positive)
- Colony adjusts behavior based on category
- Pattern recorded for reuse

## Output

```
🐜 Queen Ant Colony - Feedback Recorded

"Great progress on WebSocket layer"

Category: quality (positive)
Strength: 0.5

COLONY RESPONDING:
  ✓ Synthesizer recording positive pattern
  ✓ Executor continuing current approach

📋 NEXT STEPS:
  1. /ant:memory            - View learned patterns
  2. /ant:status            - Check colony status
```

## Categories

| Feedback | Category | Colony Response |
|----------|----------|-----------------|
| "Great work", "Perfect" | Positive | Pattern reinforced |
| "Too many bugs", "Quality issues" | Quality | Verifier intensifies testing |
| "Too slow", "Speed up" | Speed | Executor parallelizes more |
| "Wrong approach" | Direction | Planner pivots approach |

## Tips

- Be specific about what's good/bad
- Colony learns from patterns over time
- Positive feedback reinforces patterns
- Negative feedback triggers adjustments

## Related Commands

```
/ant:focus     - Guide colony attention
/ant:redirect  - Warn colony away from pattern
/ant:memory     - View learned patterns
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    if not args:
        return "❌ Usage: /ant:feedback \"<message>\"\n\nExample: /ant:feedback \"Great progress\""

    message = " ".join(args)
    return await commands.feedback(message)
</script>
