<!-- Generated from .aether/commands/profile.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-profile
description: "🧠 Inspect or refresh the behavioral profile through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether profile-read $ARGUMENTS` directly.
- Do not write `profile.json`, `behavior-observations.jsonl`, or `QUEEN.md` by hand from this command spec.
- If the user wants the latest observations consolidated first, execute `AETHER_OUTPUT_MODE=visual aether profile-update`.
- To record a new signal, execute `AETHER_OUTPUT_MODE=visual aether behavior-observe --dimension <name> --signal "<signal>" --strength <0-1> --evidence "<evidence>"`.
- If docs and runtime disagree, runtime wins.
