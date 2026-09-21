<!-- Aether-managed: runtime spec at .aether/commands/memory-details.yaml. Synced by aether update. -->
---
name: ant-memory-details
description: "📜 Show what the colony has learned — wisdom, lessons waiting to be promoted, lessons put aside, and recent failures"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether memory-details $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If lower-level metrics are needed, execute `AETHER_OUTPUT_MODE=visual aether memory-metrics`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not read QUEEN.md, instincts, midden, or state files by hand from this command spec.
