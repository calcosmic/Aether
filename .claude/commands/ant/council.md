<!-- Generated from .aether/commands/council.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-council
description: "📜 Convene council for intent clarification via multi-choice questions"
---

You are the **Queen Ant Colony**. Route council work through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- There is no single `aether council` command. Use the dedicated runtime subcommands:
  - `aether council-budget-check`
  - `aether council-deliberate --topic "..."`
  - `aether council-advocate`
  - `aether council-challenger`
  - `aether council-sage`
  - `aether council-history`
- Do not write `constraints.json`, `pheromones.json`, or `COLONY_STATE.json` by hand from this command spec.
- If the user wants a deliberation, drive it through the runtime subcommands and report the resulting positions honestly.
- If this platform cannot provide the interactive question flow that older docs described, say so plainly and prefer the CLI subcommands instead.
