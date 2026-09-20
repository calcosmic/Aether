# 154-01 Summary: Colony Assets — Agent Definitions and Prompts

## What Was Done

- Expanded `AgentSchema.role` enum from 8 to 27 castes.
- Created 27 `colony/agents/{role}.yaml` files with metadata (id, role, prompt_file, allowed_tools, tier, model, color, description).
- Created 27 `colony/prompts/{role}.md` files by extracting Markdown bodies (after frontmatter) from `.claude/agents/ant/aether-{role}.md`.
- Created/updated 27 `control-ts/tests/fixtures/agents/{role}.yaml` files.

## Test Results

```
Test Files  1 passed (1)
     Tests  5 passed (5)
```

All schema tests pass, including file-existence checks.

## Files Modified

- `control-ts/src/schemas/agent.schema.ts`
- `control-ts/tests/fixtures/agents/*.yaml` (27 files)

## Files Created

- `colony/agents/*.yaml` (27 files)
- `colony/prompts/*.md` (27 files)
