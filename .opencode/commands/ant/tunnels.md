<!-- Generated from .aether/commands/tunnels.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-tunnels
description: "🕳️ Explore tunnels (browse archived colonies, compare chambers)"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether tunnels $ARGUMENTS` directly.
- Runtime owns the restored views:
  - no arguments: chamber timeline
  - one chamber: detail and seal summary
  - two chambers: side-by-side comparison
  - `<chamber> --import-signals`: import pheromone signals from that chamber archive with chamber-prefixed IDs
- Do not inspect `.aether/chambers/`, read colony state, compare chamber files, import XML, or generate Next Up by hand from this wrapper.
- Report the CLI result directly.
