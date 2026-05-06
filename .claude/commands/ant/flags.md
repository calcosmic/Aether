<!-- Generated from .aether/commands/flags.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-flags
description: "🚩 List project flags (blockers, issues, notes)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether flags $ARGUMENTS` directly.
- To resolve or acknowledge a flag, use the runtime commands `AETHER_OUTPUT_MODE=visual aether flag-resolve --id <id>` or `AETHER_OUTPUT_MODE=visual aether flag-acknowledge --id <id>`.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
- Report the CLI result directly.
