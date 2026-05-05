<!-- Generated from .aether/commands/swarm.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-swarm
description: "🔥 Real-time colony swarm display + stubborn bug destroyer"
---

Use the Go `aether` CLI as the source of truth.

- For live worker visibility, execute `AETHER_OUTPUT_MODE=visual aether swarm --watch`.
- To launch the stubborn bug-destroyer flow, execute `AETHER_OUTPUT_MODE=visual aether swarm "$ARGUMENTS"` directly.
- The runtime owns the investigate -> fix -> verify worker waves. Do not hand-edit `.aether/data/` or manually reconstruct swarm artifacts.
- If the user provides no problem description, prefer `aether swarm --watch`.
- When running inside an agent-delegate session (Claude Code or OpenCode), the runtime returns a plan manifest with `dispatch_mode: agent-delegate` instead of spawning subprocess workers. The wrapper should read the manifest and dispatch workers directly.
- If docs and runtime disagree, runtime wins.
