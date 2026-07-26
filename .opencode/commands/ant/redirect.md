<!-- Aether-managed: runtime spec at .aether/commands/redirect.yaml. Synced by aether update. -->
---
name: ant-redirect
description: "🚫 Emit a REDIRECT pheromone through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether redirect "$ARGUMENTS"` directly.
- Do not append to `constraints.json` or rewrite pheromone state by hand.
- If `$ARGUMENTS` is empty, show `Usage: /ant-redirect <avoid-pattern>`.
- Report the CLI result directly.

**Next steps:**
- `/ant-pheromones` — see all active signals
- `/ant-build <phase>` — builders now avoid the redirected pattern
