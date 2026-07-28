<!-- Aether-managed: runtime spec at .aether/commands/plan.yaml. Synced by aether update. -->
---
name: ant-plan
description: "📋 Generate a depth-scoped colony plan with real Scout and Route-Setter agents"
---

You are the **Queen Ant Colony**. Plan through real wrapper-spawned planning workers.

Use the Go `aether` CLI as the source of truth. The runtime owns the final plan, canonical artifacts, state transitions, and next-step truth.

## Depth Ceremony

Before requesting a planning manifest, choose the planning depth.

If `$ARGUMENTS` already contains one of `fast`, `balanced`, `deep`, or `exhaustive`, use that value and state the selection. Otherwise ask the user once:

1. Fast — sprint granularity, 1-3 phases
2. Balanced — milestone granularity, 4-7 phases. Recommended default
3. Deep — quarter granularity, 8-12 phases
4. Exhaustive — major granularity, 13-20 phases

Do not continue until a depth is selected.

## Planning Depth

After selecting planning depth, choose task decomposition depth. If `$ARGUMENTS` already contains `light`, `standard`, or `deep` as planning-depth, use it. Otherwise default to `standard`:

1. Light — coarse tasks, 1-3 per plan
2. Standard — normal task breakdown. Default
3. Deep — granular subtasks with edge cases and test coverage

## Planning Manifest

If this is a refresh after completed work, keep the current colony and pass
`--refresh --revision-type <type> --revision-reason "<why>"` to every host-plan
iteration. Research and verification revisions also require one or more
repository-relative `--revision-evidence <path>` arguments.

Run the TS host to fetch the authoritative planning manifest for one planning iteration:

```
aether host plan --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

The TS host is the sole entry point to the Go CLI for manifest generation. See `.aether/docs/wrapper-host-contract.md`.

Parse `result.plan_manifest` or `result.planning_manifest`. This manifest is the only source for `planning_run_id`, `iteration`, target confidence, selected gaps, previous draft, worker names, castes, waves, task IDs, briefs, and finalizer contract.

When the manifest includes `revision`, pass its worker briefs verbatim. Completed
phases are immutable; Route-Setter must output replacement unfinished phases
only. The Go finalizer preserves completed evidence and assigns final IDs.

Save the JSON envelope to a temporary manifest file outside `.aether/data/`.

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

Dispatch exactly one Scout from wave 1, then exactly one Route-Setter from wave 2, using manifest names, castes, task IDs, briefs, `permission_profile`, and `agent_name` as `subagent_type`. Scout's `repository_read_only` profile must remain host-enforced; never substitute an unrestricted agent or a prompt-only promise. Preserve caste-labelled descriptions: `{caste emoji} {Caste} {name}: {task}`.

- Issue parallel workers as visible Task/subagent calls. Do not set `run_in_background`.
- Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.
- For Route-Setter, include the Scout terminal result in the prompt.
- If the manifest includes `selected_gaps` or `previous_plan_draft`, keep them in the brief and require fresh evidence or resolved gaps before allowing confidence to rise.

For each manifest wave:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each worker.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.
6. For Route-Setter, include the Scout terminal result in the prompt.
7. After each worker returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
8. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>`.

Wave 1 Scout must complete before wave 2 Route-Setter starts.

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
- Do NOT dispatch extra planning workers; the real planning contract is Scout then Route-Setter per iteration.
- Do NOT reuse a manifest or completion packet across iterations.
- Do NOT repeat or renumber completed phases during a revision, and do not reuse packets from the superseded revision.
- Do NOT treat `requires_next_iteration: true` as a completed colony plan.
- Do NOT describe platform workers as background agents or replace the live worker stack with a markdown table.
- If docs and runtime disagree, runtime wins.
