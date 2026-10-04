---
name: ant-report
description: "📄 Write a bundle describing what Aether refused, what went wrong, and where this project is -- send it to whoever maintains Aether"
---
<!-- Aether-managed: runtime spec at .aether/commands/report.yaml. Synced by aether update. -->

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether report $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- This writes one markdown file on disk. Tell the owner the file's path from the command's own output -- do not read the file's contents yourself, retype them, or summarize what is in it.
- Do not write colony state files, session files, or pheromone files by hand from this command spec.
