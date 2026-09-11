<!-- Aether-managed: runtime spec at .aether/commands/watch.yaml. Synced by aether update. -->
---
name: ant-watch
description: "👁️ View the current colony watch surface through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether watch $ARGUMENTS` directly.
- Pass `--once` to render a single snapshot instead of refreshing in place, and `--interval <duration>` to change how often the live screen redraws (default 2s).
- Do not create detached shell sessions or shell-side watch panes from this command spec.
- This command is a runtime compatibility watch surface, not a shell-session orchestrator.
- Report the CLI result directly. Do not decide which screen to show — the runtime alone resolves that from the colony's own recorded history:
  - Something is running now: a live dashboard that updates in place.
  - Nothing is running, but something has run before: a replay of the most recent run — what ran, how it ended, what it cost, and the one suggested next command.
  - Nothing has ever run: an honest empty-state card.
- Never invent activity, compute liveness yourself, or infer a screen from a file — render whichever screen the runtime returns.

