<!-- Aether-managed: runtime spec at .aether/commands/run.yaml. Synced by aether update. -->
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

## What the user sees

The runtime narrates the whole run in the terminal as it works: an AUTOPILOT ENGAGED banner, a header and live worker lines for each phase's build, a PHASE ADVANCEMENT block with a momentum ticker between phases, a clearly framed pause block with the reason and next step whenever the run stops on purpose, and completion celebrations when every phase is done. Relay that output as-is — do not re-render, summarize over, or hand-draw any of those banners. `aether run --dry-run` previews the planned phases and lists every pause trigger.
