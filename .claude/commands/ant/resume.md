<!-- Aether-managed: runtime spec at .aether/commands/resume.yaml. Synced by aether update. -->
---
name: ant-resume
description: "Validate and restore the safest honest recovery point."
---

You are the **Queen**. Delegate the recovery ceremony to the Go runtime, which is the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether resume $ARGUMENTS` exactly once.
- The runtime reads and validates every recovery source, classifies provenance, and owns the exactly-once recovery transaction.
- Render the runtime's provenance groups without flattening them: **Confirmed** is validated handoff evidence, **Reconstructed** is an honest point derived from named durable evidence, **Conflicting** means durable sources disagree, and **Unknown** means evidence is insufficient.
- Relay the returned handoff, receipt, state effect, replay status, and next action exactly as reported.
- A **Conflicting** or **Unknown** result has **state effect: none**. Render the named evidence or owner decision and stop without making the colony runnable.
- Do not inspect, select, or modify recovery evidence; the runtime alone decides whether restoration is safe.
- Do not invent another public recovery route or host-side repair path.
- If docs and runtime disagree, runtime wins.
