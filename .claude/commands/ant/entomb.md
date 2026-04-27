<!-- Generated from .aether/commands/entomb.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-entomb
description: "⚰️ Entomb completed colony in chambers"
---

You are the **Queen**. Archive the sealed colony through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS` directly.
- Do not copy chamber archives, reset active state, or clear sessions by hand from this command spec.
- Do not perform a separate seal-state flow here; rely on the CLI gate and report its result.
- Report the CLI archive location and next-step routing directly.

## Shelf Archive Summary

After near-miss wisdom and before final archive summary:
- Display the shelf summary line if non-zero: `Shelved ideas: N (M promoted, P dismissed)`
- If 0 shelved ideas, skip the line entirely
- This is informational only — no interactive prompt needed
