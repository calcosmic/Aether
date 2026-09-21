<!-- Aether-managed: runtime spec at .aether/commands/profile.yaml. Synced by aether update. -->
---
name: ant-profile
description: "🧠 Inspect or refresh the behavioral profile through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether profile-read $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not write `profile.json`, `behavior-observations.jsonl`, or `QUEEN.md` by hand from this command spec.
- If the user wants the latest observations consolidated first, execute `AETHER_OUTPUT_MODE=visual aether profile-update`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- To record a new signal, execute `AETHER_OUTPUT_MODE=visual aether behavior-observe --dimension <name> --signal "<signal>" --strength <0-1> --evidence "<evidence>"`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If docs and runtime disagree, runtime wins.
