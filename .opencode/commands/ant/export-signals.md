<!-- Generated from .aether/commands/export-signals.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-export-signals
description: "📤 Export colony pheromone signals to portable XML format"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether export-signals $ARGUMENTS` directly.
- If `$ARGUMENTS` is empty, the runtime writes XML to stdout; pass `--output <path>` to write a file.
- Do not read `.aether/data/COLONY_STATE.json` or pheromone files by hand from this command spec.
- Report the CLI result directly.
