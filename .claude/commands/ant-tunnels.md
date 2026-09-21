<!-- Aether-managed: runtime spec at .aether/commands/tunnels.yaml. Synced by aether update. -->
---
name: ant-tunnels
description: "🕳️ Explore tunnels (browse archived colonies, compare chambers)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether tunnels $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Runtime owns the restored views:
  - no arguments: chamber timeline
  - one chamber: detail and seal summary
  - two chambers: side-by-side comparison
  - `<chamber> --import-signals`: import pheromone signals from that chamber archive with chamber-prefixed IDs
- Do not inspect `.aether/chambers/`, read colony state, compare chamber files, import XML, or generate Next Up by hand from this wrapper.
