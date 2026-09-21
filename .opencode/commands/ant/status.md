<!-- Aether-managed: runtime spec at .aether/commands/status.yaml. Synced by aether update. -->
---
name: ant-status
description: "Show the complete authoritative colony snapshot."
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS` directly. With no arguments, the default full view is used; pass `--compact` for its strict compact projection subset. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Treat the runtime output as the complete snapshot. Do not inspect host state, cost, files, or activity to supplement it.
- Status is the complete snapshot; `aether watch` is the event stream and does not replace status authority.
- If docs and runtime disagree, runtime wins.
