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
- Treat `init-research` as a deterministic scan only. Do not present its charter
  or pheromones as the final colony intent without AI synthesis.

## Codebase Summary

Display a brief summary from the scan:
- Languages and frameworks (from `languages` and `frameworks` fields)
- README summary (if `readme_summary` is non-empty, show first 200 chars)
- Git: `{git_history.commits}` commits, `{git_history.contributors}` contributors on `{git_history.branch}`
- Governance: list detected linters, CI, test frameworks from `governance` object
- Prior colonies: `{prior_colonies.count}` archived colonies (if > 0)

## Intent Refinement

Before creating colony state, ask one compact batch of 4-7 questions when the
goal is broad, vague, or missing implementation boundaries.

For beginners: this is the part where you turn "build my app" into a clear
mission the colony can plan from.

Ask about:
- target users and the user-visible outcome
- must-have success criteria
- non-goals and things to avoid
- affected systems, integrations, or data
- constraints such as deadlines, platform support, budget, compliance, or style
- first useful milestone
- biggest risk or unknown

Use the answers plus the codebase scan to synthesize:
- `refined_goal`: a precise one-sentence colony goal
- `charter`: JSON with `intent`, `vision`, `governance`, `goals`,
  `tech_stack`, `key_risks`, and `constraints`
- `synthesized_pheromones`: at most 3 goal-specific steering signals

Do not simply echo the runtime-generated charter. Keep each charter field under
2000 characters.

## Colony Charter

Present the synthesized charter for user review:

```
**Refined Goal:** {refined_goal}
**Intent:** {synthesized_charter.intent}
**Vision:** {synthesized_charter.vision}
**Governance:** {synthesized_charter.governance}
**Goals:** {synthesized_charter.goals}
**Tech Stack:** {synthesized_charter.tech_stack}
**Key Risks:** {synthesized_charter.key_risks}
**Constraints:** {synthesized_charter.constraints}
```

## Colony Mode

Before creating colony state, ask the user to choose the operating mode:

1. Colony Mode — use the existing default lifecycle with fewer prompts.
2. Orchestrator Mode — ask guided boundary questions at phase points for tighter user control.

If the user skips the choice or the host is non-interactive, use Colony Mode.
Store the choice as `selected_colony_mode`, with value `colony` or
`orchestrator`.

## Pheromone Suggestions

Separate scan warnings from strategic pheromones:

- Scan warnings are deterministic housekeeping from `init-research`.
- Strategic pheromones are AI-synthesized steering for this specific colony.

Do not suggest README/changelog/license/formatter housekeeping as pheromones
unless the user goal is specifically documentation, release process, licensing,
or formatting. Show important housekeeping separately as "Scan warnings" instead.

If `synthesized_pheromones` is non-empty, present as tick-to-approve:

```
Suggested colony steering:

1. [{type}] {content}
   Reason: {reason}
   [ ] Approve / [ ] Skip
```

Show each suggestion and let the user approve or skip individually. If nothing
specific is worth steering, say "No strategic pheromones suggested."

## Shelf Backlog

Before colony state creation:

1. Run `aether shelf-list --json --status shelved` and parse the JSON output.
2. Check `result.total`.
3. If `result.total > 0`:
   - Display: `## Shelf Backlog — {N} ideas from prior colonies`
   - Show numbered list from `result.entries`
   - For each item, present options:
     ```
     1. Promote to this colony
     2. Keep on shelf
     3. Delete permanently
     ```
   - Collect user choices
   - If any promoted: run `aether shelf-promote-batch --ids "id1,id2" --colony "{goal}"`
   - If any dismissed: run `aether shelf-dismiss-batch --ids "id1,id2"`
   - Promoted items become todos: append `[shelf:{category}] {text}` to `active_todos` in the session file or colony state
4. If no shelved entries exist:
   - Skip silently (no prompt)

## Cross-Platform Drift Guard

If you change init interview, synthesis, pheromone, shelf, approval, or closeout
behavior here, update `.aether/commands/init.yaml`, `cmd/command_guide.go`, and
the Codex skill `aether-colony-creation` in the same change. Verify
`aether command-guide init --platform codex` still describes the matching Codex
flow.

## Approval

- Use AskUserQuestion with 3 options: proceed, revise goal, cancel.
- After approval, for each approved synthesized pheromone, run `aether pheromone-write --type "{type}" --content "{content}" --source "init-synthesis"`.
- Then run `AETHER_OUTPUT_MODE=visual aether init --colony-mode "{selected_colony_mode}" --charter-json '<synthesized charter JSON>' "<refined goal>"`, where `<synthesized charter JSON>` is the JSON-serialized charter object from the AI synthesis.
- Do not write `.aether/QUEEN.md`, `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand from this command spec.
- If setup is missing, relay the runtime guidance exactly.
- If docs and runtime disagree, runtime wins.
