<!-- Aether-managed: runtime spec at .aether/commands/status.yaml. Synced by aether update. -->
---
name: ant-status
description: "Show the complete authoritative colony snapshot."
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS` directly. With no arguments, the default full view is used; pass `--compact` for its strict compact projection subset.
- Treat the runtime output as the complete snapshot. Do not inspect host state, cost, files, or activity to supplement it.
- Status is the complete snapshot; `aether watch` is the event stream and does not replace status authority.
- If docs and runtime disagree, runtime wins.
- Keep any wrapper summary to at most 2 short sentences.
- Relay the runtime's Next Up answer without adding a competing menu or interpretation.
