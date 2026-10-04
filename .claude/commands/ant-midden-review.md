---
name: ant-midden-review
description: "🪵 Review the failure log — things that went wrong and have not been dealt with"
---
<!-- Aether-managed: runtime spec at .aether/commands/midden-review.yaml. Synced by aether update. -->

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether midden-review $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not write colony state files, session files, or pheromone files by hand from this command spec.
