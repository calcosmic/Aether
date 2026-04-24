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

## Colony Context

Before requesting the manifest, ground yourself in runtime truth:

```
AETHER_OUTPUT_MODE=visual aether status
```

Use that output to keep the user oriented, but do not parse visual output as authoritative state.

## Planning Manifest

Ask the Go runtime for the authoritative planning manifest:

```
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> $ARGUMENTS
```

Parse `result.plan_manifest` or `result.planning_manifest`. This manifest is the only source for worker names, castes, waves, task IDs, briefs, survey context, depth, granularity bounds, and finalizer contract.

If the runtime returns `existing_plan: true`, do not spawn workers. Summarize the existing plan and route to the runtime-surfaced next command.

## Clarification Gate

Before spawning planning workers, inspect the runtime result for `unresolved_clarifications` or `clarification_warning`.

- If unresolved clarifications exist, pause the planning ceremony and surface the warning plainly.
- Route first to `/ant-discuss` so the user can resolve the questions through the runtime.
- Proceed with implicit assumptions only if the user explicitly chooses to continue despite the warning.
- If the user proceeds, carry that choice into the Scout and Route-Setter prompts as a known planning constraint.

## Wave Execution

For each dispatch in the manifest, execute the planned workers by wave:

1. Before spawning, run:
   `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "{caste}" --name "{name}" --task "{task}" --depth 1`
2. Spawn the matching platform agent using the platform's Task/subagent mechanism with `subagent_type="{agent_name}"` or its equivalent.
3. Use a concise agent description: `{caste emoji} {Caste} {name}: {task}`.
4. Inject the selected depth, survey context, manifest `brief`, active signals, and exact task metadata.
5. Require every worker to return a terminal structured result with: `name`, `caste`, `stage`, `wave`, `task_id`, `status`, `summary`, `blockers`, and `duration`.
6. After each worker returns, run:
   `AETHER_OUTPUT_MODE=json aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`

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

## After Planning

Branch strictly on the `plan-finalize` result:

1. If planning succeeded, summarize selected depth, phase count, confidence, and which planning agents ran.
2. Route first to `/ant-build 1` or the exact runtime-surfaced next build command.
3. If planning blocked, translate the blocker into plain language and follow the runtime recovery command first.

## Guardrails

- Do NOT run `aether plan` without `--plan-only` from this wrapper.
- Do NOT run `aether plan --synthetic` after real agent workers complete.
- Do NOT read or write colony state files, session files, planning artifacts, or pheromone files by hand.
- Do NOT parse visual output as authoritative state.
- Do NOT invent Scout or Route-Setter names, castes, waves, or task IDs; use `plan_manifest`.
- Do NOT write `.aether/data/planning` as the authority path; pass results to `plan-finalize`.
- If docs and runtime disagree, runtime wins.
