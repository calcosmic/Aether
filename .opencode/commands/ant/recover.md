<!-- Aether-managed: runtime spec at .aether/commands/recover.yaml. Synced by aether update. -->
---
name: ant-recover
description: "Explain the safe migration from the retired recovery command."
---

Use the Go `aether` CLI as the source of truth.

Execute `AETHER_OUTPUT_MODE=visual aether recover $ARGUMENTS` exactly once and return its stdout unchanged. The runtime explains the compatibility route without repairing or changing state.

Do not infer success or select a repair. Relay the runtime's typed result and its current Next Up answer.
