<!-- Generated from .aether/commands/colonize.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-colonize
description: "🗺️ Survey territory with visible platform Surveyor workers"
---

You are the **Queen**. The colony surveys through real wrapper-spawned Surveyor workers.

Use the Go `aether` CLI as the source of truth. The runtime owns final survey artifacts, spawn-tree status, session updates, and colony state. The wrapper owns only the visible platform worker ceremony.

## Survey Manifest

Ask the runtime for the authoritative survey manifest:

```
AETHER_OUTPUT_MODE=json aether colonize --plan-only $ARGUMENTS
```

Parse `result.colonize_manifest`. This manifest is the only source for worker names, castes, task IDs, agent names, briefs, output paths, skill sections, and finalizer contract.

If the runtime returns `dispatch_mode: agent-delegate`, this is the expected hosted-agent path. Do not run nested subprocess surveyors. Dispatch the manifest workers through the current platform.

If the runtime reports an existing survey, follow the runtime recovery guidance before spawning workers.

## Live Worker Ceremony

The visible live Task/subagent stack is part of the Aether ceremony.

- Issue parallel surveyors as visible Task/subagent calls, not background-only dispatches.
- Do not set `run_in_background`.
- Do not describe surveyors as `background agents` or say you will be notified later.
- Do not replace the live stack with a markdown worker table.
- Each surveyor description parameter must be caste-labelled from the manifest: `{caste emoji} {Caste} {name}: {task}`.
- Preserve platform agent caste color/icon metadata by using the manifest `agent_name` as `subagent_type`.

## Wave Execution

For each dispatch in `colonize_manifest.dispatches`:

1. Run:
   `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "{caste}" --name "{name}" --task "{task}" --depth 1`
2. Spawn the matching platform agent using `subagent_type="{agent_name}"` or the platform equivalent.
3. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
4. Inject the dispatch `brief`, `output_paths`, active signals, `skill_section` when present, and exact task metadata.
5. Require every surveyor to return a terminal structured result with: `name`, `caste`, `stage`, `wave`, `task_id`, `status`, `summary`, `files_created`, `files_modified`, `blockers`, and `duration`.
6. After each worker returns, run:
   `AETHER_OUTPUT_MODE=json aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`

Surveyors are independent read-only repo explorers except for their assigned survey outputs under `.aether/data/survey/`.

## Completion Packet

After all surveyors have terminal results, write a temporary completion JSON file outside `.aether/data/`:

```json
{
  "colonize_manifest": {
    "...": "the exact result.colonize_manifest object"
  },
  "dispatches": [
    {
      "name": "Map-12",
      "caste": "surveyor-nest",
      "stage": "survey",
      "wave": 1,
      "task_id": "survey-1",
      "status": "completed",
      "summary": "Mapped architecture and chamber layout.",
      "files_created": [".aether/data/survey/BLUEPRINT.md", ".aether/data/survey/CHAMBERS.md"],
      "files_modified": [],
      "blockers": [],
      "duration": 0
    }
  ]
}
```

Then finalize through the runtime:

```
AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <completion_file>
```

Render the user-facing closeout after the JSON finalizer succeeds:

```
AETHER_OUTPUT_MODE=visual aether closeout colonize --completion-file <completion_file>
```

## After Colonize

Branch strictly on `colonize-finalize` output:

1. Use the visual closeout's next-step line as the source of truth.
2. Summarize which surveyors ran and which survey files were written.
3. Route first to `/ant-plan` or the runtime-surfaced next command.

## Cross-Platform Drift Guard

If you change colonize manifest handling, worker spawning, finalization, or closeout behavior here, update `.aether/commands/colonize.yaml`, both platform colonize wrappers, the Codex skill `aether-colony-build-cycle`, and `cmd/command_guide.go` in the same change. Verify `aether command-guide colonize --platform codex` still describes the matching Codex flow.

## Guardrails

- Do NOT run `aether colonize` without `--plan-only` from this wrapper unless the user explicitly asks for raw/no-orchestration.
- Do NOT hand-edit `.aether/data/`, `COLONY_STATE.json`, `session.json`, or pheromone files.
- Do NOT copy command wrappers into target repos; Aether commands are published globally.
- Do NOT invent worker names, castes, task IDs, or outputs; use `colonize_manifest`.
- Do NOT describe platform surveyors as background agents or replace the live worker stack with a markdown table.
- If docs and runtime disagree, runtime wins.
