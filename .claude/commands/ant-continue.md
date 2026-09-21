<!-- Aether-managed: runtime spec at .aether/commands/continue.yaml. Synced by aether update. -->
---
name: ant-continue
description: "👁️ Verify build work, extract learnings, and advance the colony"
---

You are the **Queen Ant Colony**. Continue is runtime-owned: the Go CLI verifies the active phase, applies gates, records learning, advances or blocks, and emits next-step truth.

Use the Go `aether` CLI as the source of truth.

## Ownership Split

| Concern | Owner |
|---------|-------|
| Manifest generation | TS host (`aether host continue --dry-run`) for heavy review |
| Reviewer spawning | Wrapper (platform Agent tool) for heavy review |
| Verification + gating | Go runtime (default continue command) |
| State mutation | Go runtime (`continue-finalize`) |

The wrapper is the sole conductor for interactive reviewer spawning in the
heavy review path. The TS host provides the manifest only.

## Required Cross-Stage State

Carry these values forward once the runtime produces them:
- `phase_id`
- `verification_depth`
- `verification_status`
- `manifest_file` (heavy path only)
- `next_action`

## Default Continue

🐜 The colony's inspection point: a build finishing without error is not the
same claim as the app working, and continue is where the runtime checks the
difference before anything advances.

**Purpose:** Ask the runtime to verify the active phase and either advance the
colony or hand back exactly what blocked it. No reviewer spawning happens on
this path — verification runs inside the runtime process itself.

**Reads:** current colony state via `aether status`.

**Spawns:** nothing. The runtime's own verification workers (Watcher, and at
heavier depth Probe, Gatekeeper, and Auditor) run inside the `aether continue`
process; the wrapper spawns no platform agents on this path. That absence is
itself useful method — default continue is one runtime call, not a
wrapper-conducted ceremony.

Ground yourself first:

```
AETHER_OUTPUT_MODE=visual aether status
```

Then run the fast path directly:

```
AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS
```

This is the normal path.

**Stop conditions:** the runtime returns a verification result — advance,
block, or complete. Do NOT use `--plan-only` or `continue-finalize` for
default fast continue, and do not spawn wrapper workers here.

## Heavy External Review

🐜 Why this matters:
- Builders verifying their own work is confirmation bias.
- Independent Watchers catch bugs builders miss.
- "Build passing" is not the same as "app working".

That is the reason this path exists at all: when the user asks for the older
visible review ritual, or the runtime itself asks for wrapper-spawned
reviewers, verification runs through independently spawned reviewer agents
instead of a single runtime call.

**Purpose:** Conduct the heavy review ceremony — fetch a reviewer manifest
from the TS host, spawn each reviewer as a visible platform agent, collect
their terminal results, and hand the completed packet to the runtime to
finalize.

### Decide the review team before spawning

🐜 Every reviewer is a full agent run — roughly 100,000 tokens and several
minutes. This is the most expensive thing the colony does.

The phase's own build/test check already ran once, before continue started —
it is not re-spawned here. Nothing is unconditional in the team you are
picking except a reviewer forced by a named risk signal (see below).

For each other reviewer, find the part of the phase that concerns its domain
and classify what the phase actually says:

| What the phase says | Verdict |
|---------------------|---------|
| Names a symptom, bug, or complaint in this domain | **include** |
| Asks for new or changed work in this domain | **include** |
| No matching words, but the plain meaning clearly falls in this domain | **include** |
| Says this property is unchanged, unaffected, or out of scope | **exclude** |
| Does not touch this domain at all | **exclude** |

**You may not write "include" against "unchanged" or "does not touch".** If you
want to, the classification is wrong — fix the classification, not the verdict.

This exists because the previous version of this instruction was prose advice
saying much the same thing, and a real session still spent 111,800 tokens on a
Measurer reviewing the phase *"the envelope re-arms once and never again;
latency and memory behaviour is unchanged"*. The words were present. The
sentence said there was nothing to review. Advice is skimmable; a
classification with a fixed verdict is not.

