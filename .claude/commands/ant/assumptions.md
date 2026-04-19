<!-- Generated from .aether/commands/assumptions.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:assumptions
description: "🧠🐜📐🐜🧠 Surface plan assumptions through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether assumptions-analyze $ARGUMENTS` directly.
- Do not write `assumptions.json` or `pheromones.json` by hand from this command spec.
- Use `AETHER_OUTPUT_MODE=visual aether assumption-list` to inspect the current assumptions file.
- To validate one assumption after confirming evidence, execute `AETHER_OUTPUT_MODE=visual aether assumption-validate --id <id> --note "<evidence>"`.
- If the runtime reports unclear assumptions, surface them honestly instead of silently deciding for the user.
- If docs and runtime disagree, runtime wins.
