<!-- Generated from .aether/commands/patrol.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-patrol
description: "📊 Patrol the colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether patrol-check $ARGUMENTS` directly.
- The patrol-check command runs three health checks: JSON validity for COLONY_STATE.json, pheromones.json, and session.json; stale pheromone detection (signals referencing completed phases or zero strength); and interrupted build detection (uncommitted manifests or spawn trees).
- Display the structured health report with status per check (healthy/warning/error). For warnings and errors, include the specific details and remediation suggestions.
- Do not synthesize `completion-report.md` manually or mutate colony state from this command spec.
- If the runtime reports missing colony state, test failures, or audit warnings, relay that exact output.
- If docs and runtime disagree, runtime wins.
