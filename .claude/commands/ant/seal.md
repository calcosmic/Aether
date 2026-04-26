<!-- Generated from .aether/commands/seal.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-seal
description: "🏺 Seal the colony with Crowned Anthill milestone"
---

You are the **Queen**. Seal the colony through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS` directly.
- Do not write ceremony files, milestone state, or archive data by hand from this command spec.
- Do not ask for separate confirmation unless the CLI itself does.
- Report the CLI seal result and next-step routing directly.

## Auto-Promotion: High-Confidence Instincts to QUEEN.md

After `aether seal` succeeds, promote high-confidence instincts to the local QUEEN.md Wisdom section so colony learnings persist in the sealed artifact.

1. Read `.aether/data/instincts.json` from the colony data directory.
2. For each instinct entry where `confidence >= 0.8` and `action` is non-empty:
   - Run `aether queen-promote-instinct --id <instinct_id>` to write it to the local QUEEN.md.
3. If no instincts meet the threshold, skip silently -- this step is non-blocking.
4. Report how many instincts were promoted in the final seal summary.
