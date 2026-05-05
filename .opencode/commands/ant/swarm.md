<!-- Generated from .aether/commands/swarm.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-swarm
description: "🔥 Real-time colony swarm display + visible bug-destroyer workers"
---

Use the Go `aether` CLI as the source of truth. The runtime owns swarm artifacts, spawn-tree status, worker wave contracts, and final result persistence. The wrapper owns only visible platform worker ceremony.

## Watch Mode

If the user provides no problem description, keep live visibility direct:

```
AETHER_OUTPUT_MODE=visual aether swarm --watch
```

Do not request a manifest for watch mode.

## Swarm Manifest

For a bug-destroyer target, ask the runtime for the authoritative swarm manifest:

```
AETHER_OUTPUT_MODE=json aether swarm --plan-only $ARGUMENTS
```

Parse `result.swarm_manifest`. This manifest is the only source for worker names, castes, roles, waves, task IDs, agent names, briefs, response contracts, and finalizer contract.

If the runtime returns `dispatch_mode: agent-delegate`, this is the expected hosted-agent path. Do not run nested subprocess swarm workers. Dispatch the manifest workers through the current platform.

## Wave Execution

For each dispatch in `swarm_manifest.dispatches`:

1. Run:
   `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Swarm" --caste "{caste}" --name "{name}" --task "{task}" --depth 1`
2. Spawn the matching platform agent using `subagent_type="{agent_name}"` or the platform equivalent.
3. Use a concise agent description: `🔥 Swarm {name}: {role}`.
4. Inject the dispatch `brief`, `response_contract`, active signals, and exact task metadata.
5. Require every worker to return a terminal structured result with: `name`, `caste`, `role`, `task`, `status`, `summary`, `files`, `tests`, `blockers`, `response`, and `duration`.
6. After each worker returns, run:
   `AETHER_OUTPUT_MODE=json aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`

Preserve the runtime wave order:

1. Wave 1 investigation workers may run together.
2. Wave 2 builder waits for wave 1 summaries.
3. Wave 3 watcher waits for builder completion.

The wrapper must not invent worker names, roles, or waves.

## Completion Packet

After all swarm workers have terminal results, write a temporary completion JSON file outside `.aether/data/`:

```json
{
  "swarm_manifest": {
    "...": "the exact result.swarm_manifest object"
  },
  "dispatches": [
    {
      "name": "Trace-12",
      "caste": "tracker",
      "role": "tracker",
      "task": "Reproduce the issue and trace the failure path.",
      "status": "completed",
      "summary": "Tracked the issue to a missing nil guard.",
      "files": [],
      "tests": [],
      "blockers": [],
      "duration": 0,
      "response": {
        "role": "tracker",
        "status": "completed",
        "summary": "Tracked the issue to a missing nil guard.",
        "findings": ["The panic starts in the session lookup path."],
        "evidence": ["pkg/auth/handler.go"],
        "root_cause": "missing session guard",
        "recommendation": "Restore the guard and add a regression test.",
        "proposed_fix": "",
        "files_touched": [],
        "tests_written": [],
        "verification": []
      }
    }
  ]
}
```

Then finalize through the runtime:

```
AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <completion_file>
```

## After Swarm

Branch strictly on `swarm-finalize` output:

1. Summarize which workers actually ran.
2. Summarize the root cause, fix or blocker, files/tests touched, and verification evidence.
3. Route first to the runtime-surfaced `next` command.

## Cross-Platform Drift Guard

If you change swarm manifest handling, worker spawning, finalization, or closeout behavior here, update `.aether/commands/swarm.yaml`, both platform swarm wrappers, the Codex skill `aether-colony-build-cycle`, and `cmd/command_guide.go` in the same change. Verify `aether command-guide swarm --platform codex` still describes the matching Codex flow.

## Guardrails

- Do NOT run nested subprocess swarm workers from this wrapper.
- Do NOT run `aether swarm` without `--plan-only` from this wrapper unless the user explicitly asks for raw/no-orchestration.
- Do NOT hand-edit `.aether/data/`, `COLONY_STATE.json`, `session.json`, or pheromone files.
- Do NOT copy command wrappers into target repos; Aether commands are published globally.
- Do NOT invent worker names, castes, roles, waves, task IDs, or outputs; use `swarm_manifest`.
- If docs and runtime disagree, runtime wins.