The trap runs the other way too: *"the dashboard feels sluggish with lots of
rows"* names no performance vocabulary at all and is entirely a performance
question.

Re-fetch with your decision:

```
aether continue --plan-only \
  --castes probe \
  --caste-why probe="correctness fix in the retrigger path; the phase states perf is unchanged" \
  --caste-reason "confirm the retrigger fix didn't reintroduce a coverage gap"
```

An empty optional team is a normal, good answer — nothing is required unless a
named risk signal forces it.

Reviewers are no longer added by default. The floor is: a reviewer is forced
only when the phase's own wording, or its changed files, names one of five
signals — credentials/auth, payments, release sign-off, data deletion,
database migration — and if it does, that forced reviewer is restored
whatever you propose, with the signal stated on the card. Trimming an unforced
reviewer is a cost decision; skipping the review a named signal forces is not
available at any cost except the owner's own explicit, recorded decline.
Relay what the runtime added or dropped.

**Reads:** the manifest returned by `aether host continue --dry-run`;
`continue_manifest.context_capsule` (read once, not per-dispatch — the capsule
is the SOLE source of pheromone signals; `pheromone_section` is no longer
populated); each dispatch's runtime-provided `brief` verbatim;
`dispatch.skill_section` when present.

**Spawns:** the reviewers named in `result.manifest.continue_manifest` — one
platform agent per named dispatch, spawned in the manifest's own wave order.

Only use this path when the user explicitly requests `--classic-ceremony`,
`--verification-depth heavy`, or the runtime asks for wrapper-spawned review
workers.

Fetch the manifest from the TS host in plan-only mode:

```
aether host continue --dry-run --classic-ceremony $ARGUMENTS
```

`aether host continue --dry-run --verification-depth heavy $ARGUMENTS` is
equivalent for callers that already use depth flags. The TS host is the sole
manifest authority.

Parse `result.manifest.continue_manifest`. Save the full JSON envelope to a
temporary manifest file outside `.aether/data/`. See
`.aether/docs/wrapper-host-contract.md` for the full field shapes.

Before spawning reviewers, inspect `result.manifest.continue_manifest` for
`orchestrator_boundary_guidance`. If active or `next` is `aether discuss`,
stop the flow, route to `aether discuss`, and request a fresh manifest after
resolution. Rerun `after_discuss_next` after resolution.

Render the runtime-owned heavy-review ceremony:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow continue --manifest-file <manifest_file>
```

Spawn reviewers as visible live Task/subagent panels. Do not set `run_in_background`. Each reviewer's prompt = `continue_manifest.context_capsule` (read once from the manifest, not per-dispatch, reused for every reviewer this continue run spawns — it already carries every active pheromone signal; do not prepend `pheromone_section`, which the runtime no longer populates) prepended VERBATIM ahead of each dispatch's own runtime-provided `brief`, then `dispatch.skill_section` appended when present. Nothing else, nothing invented.

For each heavy-review wave:

1. Render `AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow continue --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each reviewer.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`. Keep `{name}` in it: the worker name is what this phase's token record joins a transcript row to a worker on, so a shortened label reports the whole run as costing nothing.
5. The reviewer's prompt = `continue_manifest.context_capsule` (read once, prepended verbatim — the sole carrier of pheromone signals) + each dispatch's runtime-provided `brief` verbatim + `dispatch.skill_section` when present. Nothing else, nothing invented.
6. After each reviewer returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
7. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow continue --worker-file <worker_file>`.

Collect terminal results into a completion JSON containing the original
`continue_manifest` and a `dispatches` array.

Finalize with:

```
AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file <completion_file>
```

