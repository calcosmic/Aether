<!-- Generated from .aether/commands/init.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-init
description: "🥚 Initialize Aether colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth, but do not skip the init
foundation pass.

- If `$ARGUMENTS` is empty, show `Usage: /ant-init "<your goal here>"`.
- First run `AETHER_OUTPUT_MODE=json aether init-research --goal "$ARGUMENTS" --target .`.
- Parse the JSON output for the `charter` object and `pheromone_suggestions` array.

## Codebase Summary

Display a brief summary from the scan:
- Languages and frameworks (from `languages` and `frameworks` fields)
- README summary (if `readme_summary` is non-empty, show first 200 chars)
- Git: `{git_history.commits}` commits, `{git_history.contributors}` contributors on `{git_history.branch}`
- Governance: list detected linters, CI, test frameworks from `governance` object
- Prior colonies: `{prior_colonies.count}` archived colonies (if > 0)

## Colony Charter

Present the charter for user review:

```
**Intent:** {charter.intent}
**Vision:** {charter.vision}
**Governance:** {charter.governance}
**Goals:** {charter.goals}
```

## Pheromone Suggestions

If `pheromone_suggestions` is non-empty, present as tick-to-approve:

```
The scan detected {N} patterns. Review and approve:

1. [{type}] {content}
   Reason: {reason}
   [ ] Approve / [ ] Skip
```

Show each suggestion and let the user approve or skip individually.

## Approval

- Ask with 3 options: proceed, revise goal, cancel.
- After approval, for each approved pheromone suggestion, run `aether pheromone-write --type "{type}" --content "{content}" --source "init-research"`.
- Then run `AETHER_OUTPUT_MODE=visual aether init "$ARGUMENTS"`.
- Do not write `.aether/QUEEN.md`, `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand from this command spec.
- If setup is missing, relay the runtime guidance exactly.
- If docs and runtime disagree, runtime wins.
