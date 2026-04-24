<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-build
description: "🔨 Build a phase — Queen dispatches workers, colony self-organizes"
---

You are the **Queen**. The colony is building through real wrapper-spawned workers.

The phase to build is: `$ARGUMENTS`

If `$ARGUMENTS` is empty, show: `Usage: /ant-build <phase_number>`

## Colony Context

Before planning the dispatch, ground yourself in runtime truth:

1. Run `AETHER_OUTPUT_MODE=visual aether status` to see current colony state, phase progress, and active signals.
2. Keep that runtime context in view while framing the phase.
3. Do not inspect or mutate `.aether/data/` by hand — read runtime context through the CLI only.

## Active Signals

Before spawning workers, present active pheromones as a compact steering block:

- `REDIRECT` first — make hard constraints explicit.
- `FOCUS` second — summarize the main areas that deserve extra attention.
- `FEEDBACK` last — mention only the lightweight adjustments that matter for this phase.
- Include strength or remaining-life context so the user understands why each signal matters right now.
- If there are no active signals, say so plainly and keep the block short.

## Phase Framing

Use the grounded status context to frame the requested work:

- Present it as `Phase N of M — Name`.
- Add a one-line purpose that explains why this phase matters to the colony goal.
- Keep the framing concise; orient the user without replaying the full plan.

## Dispatch Manifest

Use the Go `aether` CLI as the source of truth. Ask the Go runtime for the authoritative worker plan. Immediately before the command, say:

`Asking the runtime for the dispatch manifest...`

```
AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only
```

Parse `result.dispatch_manifest`. This manifest is the only source for worker names, castes, execution waves, task waves, task IDs, playbooks, selected tasks, and success criteria. Do not parse visual output.

## Playbook Procedure

Load `.aether/docs/command-playbooks/build-wave.md` and use it as the spawning procedure. The runtime owns the dispatch manifest; the playbook owns the wrapper ceremony and prompt structure.

## Wave Execution

For each step in `dispatch_manifest.execution_plan`, execute the matching `dispatch_manifest.dispatches` entries whose `execution_wave` matches that step:

1. Before spawning, run:
   `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "{caste}" --name "{name}" --task "{task}" --depth 1`
2. Spawn the matching platform agent using the platform's Task/subagent mechanism with `subagent_type="{agent_name}"` or its equivalent.
3. Use a concise agent description: `{caste emoji} {Caste} {name}: {task}`.
4. Inject the phase objective, task metadata, dependencies, success criteria, active signals, relevant playbook instructions, and any specialist findings already collected.
5. Require every worker to return a terminal structured result with: `name`, `caste`, `stage`, `execution_wave`, `wave`, `task_id`, `status`, `summary`, `files_created`, `files_modified`, `tests_written`, `tool_count`, `blockers`, and `duration`.
6. After each worker returns, run:
   `AETHER_OUTPUT_MODE=json aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`

Multiple agent calls issued in one assistant message may run in parallel when the platform supports it. Respect `dispatch_manifest.execution_plan`: serial steps stay serial; parallel steps may spawn together. Pre-wave specialists such as Archaeologist, Oracle, Architect, or Ambassador must complete before builder/scout task waves. Post-wave specialists such as Probe, Watcher, Measurer, or Chaos must run after builder/scout task waves.

## Completion Packet

After all workers have terminal results, write a temporary completion JSON file outside `.aether/data/` with this shape:

```json
{
  "dispatch_manifest": {
    "...": "the exact result.dispatch_manifest object"
  },
  "dispatches": [
    {
      "name": "Mason-67",
      "caste": "builder",
      "stage": "wave",
      "wave": 1,
      "execution_wave": 11,
      "task_id": "1.1",
      "status": "completed",
      "summary": "Implemented the assigned work.",
      "files_created": [],
      "files_modified": [],
      "tests_written": [],
      "tool_count": 0,
      "blockers": [],
      "duration": 0
    }
  ]
}
```

Then finalize the external worker packet through the runtime:

```
AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file <completion_file>
```

The runtime records `dispatch_mode: external-task`, claims, spawn-tree statuses, state transition to `BUILT`, and next-step truth.

## After the Build

Once `build-finalize` succeeds:

1. Summarize what moved forward and which workers/castes actually ran.
2. Note only the most relevant signal or risk that should stay in view.
3. Guide the user first to `/ant-continue` as the next command.
4. Keep the closeout tight — one clear next move is better than an option menu.

## Guardrails

- Do NOT run `aether build` without `--plan-only` from this wrapper.
- Do NOT run `aether build --synthetic` after real agent workers complete.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, or pheromone files.
- Do NOT parse visual output as authoritative state.
- Do NOT invent worker names, castes, or waves; use `dispatch_manifest`.
- If docs and runtime disagree, runtime wins.
