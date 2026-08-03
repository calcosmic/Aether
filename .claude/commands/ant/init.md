<!-- Aether-managed: runtime spec at .aether/commands/init.yaml. Synced by aether update. -->
---
name: ant-init
description: "🥚 Initialize Aether colony through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth, but do not skip the init
foundation pass.

init is a guided setup ritual, not a build — it spawns no workers, so no
stage below carries a `**Spawns:**` line. That absence is itself useful
method: it is also why init.md has no host-manifest step to parse, unlike
build.md, plan.md, and continue.md.

- If `$ARGUMENTS` is empty, show `Usage: /ant-init "<your goal here>"`.
- First run `AETHER_OUTPUT_MODE=json aether init-research --goal "$ARGUMENTS" --target .`.
- Parse the JSON output for the `charter` object and `pheromone_suggestions` array.
- Treat `init-research` as a deterministic scan only. Do not present its charter
  or pheromones as the final colony intent without AI synthesis.

<success_criteria>
Command is complete when:
- a colony exists with the user-approved `refined_goal` and charter
- the chosen colony mode (`colony` or `orchestrator`) is recorded
- every approved synthesized pheromone was written through `aether pheromone-write`, never by hand
- the next-step routing (`/ant-colonize`, `/ant-discuss`, `/ant-plan`) is shown
</success_criteria>

<failure_modes>
### Goal Empty
If `$ARGUMENTS` is empty:
- Show `Usage: /ant-init "<your goal here>"`
- Stop before running `init-research`

### User Cancels At Approval
If the user chooses cancel at `## Approval`:
- Write nothing — no charter call, no pheromone writes, no shelf promotion or dismissal, nothing persisted
- Shelf choices collected earlier are discarded; the backlog is left exactly as it was
- Stop the command

### Previous Colony Was Sealed
If the runtime reports a previous colony was sealed:
- Say so plainly to the user
- Start fresh rather than silently overwriting the sealed colony's state

### Setup Missing
If `aether init-research` or `aether init` reports the runtime or hub is unavailable:
- Relay the runtime guidance exactly
- Do not hand-copy assets or reconstruct state to compensate
</failure_modes>

<read_only>
This wrapper never writes these files by hand — every one is Go-runtime-owned:
- .aether/data/COLONY_STATE.json
- .aether/data/session.json
- .aether/data/constraints.json
- .aether/data/pheromones.json
- .aether/QUEEN.md
</read_only>

## Required Cross-Stage State

Carry these values forward once produced, in current vocabulary only:
- `refined_goal` — the precise one-sentence goal from Intent Refinement
- `synthesized_charter` — the AI-synthesized charter JSON from Intent Refinement
- `synthesized_pheromones` — at most 3 goal-specific steering signals from Intent Refinement
- `selected_colony_mode` — `colony` or `orchestrator`, from Colony Mode
- `approved_pheromones` — the subset of `synthesized_pheromones` the user approved at Approval
- `next_action` — the next-step command the user should run after init completes
- `promoted_shelf_ids` — the shelf entry IDs the user chose to promote, spent only in the Approval init call
- `dismissed_shelf_ids` — the shelf entry IDs the user chose to dismiss, spent only in the Approval init call

## Codebase Summary

🐜 A quick look around before anything is asked.

**Purpose:** Ground the interview and charter synthesis in what this repo
already is, before asking the user anything.
**Reads:** `languages`, `frameworks`, `readme_summary`, `git_history`, and
`governance` from the `init-research` JSON output.

Display a brief summary from the scan:
- Languages and frameworks (from `languages` and `frameworks` fields)
- README summary (if `readme_summary` is non-empty, show first 200 chars)
- Git: `{git_history.commits}` commits, `{git_history.contributors}` contributors on `{git_history.branch}`
- Governance: list detected linters, CI, test frameworks from `governance` object

**Stop conditions:** If `init-research` fails or reports setup is missing,
relay the runtime guidance exactly and stop — do not ask interview questions
from a stage that never scanned the repo.

## Prior Context

🐜 Past colonies shape what this one should be.

**Purpose:** Show what came before so the new goal is informed by prior
outcomes, not written in a vacuum.
**Reads:** `prior_colonies.count` and up to 3 entries from
`prior_colonies.recent` in the same `init-research` output.

If `prior_colonies.count > 0`, show what came before **before** asking for the new goal — past colonies shape what the next one should be:

```
Prior Context — {count} archived colonies

Most recent:
1. "{recent[0].goal}" — {recent[0].outcome} ({recent[0].entombed_at date})
2. "{recent[1].goal}" — {recent[1].outcome}
3. "{recent[2].goal}" — {recent[2].outcome}
```

Show up to 3 entries from `prior_colonies.recent` (goal truncated to ~120 chars). If `recent` is empty but `count > 0`, fall back to `Prior colonies: {count} archived`. If `count` is 0, skip this section silently.

**Stop conditions:** No user input is required here. If `prior_colonies.count`
is 0, skip this section silently and move straight to Intent Refinement.

## Intent Refinement

🐜 Turn a rough goal into a mission the colony can plan from.

**Purpose:** Turn a broad, vague, or boundary-less goal into a precise
`refined_goal` and a synthesized charter, using the codebase scan plus a short
user interview.
**Reads:** `$ARGUMENTS` (the raw goal), the Codebase Summary scan fields, and
Prior Context (if shown).

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

