<!-- Generated from .aether/commands/help.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:help
description: "🐜 Show the current Aether command surface"
---

Use the runtime CLI as the source of truth.

- Execute `aether --help` directly.
- If the user asked about a specific command, prefer `aether <command> --help`.
- Do not describe shell-managed watch sessions or direct `constraints.json` editing as the authoritative path.
- Treat `/ant:resume` and `/ant:resume-colony` as the same runtime alias.
- If the user only wants a read-only overview, prefer `aether resume-dashboard`.
- If docs and runtime disagree, runtime wins.
