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

## Blocker Handling

If `aether seal` exits with an error (non-zero exit code), check whether the output contains a blocker table:

- If blockers are listed, relay the blocker table to the user showing each blocker ID, description, and resolution command: `aether flag <id> --resolve`
- Suggest the user either resolves blockers first or re-runs with the --force flag: `/ant-seal --force`
- The `--force` flag passes through to the runtime via `$ARGUMENTS`

## Shelf Candidate Detection

After the blocker check and before archive creation, the runtime automatically detects shelf candidates:

1. Run `aether shelf-detect` to get candidate JSON
2. If candidates exist, present them in a tick-to-approve checkbox list:
   ```
   [ ] {category}: {text} (auto-detected)
   [ ] {category}: {text} (auto-detected)
   ```
3. Include a "Permanent guidance candidates" section for recurring REDIRECTs
4. User ticks which ones to shelf; unticked items are discarded
5. If any approved: run `aether shelf-add --text "..." --category ...` for each
6. If no candidates exist, skip silently

## Post-Seal Report

After seal succeeds, report what the runtime did:

- "Instincts promoted to local QUEEN.md: {count from output}"
- If the runtime printed a SUGGESTION about hive-eligible instincts, relay that to the user: "Consider promoting eligible instincts globally with `aether queen-promote-instinct <id>`"
- "FOCUS signals expired: {count}"
- If shelf candidates were detected: "Shelf candidates: {N} (X instincts, Y pheromones, Z flags, W redirects)"

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
