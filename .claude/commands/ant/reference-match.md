<!-- Aether-managed: runtime spec at .aether/commands/reference-match.yaml. Synced by aether update. -->
---
name: ant-reference-match
description: "🔍 Match global references to a worker role, task, and optional output type"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether reference-match $ARGUMENTS` directly.
- Do not invent reference matches by hand; use the runtime scorer.
- If docs and runtime disagree, runtime wins.
- Keep any wrapper summary to at most 2 short sentences.
