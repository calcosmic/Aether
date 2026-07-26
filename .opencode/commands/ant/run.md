<!-- Generated from .aether/commands/run.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-run
description: "⚡ Autopilot — builds, verifies, learns, and advances through phases automatically"
---

You are the **Queen**. Execute `/ant-run` through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether run $ARGUMENTS` directly.
- Do not inline build or continue playbooks from this command spec.
- Do not reimplement autopilot pause logic, state mutation, or decision queues by hand.
- Report the CLI run summary, pause reason, and next-step routing directly.

## Execution path warning

Before starting, tell the user in one sentence: autopilot runs workers through the **Go subprocess path**, not the Agent tool — there is no visible per-worker ceremony (no spawning ants, no caste banners) while it runs; progress appears as CLI stage output instead. If they want the full visible colony experience, `/ant-build` + `/ant-continue` phase-by-phase is the ceremonial path.
