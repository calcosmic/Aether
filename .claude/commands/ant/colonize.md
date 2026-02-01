---
name: ant:colonize
description: Colonize codebase - analyze existing code before starting project
---

<objective>
Analyze existing codebase to understand tech stack, architecture patterns, code conventions, and dependencies. Colony uses this to generate code that matches your existing patterns.
</objective>

<reference>
# `/ant:colonize` - Usage

## Command

```
/ant:colonize
```

## What It Does

Spawns parallel agents to analyze your codebase:
- **Mapper**: Explores codebase structure
- **Researcher**: Identifies technologies
- **Planner**: Analyzes architecture
- **Synthesizer**: Extracts patterns
- **Verifier**: Finds issues

## Output

```
🐜 Queen Ant Colony - Colonize Codebase

MAPPING IN PROGRESS...

Colony is analyzing your codebase in parallel:

  [1/5] Mapper: Exploring codebase structure
  [2/5] Researcher: Identifying technologies
  [3/5] Planner: Analyzing architecture
  [4/5] Synthesizer: Extracting patterns
  [5/5] Verifier: Finding issues

SCAN RESULTS:

TECHNOLOGIES DETECTED:
  • Python 3.10+
  • FastAPI framework
  • PostgreSQL database
  • React frontend
  • Redis caching

ARCHITECTURE PATTERNS:
  • RESTful API structure
  • Service layer pattern
  • Repository pattern
  • Dependency injection

CODE CONVENTIONS:
  • snake_case for files
  • PascalCase for classes
  • SPACING_2 for constants

DEPENDENCIES FOUND:
  • fastapi
  • sqlalchemy
  • pydantic
  • pytest

✅ CODEBASE COLONIZED

Colony now understands:
  • Your tech stack and patterns
  • Your coding conventions
  • Your architecture

This context will be used for:
  • Phase planning (tasks match your patterns)
  • Code generation (follows your conventions)
  • Integration (matches your architecture)
```

## When to Use

- Before starting a new project in existing codebase
- When you want new code to match existing patterns
- When colony needs to understand your codebase style

## After Running

```
📋 NEXT STEPS:
  1. /ant:init "<your goal>"  - Start your new project
  2. /ant:plan               - Review phases
  3. /ant:phase 1           - Start first phase

💡 RECOMMENDATION:
   Colony is now ready to build that matches your codebase style.
   Your new code will seamlessly integrate with existing patterns.

🔄 CONTEXT: Lightweight - safe to continue
```

## Benefits

- New code matches your existing architecture
- Follows your coding conventions
- Integrates seamlessly with existing code
- Colony understands your patterns

## Related Commands

```
/ant:init     - Start new project
/ant:plan     - Review phases
/ant:status   - Check colony status
```
</reference>

<script>
from .aether.interactive_commands import get_commands

async def main(args):
    commands = get_commands()

    return await commands.colonize()
</script>
