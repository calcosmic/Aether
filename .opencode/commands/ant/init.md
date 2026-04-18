<!-- Generated from .aether/commands/init.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:init
description: "Initialize Aether colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- If `$ARGUMENTS` is empty, show `Usage: /ant:init "<your goal here>"`.
- Otherwise execute `AETHER_OUTPUT_MODE=visual aether init "$ARGUMENTS"` directly.
- Do not write `.aether/QUEEN.md`, `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand from this command spec.
- If setup is missing, relay the runtime guidance exactly.
- If docs and runtime disagree, runtime wins.
- Keep any wrapper summary to at most 2 short sentences.
- Do not add extra option menus, recovery branches, or manual state-mutation advice unless the runtime itself explicitly asks for them.
