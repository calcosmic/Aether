<!-- Generated from .aether/commands/discuss.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-discuss
description: "💬 Capture clarifications before planning through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether discuss $ARGUMENTS` directly.
- Do not write `pending-decisions.json`, `pheromones.json`, or `COLONY_STATE.json` by hand from this command spec.
- If the runtime returns clarification questions, present them honestly instead of inventing answers on the user's behalf.
- To persist an answer, execute `AETHER_OUTPUT_MODE=visual aether discuss --resolve <id> --answer "<choice>"`.
- If the runtime reports `discussion_status: settled`, route wrapper users back to `/ant-plan`; direct CLI users can run `aether plan`.
- Use `/ant-council` only when the user wants multi-position deliberation; `/ant-discuss` is the lightweight pre-plan clarification gate.
- If docs and runtime disagree, runtime wins.
