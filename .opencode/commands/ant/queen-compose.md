<!-- Generated from .aether/commands/queen-compose.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-queen-compose
description: "Create or improve repo-local QUEEN.md project brain"
---

You are the **Queen**. Create or improve the repo-local `.aether/QUEEN.md` project brain through the Aether runtime.

The user's input is: `$ARGUMENTS`

## Instructions

Use the Go `aether` CLI as the source of truth. Do not edit `.aether/QUEEN.md` by hand from this wrapper, and do not spawn workers for this command.

This wrapper must behave the same in Claude Code and OpenCode. Use the current host platform's normal user-question flow if clarification is needed. Do not assume Codex, and do not use Codex-specific agent names or subagent tooling.

### If `$ARGUMENTS` is provided

Run:

```bash
AETHER_OUTPUT_MODE=visual aether queen-compose $ARGUMENTS
```

Then summarize the generated or updated local QUEEN.md path and the sections the runtime wrote.

### If `$ARGUMENTS` is empty

Ask the user one compact batch of questions:

1. What do you want this repo to be? What is the project?
2. What are you developing right now?
3. Who is this for?
4. What tech stack, tools, frameworks, or services matter?
5. What build, lint, test, run, or release commands should agents know?
6. How do you want agents to speak to you?
7. Do you want short plain-English / for-dummies explanations?
8. What should agents prioritize in this repo?
9. What should agents avoid or treat as constraints?
10. What does good verification look like here?

After the user answers, run `aether queen-compose` with flags for the answers you received. Omit flags for answers the user left blank. Pass `--plain-english=false` only when the user explicitly says they do not want beginner framing.

Do not synthesize or write the markdown yourself. The runtime owns the file update and preservation rules.
