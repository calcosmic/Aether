<!-- Generated from .aether/commands/resume.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-resume
description: "💾 Resume previous session through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether resume` directly.
- `resume` is currently an alias of `resume-colony`, not a distinct recovery flow.
- For a read-only overview before resuming, use `AETHER_OUTPUT_MODE=visual aether resume-dashboard`.
- Do not reconstruct state manually from `session.json`, `COLONY_STATE.json`, or `.aether/HANDOFF.md`.
- If docs and runtime disagree, runtime wins.
