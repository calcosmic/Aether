<!-- Aether-managed: runtime spec at .aether/commands/history.yaml. Synced by aether update. -->
---
name: ant-history
description: "📜 Show colony event history"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether history $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not read `.aether/data/COLONY_STATE.json` or parse event strings by hand.
