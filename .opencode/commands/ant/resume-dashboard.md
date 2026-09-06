<!-- Aether-managed: runtime spec at .aether/commands/resume-dashboard.yaml. Synced by aether update. -->
---
name: ant-resume-dashboard
description: "Show the read-only resume dashboard."
---

Use the Go `aether` CLI as the source of truth.

Execute `AETHER_OUTPUT_MODE=visual aether resume-dashboard $ARGUMENTS` exactly once and return its stdout unchanged. The dashboard is read-only and does not restore lifecycle progress.
