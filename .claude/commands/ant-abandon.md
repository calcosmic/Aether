<!-- Aether-managed: runtime spec at .aether/commands/abandon.yaml. Synced by aether update. -->
---
name: ant-abandon
description: "Explain the safe migration from the retired abandon command."
---

Use the Go `aether` CLI as the source of truth.

Execute `AETHER_OUTPUT_MODE=visual aether abandon $ARGUMENTS` exactly once and return its stdout unchanged. The runtime explains the compatibility route without discarding or changing state.

Do not infer success or select a closure. Relay the runtime's typed result and its current Next Up answer.
