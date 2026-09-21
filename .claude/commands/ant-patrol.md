<!-- Aether-managed: runtime spec at .aether/commands/patrol.yaml. Synced by aether update. -->
---
name: ant-patrol
description: "📊 Patrol the colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether patrol-check $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- The patrol-check command runs three health checks: JSON validity for COLONY_STATE.json, pheromones.json, and session.json; stale pheromone detection (signals referencing completed phases or zero strength); and interrupted build detection (uncommitted manifests or spawn trees).
- Display the structured health report with status per check (healthy/warning/error). For warnings and errors, include the specific details and remediation suggestions.
- As part of interrupted-build detection, also run `aether spawn-orphans` to list any spawned helpers that have gone quiet past the configured reap threshold with no completion reported. If any are found, report them plainly and mention that `aether spawn-orphans --clear` marks them abandoned and frees the budget slots they are holding.
- Do not synthesize `completion-report.md` manually or mutate colony state from this command spec.
- If the runtime reports missing colony state, test failures, or audit warnings, relay that exact output.
- If docs and runtime disagree, runtime wins.
