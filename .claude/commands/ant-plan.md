<!-- Aether-managed: runtime spec at .aether/commands/plan.yaml. Synced by aether update. -->
---
name: ant-plan
description: "📋 Generate a depth-scoped colony plan with real Scout and Route-Setter agents"
---

You are the **Queen Ant Colony**. Plan through real wrapper-spawned planning workers.

Use the Go `aether` CLI as the source of truth. The runtime owns the final plan, canonical artifacts, state transitions, and next-step truth.

## Decision Moment 1 — Depth Proposal

The plan flow has exactly two decision moments; this is the first. Resolve all three depth knobs — granularity, task decomposition depth, and verification depth — together as one tap-to-approve proposal instead of three separate questions.

If this is a refresh after completed work, keep the current colony and pass `--refresh --revision-type <type> --revision-reason "<why>"` to every `aether host plan` call below. Research and verification revisions also require one or more repository-relative `--revision-evidence <path>` arguments.

1. Request a first manifest with `aether host plan $ARGUMENTS`, omitting `--depth`, `--planning-depth`, and `--verification-depth` unless `$ARGUMENTS` already names them, so the runtime computes its smart defaults and the reason for each.
2. Print `result.depth_proposal_card` verbatim. Do not restate, summarize, or re-reason the recommendations — the runtime computes the reasons; the wrapper only prints the card it is given (see `.aether/docs/wrapper-runtime-ux-contract.md`).
3. Accept the recommendations on a single confirmation, or, when the user names a knob and an option number, request a fresh manifest with the corresponding `--depth` / `--planning-depth` / `--verification-depth` flag set to that option's value. Never ask the user to type a value.

Under `/ant-run`, accept the depth recommendations without prompting.

## Planning Manifest

Run the TS host to fetch the authoritative planning manifest for one planning iteration — the same call as Decision Moment 1, now carrying the accepted depth flags:

```
aether host plan --depth <choice> --planning-depth <choice2> --verification-depth <choice3> $ARGUMENTS
```

The TS host is the sole entry point to the Go CLI for manifest generation. See `.aether/docs/wrapper-host-contract.md`.

Parse `result.plan_manifest` or `result.planning_manifest`. This manifest is the only source for `planning_run_id`, `iteration`, target confidence, selected gaps, previous draft, worker names, castes, waves, task IDs, briefs, and finalizer contract.

When the manifest includes `revision`, pass its worker briefs verbatim. Completed
phases are immutable; Route-Setter must output replacement unfinished phases
only. The Go finalizer preserves completed evidence and assigns final IDs.

Save the JSON envelope to a temporary manifest file outside `.aether/data/`.

## Decision Moment 2 — Research Batch

The second and final decision moment. Answer the whole per-phase research batch in one interaction, before any worker spawns.

1. Print `result.research_proposal_card` verbatim when it is non-empty. Do not compose your own recommendation or reason.
2. Approve the batch with `aether plan-research-approve --approve-all`, or flip specific phases with `aether plan-research-approve --flip <ids>`.
3. Request a fresh manifest afterward (same command as the Planning Manifest section) so the gated `phase_research` dispatches appear, and surface `result.research_warning` whenever `result.research_awaiting_approval` is true — a plan that skipped research because nobody answered must say so.
4. Fast-depth note: on a fast run the Queen recommends skip for every phase, the batch still appears, and a flipped-on phase researches at the fast preset's 80% / 4-iteration budget.

Under `/ant-run`, answer the research batch with `aether plan-research-approve --auto` and print the returned `log_line` in the run log. Autopilot never pauses for either decision moment.

## Clarification Gate

Before spawning workers, inspect `result.orchestrator_boundary_guidance` and `unresolved_clarifications`:

- If boundary guidance is active or `next` is `aether discuss`, pause and route to `aether discuss`. Request a fresh manifest after resolution. Do not reuse the pre-discuss manifest. Rerun `after_discuss_next` after resolution.
- If unresolved clarifications exist, route to `/ant-discuss`. Proceed with implicit assumptions only if the user explicitly chooses to continue.

## Runtime Spawn Ceremony

