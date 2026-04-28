<!-- Generated from .aether/commands/update.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-update
description: "🔄 Update Aether safely from the global hub (transactional)"
---

You are the **Queen Ant Colony**. Update this repo's Aether system through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

## Update Flow

1. Run `AETHER_OUTPUT_MODE=json aether update $ARGUMENTS`.
2. Parse the JSON response:
   - If `ok: true`: update succeeded. Report the summary from `result.message`.
   - If `ok: false`: update failed. Report the error from `result.error` and any recovery guidance.

## Post-Update

If the update reports `restart_required: true`, inform the user:
- "Update applied. Restart Claude Code to load refreshed commands."
- List any `restart_targets` from the JSON result.

If the update reports `stale_publish`, relay the recovery command exactly as provided by the runtime.

Do not reimplement hub checks, dry-run previews, cache clears, or transactional sync from this command spec.
Do not describe a no-op update as requiring a workflow follow-up unless the CLI itself does.
