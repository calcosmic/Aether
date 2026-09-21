<!-- Aether-managed: runtime spec at .aether/commands/insert-phase.yaml. Synced by aether update. -->
---
name: ant-insert-phase
description: "➕ Insert a corrective phase into the active plan"
---

Use the Go `aether` CLI as the source of truth.

- Simplest usage: `aether insert-phase "problem to stabilise"`.
- For non-interactive automation: `aether insert-phase --after 2 --name "Stabilize login retries" --description "login retries lose state" --constraints "do not change the provider"`.
- Execute `AETHER_OUTPUT_MODE=visual aether insert-phase $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Let the Go command resolve phase position, name, and description; do not derive those values in this wrapper.
- Do not read `.aether/data/COLONY_STATE.json`, rewrite plan phases, or generate Next Up by hand.
