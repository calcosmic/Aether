<!-- Generated from .aether/commands/interpret.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-interpret
description: "🔍 Read-only interpretation of dream sessions against the current codebase"
---

This command is read-only.

- Review the selected dream session from `.aether/dreams/` and investigate its claims against the current codebase with concrete file and line evidence.
- If no dream sessions exist, tell the user to run `/ant-dream` first.
- Do not modify code or write `COLONY_STATE.json`, `constraints.json`, `pheromones.json`, or `TO-DOS.md`.
- Do not auto-inject pheromones or create action items. If the user wants to act, recommend explicit follow-up commands such as `aether focus`, `aether redirect`, `aether feedback`, `aether plan`, or `aether build`.
- If docs and runtime disagree, runtime wins where a CLI command exists.
