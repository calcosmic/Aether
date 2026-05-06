<!-- Generated from .aether/commands/flag.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-flag
description: "🚩 Create a project-specific flag (blocker, issue, or note)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether flag $ARGUMENTS` directly.
- If `$ARGUMENTS` is empty, show `Usage: /ant-flag "<description>" [--type blocker|issue|note] [--phase N]`.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
- Report the CLI result directly.
