<!-- Aether-managed: runtime spec at .aether/commands/reference-match.yaml. Synced by aether update. -->
---
name: ant-reference-match
description: "🔍 Match global references to a worker role, task, and optional output type"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether reference-match $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not invent reference matches by hand; use the runtime scorer.
- If docs and runtime disagree, runtime wins.
