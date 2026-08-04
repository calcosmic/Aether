<!-- Aether-managed: runtime spec at .aether/commands/feedback.yaml. Synced by aether update. -->
---
name: ant-feedback
description: "💬 Emit FEEDBACK through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether feedback "$ARGUMENTS"` directly.
- Do not append instinct records to `COLONY_STATE.json` or rewrite signal files by hand.
- If `$ARGUMENTS` is empty, show `Usage: /ant-feedback <note>`.
- Report the CLI result directly.

**Next steps:**
- `/ant-pheromones` — see all active signals
- `/ant-continue` — verification applies the adjustment
