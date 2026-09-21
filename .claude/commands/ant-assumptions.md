<!-- Aether-managed: runtime spec at .aether/commands/assumptions.yaml. Synced by aether update. -->
---
name: ant-assumptions
description: "📐 Surface plan assumptions through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether assumptions-analyze $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not write `assumptions.json` or `pheromones.json` by hand from this command spec.
- Use `AETHER_OUTPUT_MODE=visual aether assumption-list` to inspect the current assumptions file. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- To validate one assumption after confirming evidence, execute `AETHER_OUTPUT_MODE=visual aether assumption-validate --id <id> --note "<evidence>"`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If the runtime reports unclear assumptions, surface them honestly instead of silently deciding for the user.
- If docs and runtime disagree, runtime wins.

**Next steps:**
- `/ant-build <phase>` — build with assumptions surfaced
- `/ant-discuss` — resolve any assumption that looks wrong
