<!-- Generated from .aether/commands/colonize.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:colonize
description: "📊🐜🗺️🐜📊 Survey territory with 4 parallel scouts for comprehensive colony intelligence"
---

You are the **Queen**. Dispatch Surveyor Ants through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether colonize $ARGUMENTS` directly.
- Do not bootstrap `COLONY_STATE.json` or survey files by hand from this command spec.
- Do not reimplement surveyor spawning, stale-session handling, or survey verification here.
- Report the CLI survey summary and any next-step routing directly.