**Stop conditions:** If `$ARGUMENTS` was empty, this stage never runs — that
was already handled above. Otherwise a synthesized `refined_goal` and
`synthesized_charter` are required before continuing to Colony Charter.

## Colony Charter

🐜 Show the synthesis back to the user before anything is written.

**Purpose:** Present the synthesized charter for review so the user can catch
a bad synthesis before Approval.
**Reads:** `refined_goal` and `synthesized_charter` from Intent Refinement.

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

**Stop conditions:** This is a display-only stage; it never halts on its own.
Charter values carry forward to Approval unchanged unless the user later
revises the goal or cancels.

## Colony Mode

🐜 Two ways to run the colony — pick one before state exists.

**Purpose:** Let the user choose between the low-friction Colony Mode and the
tighter-control Orchestrator Mode before any state is created.
**Reads:** whether the host is interactive; no file inputs.

Before creating colony state, ask the user to choose the operating mode:

1. Colony Mode — use the existing default lifecycle with fewer prompts.
2. Orchestrator Mode — ask guided boundary questions at phase points for tighter user control.

If the user skips the choice or the host is non-interactive, use Colony Mode.
Store the choice as `selected_colony_mode`, with value `colony` or
`orchestrator`.

**Stop conditions:** If the user skips the choice or the host is
non-interactive, default to Colony Mode and continue — never block init on
this choice.

## Pheromone Suggestions

🐜 Separate deterministic housekeeping from strategic steering.

**Purpose:** Let the user approve or skip AI-synthesized steering signals
individually, without conflating them with scan housekeeping.
**Reads:** `synthesized_pheromones` from Intent Refinement and housekeeping
warnings from `init-research`.

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

**Stop conditions:** If no strategic pheromones are synthesized, say so and
continue — this stage never blocks init.

## Shelf Backlog

🐜 Give old shelved ideas one more chance before this colony starts without them.

**Purpose:** Offer the user a chance to promote, keep, or delete backlog ideas
from prior colonies before this colony's state exists.
**Reads:** `aether shelf-list --json --status shelved` output (`result.total`,
`result.entries`).

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
   - Record the chosen IDs as `promoted_shelf_ids` and `dismissed_shelf_ids`. Nothing is written yet.
   - Show the chosen entries' `text` and `category` from `result.entries` back to the user as what this colony will carry forward if they approve
   - State plainly that the shelf is not touched at this stage: `aether init` performs the promotion and dismissal itself at Approval, after consent, so a cancel, a revised goal, or a failed init leaves the backlog exactly as it was
4. If no shelved entries exist:
   - Skip silently (no prompt)

**Stop conditions:** If no shelved entries exist, skip silently. This stage
never writes — it only records the chosen IDs for Approval to spend.

## Cross-Platform Drift Guard

If you change init interview, synthesis, pheromone, shelf, approval, or closeout
behavior here, update `.aether/commands/init.yaml`, `cmd/command_guide.go`, and
the Codex skill `aether-colony-creation` in the same change. Verify
`aether command-guide init --platform codex` still describes the matching Codex
flow.

## Approval

👑 The Queen sets the colony's intention once the user says yes.

**Purpose:** Get explicit user consent before any persistence happens, and
mark the moment the colony's intention becomes real.
**Reads:** the accumulated cross-stage state — `refined_goal`,
`synthesized_charter`, `selected_colony_mode`, `synthesized_pheromones`,
`promoted_shelf_ids`, `dismissed_shelf_ids`.

- Use AskUserQuestion with 3 options: proceed, revise goal, cancel.
- On proceed, before calling the runtime: 👑 Queen has set the colony's intention — "{refined_goal}"
- Then run `AETHER_OUTPUT_MODE=visual aether init --colony-mode "{selected_colony_mode}" --charter-json '<synthesized charter JSON>' --promote-shelf "{promoted_shelf_ids}" --dismiss-shelf "{dismissed_shelf_ids}" "<refined goal>"`, where `<synthesized charter JSON>` is the JSON-serialized charter object from the AI synthesis. Omit `--promote-shelf` and `--dismiss-shelf` when the user chose nothing on the shelf. The goal in this call is the same `refined_goal` the promotion is recorded against, so a revised goal can never orphan a promoted entry.
- Only once `aether init` has returned success, for each approved synthesized pheromone, run `aether pheromone-write --type "{type}" --content "{content}" --source "init-synthesis"`. If `aether init` failed, skip this step entirely — nothing is written.
- If the runtime reports failed shelf IDs (`shelf_failed`), tell the user which backlog ideas did not carry forward and that they are still on the shelf.
- Do not write `.aether/QUEEN.md`, `.aether/data/COLONY_STATE.json`, `session.json`, `constraints.json`, or `pheromones.json` by hand from this command spec.
- Do not hand-render the init banner — `init_ceremony.go` already owns it; the runtime call above shows the banner and result.
- If setup is missing, relay the runtime guidance exactly.
- If docs and runtime disagree, runtime wins.

**Stop conditions:** A cancel or a failed `aether init` both end the command
with nothing persisted — no charter, no pheromones, no shelf promotion or
dismissal.

**Next steps:**
- `/ant-colonize` — map an existing codebase before planning
- `/ant-discuss` — clarify intent before the plan is drawn
- `/ant-plan` — generate the phase plan
