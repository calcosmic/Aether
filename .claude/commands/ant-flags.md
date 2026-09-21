<!-- Aether-managed: runtime spec at .aether/commands/flags.yaml. Synced by aether update. -->
---
name: ant-flags
description: "🚩 List project flags (blockers, issues, notes)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether flags $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- To resolve or acknowledge a flag, use the runtime commands `AETHER_OUTPUT_MODE=visual aether flag-resolve --id <id>` or `AETHER_OUTPUT_MODE=visual aether flag-acknowledge --id <id>`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
