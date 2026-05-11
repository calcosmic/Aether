<!-- Generated from .aether/commands/plan.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-plan
description: "📋 Generate a depth-scoped colony plan with real Scout and Route-Setter agents"
---

You are the **Queen Ant Colony**. The colony plans through real wrapper-spawned planning workers.

Use the Go `aether` CLI as the source of truth. The runtime owns the final plan, canonical artifacts, state transitions, and next-step truth. The wrapper owns only the user-facing depth ceremony and platform Task/subagent spawning.

## Depth Ceremony

Before requesting a planning manifest, choose the planning depth.

If `$ARGUMENTS` already contains one of `fast`, `balanced`, `deep`, or `exhaustive`, use that value and state the selection. Otherwise ask the user once:

1. Fast — sprint granularity, 1-3 phases
2. Balanced — milestone granularity, 4-7 phases. Recommended default
3. Deep — quarter granularity, 8-12 phases
4. Exhaustive — major granularity, 13-20 phases

Do not continue until a depth is selected.

## Planning Depth

After selecting the planning depth (which controls how many phases are generated), choose the task decomposition depth. This controls how thoroughly tasks are decomposed within each plan -- it is independent of phase count.

If `$ARGUMENTS` already contains one of `light`, `standard`, or `deep` as a planning-depth value, use it and state the selection. Otherwise default to `standard`:

1. Light — coarse tasks, 1-3 per plan with objective-level descriptions
2. Standard — normal task breakdown. Default
3. Deep — granular subtasks including edge cases, error handling, and test coverage as separate tasks

## Colony Context

Before requesting the manifest, ground yourself in runtime truth:

```
AETHER_OUTPUT_MODE=visual aether status
```

Use that output to keep the user oriented, but do not parse visual output as authoritative state.

## Planning Manifest

Ask the Go runtime for the authoritative planning manifest:

```
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice2> $ARGUMENTS
```

Parse `result.plan_manifest` or `result.planning_manifest`. This manifest is the only source for worker names, castes, waves, task IDs, briefs, survey context, depth, granularity bounds, and finalizer contract.

Save the full JSON envelope to a temporary manifest file outside `.aether/data/`. The ceremony commands read that file so the visual plan stays tied to the same runtime manifest.

If the user requested a refresh and the runtime returns `dispatch_mode: agent-delegate`, do not run nested subprocess planning. Treat the returned manifest as the same host-dispatch contract: dispatch Scout and Route-Setter through the current platform, then finish with `plan-finalize`.

If the runtime returns `existing_plan: true`, do not spawn workers. Summarize the existing plan and route to the runtime-surfaced next command.

## Clarification Gate

Before spawning planning workers or rendering spawn ceremonies, inspect the runtime result for `orchestrator_boundary_guidance`, `unresolved_clarifications`, and `clarification_warning`.

- If `orchestrator_boundary_guidance.active` is true or `next` is `aether discuss`, pause the planning ceremony and surface its summary plainly.
- Route first to `aether discuss` so the user can resolve the runtime-owned questions.
- Tell the user to rerun `after_discuss_next` after the answers are resolved.
- After a guided answer is resolved, request a fresh plan-only manifest. Do not reuse the pre-discuss manifest and do not ask, answer, or store boundary questions in wrapper markdown.

- If unresolved clarifications exist, pause the planning ceremony and surface the warning plainly.
- Route first to `/ant-discuss` so the user can resolve the questions through the runtime.
- Proceed with implicit assumptions only if the user explicitly chooses to continue despite the warning.
- If the user proceeds, carry that choice into the Scout and Route-Setter prompts as a known planning constraint.

## Runtime Spawn Ceremony

Before spawning Scout and Route-Setter workers, render the runtime-owned old-style planning ceremony:

```
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest_file>
```

This output is for display only. Do not parse it as state.

## Live Worker Ceremony

The visible live Task/subagent stack is part of the Aether ceremony.

