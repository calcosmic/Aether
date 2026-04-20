<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:build
description: "Build a phase — Queen dispatches workers, colony self-organizes"
---

You are the **Queen**. The colony is building.

The phase to build is: `$ARGUMENTS`

## Colony Context

Before dispatching, take a moment to ground yourself in the colony's current state:

1. Run `AETHER_OUTPUT_MODE=visual aether status` to see where the colony stands
2. Note any active pheromone signals — they steer worker behavior during the build
3. Understand what previous phases accomplished so you can frame this phase's purpose

This context helps you narrate the build with colony awareness, not just pass through CLI output.

## Dispatch

Execute the build through the runtime. Use the Go `aether` CLI as the source of truth.

```
AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS
```

The runtime owns all state transitions, worker dispatch, and verification. Your role is to
frame what happens with colony identity and provide the human layer around the CLI output.

## After the Build

Once the runtime completes its dispatch:

1. **Summarize what happened** in colony terms (which workers were dispatched, what they're tasked with)
2. **Note any signals** that are particularly relevant to this phase's work
3. **Guide the user** on what to do next — typically waiting for work to complete, then running `/ant:continue`

## Guardrails

- Do NOT load playbooks or reimplement build orchestration
- Do NOT read or write colony state files by hand
- Do NOT mutate COLONY_STATE.json, session.json, or pheromone files
- Do NOT add extra option menus or recovery advice unless the runtime explicitly asks
- If docs and runtime disagree, runtime wins
- If `$ARGUMENTS` is empty, show: `Usage: /ant:build <phase_number>`
