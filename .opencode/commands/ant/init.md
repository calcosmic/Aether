<!-- Generated from .aether/commands/init.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-init
description: "🥚 Initialize Aether colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth, but do not skip the init
foundation pass.

- If `$ARGUMENTS` is empty, show `Usage: /ant-init "<your goal here>"`.
- First run `AETHER_OUTPUT_MODE=json aether init-research --goal "$ARGUMENTS" --target .`.
- Read at most a few top-level manifests or README files only if needed to ground the approval prompt.
- If the repo is non-trivial and the platform can spawn helpers, use Scout + Architect in read-only mode for a short foundation pass. Keep it bounded.
- Present an approval summary covering: goal, detected stack, likely impact areas, and key risks.
- Ask with 3 options: proceed, revise goal, cancel.
- Only after approval execute `AETHER_OUTPUT_MODE=visual aether init "$ARGUMENTS"`.
- Do not write `.aether/QUEEN.md`, `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand from this command spec.
- If setup is missing, relay the runtime guidance exactly.
- If docs and runtime disagree, runtime wins.
