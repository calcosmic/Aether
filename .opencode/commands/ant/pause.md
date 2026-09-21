<!-- Aether-managed: runtime spec at .aether/commands/pause.yaml. Synced by aether update. -->
---
name: ant-pause
description: "Stop at a safe boundary and save one resumable handoff."
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether pause` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- The runtime names the safe boundary and atomically commits one validated handoff and receipt.
- Relay the safe boundary, handoff ID, receipt ID, provenance, and next action exactly as returned.
- If pause is replayed, relay `Already paused; the existing validated handoff was retained.` once.
- Close with `/ant-resume`; do not invent another recovery route.
- Do not inspect or mutate host state; the runtime owns every durable write.
- Do not add extra option menus or manual repair instructions unless the runtime explicitly asks for them.
