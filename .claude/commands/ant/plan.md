<!-- Generated from .aether/commands/plan.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-plan
description: "📋 Generate or review a project plan through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS` directly.
- Do not write watch files, planning artifacts, or `COLONY_STATE.json` by hand from this command spec.
- If the runtime says no colony exists, the colony is sealed, or a survey/plan issue blocks progress, relay that exact guidance.
- If docs and runtime disagree, runtime wins.
