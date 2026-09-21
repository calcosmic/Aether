<!-- Aether-managed: runtime spec at .aether/commands/flag.yaml. Synced by aether update. -->
---
name: ant-flag
description: "🚩 Create a project-specific flag (blocker, issue, or note)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether flag $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If `$ARGUMENTS` is empty, show `Usage: /ant-flag "<description>" [--type blocker|issue|note] [--phase N]`.
- Do not read `.aether/data/COLONY_STATE.json`, run `jq`, or generate Next Up by hand.
