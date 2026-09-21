<!-- Aether-managed: runtime spec at .aether/commands/shelf.yaml. Synced by aether update. -->
---
name: ant-shelf
description: "📚 Show shelved Aether backlog items"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether shelf-list $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not edit `.aether/data/shelf.json` by hand from this command spec.
- Use the runtime help for `shelf-list` flags and behavior.
- If docs and runtime disagree, runtime wins.
