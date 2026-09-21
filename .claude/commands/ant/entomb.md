<!-- Aether-managed: runtime spec at .aether/commands/entomb.yaml. Synced by aether update. -->
---
name: ant-entomb
description: "Archive and clear the sealed colony."
---

You are the **Queen**. Delegate this archive-and-clear transaction to the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Entomb is a separate, optional owner action after reviewable seal state. Never invoke it from the seal workflow.
- Execute `AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS` exactly once after the owner chooses it. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- The runtime owns the preflight preview, confirmation, and ordered stages: **Stage archive** → **Write digest manifest** → **Verify bytes and cross-references** → **Publish chamber and tombstone** → **Clear active state**.
- Treat the typed runtime result as authoritative. Do not copy archive bytes, do not parse visual output, do not clear or write active state, and do not fabricate verification.
- Report the runtime's chamber, digest, receipt, retained-state or success status, and next-step routing directly.
