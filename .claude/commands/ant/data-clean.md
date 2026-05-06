<!-- Generated from .aether/commands/data-clean.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-data-clean
description: "🧹 Scan and remove test artifacts from colony data files"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether data-clean $ARGUMENTS` directly.
- If the user wants destructive cleanup, require the runtime-supported confirmation flag in `$ARGUMENTS`.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
- Report the CLI result directly.
