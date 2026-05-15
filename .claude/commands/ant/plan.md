<!-- Generated from .aether/commands/plan.yaml - DO NOT EDIT DIRECTLY -->
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

Run the TS host to fetch the authoritative planning manifest:

```
aether host plan --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

The TS host is the sole entry point to the Go CLI for manifest generation. See `.aether/docs/wrapper-host-contract.md`.

Parse `result.plan_manifest` or `result.planning_manifest`. This manifest is the only source for worker names, castes, waves, task IDs, briefs, and finalizer contract.

Save the JSON envelope to a temporary manifest file outside `.aether/data/`.

## Clarification Gate

Before spawning workers, inspect `result.orchestrator_boundary_guidance` and `unresolved_clarifications`:

- If boundary guidance is active or `next` is `aether discuss`, pause and route to `aether discuss`. Request a fresh manifest after resolution. Do not reuse the pre-discuss manifest. Rerun `after_discuss_next` after resolution.
- If unresolved clarifications exist, route to `/ant-discuss`. Proceed with implicit assumptions only if the user explicitly chooses to continue.

## Worker Spawning

Dispatch Scout from wave 1, then Route-Setter from wave 2, using manifest names, castes, task IDs, briefs, and `agent_name` as `subagent_type`. Preserve caste-labelled descriptions: `{caste emoji} {Caste} {name}: {task}`.

- Issue parallel workers as visible Task/subagent calls. Do not set `run_in_background`.
- Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.
- For Route-Setter, include the Scout terminal result in the prompt.

Wave 1 Scout must complete before wave 2 Route-Setter starts.

## Finalize

After workers return, collect results into a completion JSON and finalize through the runtime:

```
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>
```

Then render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <completion_file>
```

## After Planning

Branch on the `plan-finalize` result:

1. If planning succeeded, use the visual closeout's next-step line as the source of truth.
2. Summarize selected depth, phase count, confidence, and which agents ran.
3. Route first to `/ant-build 1` or the runtime-surfaced next build command.
4. If planning blocked, follow the runtime recovery command first.

## Cross-Platform Drift Guard

If you change planning depth selection, clarification handling, worker spawning,
finalization, or closeout behavior here, update `.aether/commands/plan.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the
same change. Verify `aether command-guide plan --platform codex` still describes
the matching Codex flow.

## Guardrails

- Do NOT run `aether plan` without `--plan-only` from this wrapper.
- Do NOT run `aether plan --synthetic` after real agent workers complete.
- Do NOT read or write colony state files, session files, planning artifacts, or pheromone files by hand.
- Do NOT parse visual output as authoritative state.
- Do NOT invent Scout or Route-Setter names, castes, waves, or task IDs; use `plan_manifest`.
- Do NOT describe platform workers as background agents or replace the live worker stack with a markdown table.
- If docs and runtime disagree, runtime wins.
