<!-- Aether-managed: runtime spec at .aether/commands/verify-castes.yaml. Synced by aether update. -->
---
name: ant-verify-castes
description: "✓ Verify colony caste assignments and system status"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether verify-castes $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not reconstruct caste tables, read `.aether/data/COLONY_STATE.json`, or generate Next Up by hand.
