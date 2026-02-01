---
name: ant:memory
description: View Queen Ant Colony memory - learned preferences, pheromone patterns
---

<objective>
Display colony's learned patterns from pheromone signals including focus topics, avoid patterns, and feedback categories.
</objective>

<reference>
# `/ant:memory` - Usage

## Command

```
/ant:memory
```

## What It Shows

- Learned focus topics (what Queen prioritizes)
- Learned avoid patterns (what Queen redirects away from)
- Feedback categories and counts

## Output Example

```
LEARNED PREFERENCES:

FOCUS TOPICS:
  WebSocket security (3 occurrences)
  message reliability (2 occurrences)
  authentication (1 occurrence)

AVOID PATTERNS:
  string concatenation for SQL (2 occurrences) → One more becomes constraint

FEEDBACK CATEGORIES:
  Quality: 12 positive, 3 negative
  Speed: 5 "too slow", 8 "good pace"
  Direction: 2 "wrong approach" corrections
```

## How Learning Works

- **3+ focuses** on same topic → Preference learned
- **3+ redirects** on same pattern → Constraint created
- **5+ positive feedback** → Best practice established

## Output

```
📋 NEXT STEPS:
  1. /ant:status            - Check colony status
  2. /ant:focus <area>      - Add focus (teaches preferences)
  3. /ant:redirect <pattern> - Avoid pattern (teaches constraints)

💡 MEMORY TIP:
  Colony learns from your signals over time.
```

## Related Commands

```
/ant:status    - Check colony status
/ant:focus     - Add focus pheromone
/ant:redirect  - Add redirect pheromone
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    if not commands.started:
        return "❌ No project initialized. Run /ant:init <goal> first."

    return await commands.memory()
</script>
