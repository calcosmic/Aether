<!-- Aether-managed: runtime spec at .aether/commands/data-clean.yaml. Synced by aether update. -->
---
name: ant-data-clean
description: "🧹 Scan and remove test artifacts from colony data files"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether data-clean $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If the user wants destructive cleanup, require the runtime-supported confirmation flag in `$ARGUMENTS`.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
