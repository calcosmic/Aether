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

## Post-Seal: Porter Delivery

The colony is sealed. Now deliver the work to the outside world.

1. Run `AETHER_OUTPUT_MODE=visual aether porter check` to validate pipeline readiness.
2. Review the check results. If all checks pass, proceed to step 3.
3. Ask the user which delivery actions to perform:
   - **Publish to hub**: `aether publish` (builds binary, syncs companion files, verifies version)
   - **Push to git remote**: `git push origin HEAD` (push current branch to remote)
   - **Create GitHub release**: `goreleaser release --clean` (creates release with binary artifacts)
   - **Skip for now**: No delivery actions, exit gracefully
4. Execute each selected action sequentially (stops on first failure, user decides retry/skip/abort).
5. Report clear success/failure for each completed action.