Then render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <completion_file>
```

**Stop conditions:** every manifest reviewer has returned a terminal result
and `continue-finalize` has run on the collected packet, or
`orchestrator_boundary_guidance` routed the flow to `aether discuss` before
any reviewer spawned.

## After Continue

**Purpose:** Translate the runtime's verification result into the one next
thing the user should do — advance, recover from a block, or move to seal.

**Reads:** the closeout command's next-step line (heavy path) or the default
continue command's own output (fast path).

**Spawns:** nothing; this stage only communicates the runtime's result.

**If the phase advanced:**
1. Use the visual closeout's next-step line as the source of truth.
2. Summarize what the runtime verified and learned.
3. If the runtime's result reports a `phase_commit`, mention in one line that the colony made a git save-point of the files its workers changed (or why it skipped). The `aether phase-commits` toggle controls this: check it with `aether phase-commits get`, turn it off with `aether phase-commits set off`.
4. Route first to `/ant-build N+1`.

**If continue is blocked:**
1. Translate the blocker into plain language.
2. If the runtime surfaced a specific recovery command, route to that first.
3. Only fall back to `/ant-continue` when the runtime did not surface a more specific recovery step.

**If the colony completed:**
1. Mark completion briefly.
2. Route first to `/ant-seal`.

**Worker questions checkpoint (after the result is reported):** run
`AETHER_OUTPUT_MODE=json aether handoff-decisions --phase <n>`. If `count` > 0,
ask each question via AskUserQuestion (at most 4; always offering "Let the
colony proceed on its current assumption") and record real answers with
`AETHER_OUTPUT_MODE=json aether decision-answer --question "<q>" --answer "<a>" --phase <n>`.
An unanswered question never blocks — it resurfaces at the next boundary.

**Steering checkpoint (after the result is reported):** if the runtime output
contains a "Suggested Steering" section, present those proposals to the user
as a real multiple-choice question (the AskUserQuestion tool, multi-select) —
one option per suggestion, each stating its plain-English consequence, plus a
"none of these" option. For each approval run
`AETHER_OUTPUT_MODE=json aether suggest-approve --approve <id>`; for each
explicit rejection run `--dismiss <id>`. Never write a suggestion the user did
not pick, and never re-ask about suggestions they dismissed.

**If the user asks what it cost:** tell them they can run `aether spend` to see, worker by worker, how many tokens each worker's own tool reported for this run.
That is a read-only detail view and the wrapper must never run it unprompted: the run's own figures are already on screen, and repeating them would print the same numbers twice.

**Stop conditions:** the user has one clear next command, and no verification
result was fabricated by the wrapper.

<success_criteria>
- The runtime verified the phase and emitted a next-step line.
- On the heavy path, every manifest reviewer returned a terminal result and `continue-finalize` ran on the collected packet.
</success_criteria>

<failure_modes>
- Continue blocked: translate the blocker into plain language and route to the runtime's own recovery command first.
- Boundary guidance active: route to `aether discuss` and request a fresh manifest.
- Reviewer failure on the heavy path: never fabricate a verification result.
</failure_modes>

<read_only>
This wrapper never reads or writes, by hand: `COLONY_STATE.json`, `session.json`, `CONTEXT.md`, `HANDOFF.md`, and pheromone files. Runtime state is read only through runtime commands such as `aether status` and the runtime's own continue output.
</read_only>

## Cross-Platform Drift Guard

If you change continue behavior here, update `.aether/commands/continue.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the
same change. Verify `aether command-guide continue --platform codex` still describes the matching Codex flow.

## Guardrails

- Do NOT use `--plan-only` or `continue-finalize` for default fast continue.
- Do NOT run `aether host continue --classic-ceremony` without `--dry-run` from this wrapper; that triggers the TS host dispatched path which duplicates the wrapper's own reviewer spawning. Always use `aether host continue --dry-run`.
- Do NOT replay verification loops or reimplement runtime gate logic.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, `CONTEXT.md`, `HANDOFF.md`, or pheromone files.
- Do NOT invent worker names, castes, or waves; use `continue_manifest` only in the heavy path.
- Do NOT describe platform reviewers as background agents or replace the live worker stack with a markdown worker table.
- Do NOT expose raw provider stdout/stderr, tokens, or auth probe output; use the Go availability category and sanitized next action.
- If docs and runtime disagree, runtime wins.