Before spawning planning workers, render the runtime-owned planning ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest_file>
```

This output is display-only; do not parse it as state.

## Worker Spawning

Dispatch every worker in `plan_manifest.dispatches`, using manifest names, castes, task IDs, briefs, `permission_profile`, and `agent_name` as `subagent_type`. The set is: one Scout in wave 1, zero or more `phase_research` Scouts also in wave 1 (parallel with the base Scout — one per approved research phase, researching its domain), then exactly one Route-Setter in wave 2. Scout's `repository_read_only` profile must remain host-enforced; never substitute an unrestricted agent or a prompt-only promise. Preserve caste-labelled descriptions: `{caste emoji} {Caste} {name}: {task}`. Research Scouts iterate under a confidence loop; their per-iteration confidence lines are runtime-emitted, not composed by the wrapper.

- Issue parallel workers as visible Task/subagent calls. Do not set `run_in_background`.
- Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.
- Spawn all wave-1 workers (base Scout + research Scouts) in the same message so they run concurrently. Announce the research wave in one line: `🔍 Researching {N} phases before routing`.
- For Route-Setter, include the Scout terminal result in the prompt and note that fresh per-phase research now exists at `.aether/data/phase-research/`.
- If the manifest includes `selected_gaps` or `previous_plan_draft`, keep them in the brief and require fresh evidence or resolved gaps before allowing confidence to rise. Surface `selected_gaps` to the user between iterations: `Unresolved gaps this iteration:` followed by the list, so they can see what the next pass is chasing.

For each manifest wave:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each worker.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.
6. For Route-Setter, include the Scout terminal result in the prompt.
7. After each worker returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
8. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>`.

All wave-1 workers (base Scout and any research Scouts) must complete before the wave-2 Route-Setter starts — the route is set with research in hand.

## Finalize

After workers return, collect results into a completion JSON. Include `planning_run_id`, `iteration`, Scout `scout_report`, Route-Setter `phase_plan`, and a compact `source_summary`, then finalize through the runtime:

```
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>
```

If the JSON result contains `requires_next_iteration: true`, do not render final closeout and do not claim the colony plan is complete. Request a fresh `aether host plan` manifest with the same depth, planning depth, target, and max-iteration controls, then repeat Scout -> Route-Setter -> `plan-finalize`.

When `plan-finalize` returns a completed plan, render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <completion_file>
```

## After Planning

Branch on the `plan-finalize` result:

1. If planning succeeded, use the visual closeout's next-step line as the source of truth.
2. Summarize selected depth, phase count, confidence, `planning_loop.stop_reason`, and which agents ran.
3. For a revision, surface the accepted `plan_revision` reason and preserved, superseded, and replacement phase IDs.
4. Route first to `/ant-build 1` or the runtime-surfaced next build command.
5. If planning blocked, follow the runtime recovery command first.

## Cross-Platform Drift Guard

If you change planning depth selection, clarification handling, worker spawning,
finalization, or closeout behavior here, update `.aether/commands/plan.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the
same change. Verify `aether command-guide plan --platform codex` still describes
the matching Codex flow.

## Guardrails

- Do NOT run direct `aether plan` from this wrapper for manifest generation; use `aether host plan`.
- Do NOT run `aether plan --synthetic` after real agent workers complete.
- Do NOT read or write colony state files, session files, planning artifacts, or pheromone files by hand.
- Do NOT parse visual output as authoritative state.
- Do NOT invent Scout or Route-Setter names, castes, waves, or task IDs; use `plan_manifest`.
- Do NOT dispatch planning workers beyond the manifest's dispatch list; the contract is Scout (+ manifest-listed phase_research Scouts) then Route-Setter per iteration.
- Do NOT reuse a manifest or completion packet across iterations.
- Do NOT repeat or renumber completed phases during a revision, and do not reuse packets from the superseded revision.
- Do NOT treat `requires_next_iteration: true` as a completed colony plan.
- Do NOT describe platform workers as background agents or replace the live worker stack with a markdown table.
- Do NOT add a third decision moment; the plan flow has exactly two — the depth proposal card and the research batch card.
- Do NOT route either decision-moment card through `aether discuss`; the discuss redirect remains only for `orchestrator_boundary_guidance`.
- Do NOT compose depth recommendations, reasons, or research recommendations in this wrapper; print the runtime-emitted card verbatim.
- If docs and runtime disagree, runtime wins.
