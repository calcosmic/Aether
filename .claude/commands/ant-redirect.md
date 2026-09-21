<!-- Aether-managed: runtime spec at .aether/commands/redirect.yaml. Synced by aether update. -->
---
name: ant-redirect
description: "🚫 Emit a REDIRECT pheromone through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether redirect "$ARGUMENTS"` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not append to `constraints.json` or rewrite pheromone state by hand.
- If `$ARGUMENTS` is empty, show `Usage: /ant-redirect <avoid-pattern>`.

**Next steps:**
- `/ant-pheromones` — see all active signals
- `/ant-build <phase>` — builders now avoid the redirected pattern
