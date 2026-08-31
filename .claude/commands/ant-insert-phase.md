<!-- Aether-managed: runtime spec at .aether/commands/insert-phase.yaml. Synced by aether update. -->
---
name: ant-insert-phase
description: "➕ Insert a corrective phase into the active plan"
---

Use the Go `aether` CLI as the source of truth.

- Simplest usage: `aether insert-phase "problem to stabilise"`.
- For non-interactive automation: `aether insert-phase --after 2 --name "Stabilize login retries" --description "login retries lose state" --constraints "do not change the provider"`.
- Execute `AETHER_OUTPUT_MODE=visual aether insert-phase $ARGUMENTS` directly.
- Let the Go command resolve phase position, name, and description; do not derive those values in this wrapper.
- Do not read `.aether/data/COLONY_STATE.json`, rewrite plan phases, or generate Next Up by hand.
- Report the CLI result directly.
