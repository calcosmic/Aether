<!-- Aether-managed: runtime spec at .aether/commands/entomb.yaml. Synced by aether update. -->
---
name: ant-entomb
description: "Archive and clear the sealed colony."
---

You are the **Queen**. Delegate this archive-and-clear transaction to the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Entomb is a separate, optional owner action after reviewable seal state. Never invoke it from the seal workflow.
- Execute `AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS` exactly once after the owner chooses it.
- The runtime owns the preflight preview, confirmation, and ordered stages: **Stage archive** → **Write digest manifest** → **Verify bytes and cross-references** → **Publish chamber and tombstone** → **Clear active state**.
- Treat the typed runtime result as authoritative. Do not copy archive bytes, do not parse visual output, do not clear or write active state, and do not fabricate verification.
- Report the runtime's chamber, digest, receipt, retained-state or success status, and next-step routing directly.