- Issue parallel planning workers as visible Task/subagent calls, not background-only dispatches.
- Do not set `run_in_background`.
- Do not describe workers as `background agents` or say you will be notified later.
- Do not replace the live stack with a markdown worker table.
- Each worker description parameter must be exactly caste-labelled from the manifest: `{caste emoji} {Caste} {name}: {task}`.
- Preserve platform agent caste color/icon metadata by using the manifest `agent_name` as `subagent_type`.

## Wave Execution

For each dispatch in the manifest, execute the planned workers by wave:

1. Before spawning a manifest wave, render:
   `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave "{execution_wave}"`
2. Then run:
   `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "{caste}" --name "{name}" --task "{task}" --depth 1`
3. Spawn the matching platform agent using the platform's Task/subagent mechanism with `subagent_type="{agent_name}"` or its equivalent.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Inject the selected depth, planning depth selection, survey context, manifest `brief`, active signals, dispatch `skill_section` when present, and exact task metadata.
6. Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading. The brief contains the read budget, no-repeat loop guard, output contract, and stop condition.
7. For Route-Setter, include the Scout terminal result in the prompt so it can consume Scout findings directly instead of re-running the survey.
8. If a planning worker keeps rereading the same file or command, stop waiting for more exploration and mark that worker `blocked` with a concrete blocker; do not manually reconcile it as completed.
9. Require every worker to return a terminal structured result with: `name`, `caste`, `stage`, `wave`, `task_id`, `status`, `summary`, `blockers`, and `duration`.
10. After each worker returns, run:
   `AETHER_OUTPUT_MODE=json aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`
11. Write that one terminal result to a temporary worker JSON file and render:
   `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>`

Wave 1 Scout must complete before wave 2 Route-Setter starts. The Route-Setter result must include `phase_plan` using the manifest's required `phase-plan.json` schema:

```json
{
  "phases": [
    {
      "name": "",
      "description": "",
      "tasks": [
        {
          "goal": "",
          "constraints": [],
          "hints": [],
          "success_criteria": [],
          "depends_on": []
        }
      ],
      "success_criteria": []
    }
  ],
  "confidence": {
    "knowledge": 0,
    "requirements": 0,
    "risks": 0,
    "dependencies": 0,
    "effort": 0,
    "overall": 0
  },
  "gaps": []
}
```

## Completion Packet

After Scout and Route-Setter have terminal results, write a temporary completion JSON file outside `.aether/data/` with this shape:

```json
{
  "plan_manifest": {
    "...": "the exact result.plan_manifest object"
  },
  "dispatches": [
    {
      "name": "Track-80",
      "caste": "scout",
      "stage": "scouting",
      "wave": 1,
      "task_id": "plan-scout",
      "status": "completed",
      "summary": "Mapped the planning surface.",
      "blockers": [],
      "duration": 0,
      "scout_report": {
        "findings": [],
        "gaps": [],
        "confidence": 90,
        "study_files": []
      }
    },
    {
      "name": "Route-12",
      "caste": "route_setter",
      "stage": "routing",
      "wave": 2,
      "task_id": "plan-route-setter",
      "status": "completed",
      "summary": "Produced the executable phase plan.",
      "blockers": [],
      "duration": 0,
      "phase_plan": {
        "phases": [],
        "confidence": {
          "knowledge": 0,
          "requirements": 0,
          "risks": 0,
          "dependencies": 0,
          "effort": 0,
          "overall": 0
        },
        "gaps": []
      }
    }
  ]
}
```

Then finalize through the runtime:

```
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>
```

The runtime writes canonical planning artifacts, updates `COLONY_STATE.json`, records spawn-tree statuses, updates session/CONTEXT/HANDOFF, and emits next-step truth.

Render the user-facing closeout after the JSON finalizer succeeds:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <completion_file>
```

## After Planning

Branch strictly on the `plan-finalize` result:

1. If planning succeeded, use the visual closeout's next-step line as the source of truth.
2. Summarize selected depth and planning depth, phase count, confidence, and which planning agents ran.
3. Route first to `/ant-build 1` or the exact runtime-surfaced next build command.
4. If planning blocked, translate the blocker into plain language and follow the runtime recovery command first.

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
- Do NOT write `.aether/data/planning` as the authority path; pass results to `plan-finalize`.
- If docs and runtime disagree, runtime wins.
