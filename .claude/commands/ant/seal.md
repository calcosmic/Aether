<!-- Generated from .aether/commands/seal.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-seal
description: "🏺 Seal the colony with visible final review workers"
---

You are the **Queen**. Seal the colony through the runtime manifest/finalizer contract.

Use the Go `aether` CLI as the source of truth. The wrapper only dispatches host-platform agents and reports their terminal results back to the runtime.

## Raw Bypass

If the user explicitly asks for raw, exact, direct, or no-orchestration seal, run:

```bash
AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS
```

Otherwise use the hosted review flow below.

## Seal Manifest

Run:

```bash
AETHER_OUTPUT_MODE=json aether seal --plan-only $ARGUMENTS
```

Parse `result.seal_manifest`. If the runtime returns blockers or recovery guidance, surface that output and stop. Do not fabricate review results.

Save the full JSON envelope to a temporary manifest file outside `.aether/data/`. The ceremony commands read that file so the final-review display uses the same runtime manifest.

Expected manifest:

- `dispatch_mode`: `plan-only` or `agent-delegate`
- `requires_finalizer`: `true`
- `dispatches`: Gatekeeper, Auditor, and Probe final-review workers
- `finalizer_command`: `AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>`

## Runtime Spawn Ceremony

Before spawning final-review workers, render the runtime-owned old-style seal ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow seal --manifest-file <manifest_file>
```

This output is display-only; do not parse it as state.

## Live Worker Ceremony

The visible live Task/subagent stack is part of the Aether ceremony.

- Issue same-wave final-review workers as visible Task/subagent calls, not background-only dispatches.
- Do not set `run_in_background`.
- Do not describe reviewers as `background agents` or say you will be notified later.
- Do not replace the live stack with a markdown worker table.
- Each reviewer description parameter must be caste-labelled from the manifest: `{caste emoji} {Caste} {name}: {task}`.
- Preserve platform agent caste color/icon metadata by using the manifest `agent_name` as `subagent_type`.

## Worker Dispatch

Dispatch the runtime-provided workers through the host platform in manifest wave order.

For each dispatch:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow seal --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1`.
3. Spawn the host agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Give the worker the exact `brief` from the manifest.
6. Tell the worker this is final review before seal and it must not modify repo source files.
7. Collect a terminal result with:
   - `name`
   - `caste`
   - `stage`
   - `wave`
   - `task_id`
   - `status`
   - `summary`
   - `blockers`
   - `report`
8. Run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
9. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow seal --worker-file <worker_file>`.

Terminal statuses are `completed`, `passed`, `blocked`, `failed`, or `timeout`.

## Completion Packet

Write a JSON completion packet:

```json
{
  "seal_manifest": { "...": "the exact manifest from result.seal_manifest" },
  "dispatches": [
    {
      "stage": "seal-review",
      "wave": 1,
      "caste": "gatekeeper",
      "name": "Gate-12",
      "task_id": "seal-review-gatekeeper",
      "status": "completed",
      "summary": "No release blockers found.",
      "blockers": [],
      "report": "..."
    }
  ]
}
```

Then run:

```bash
AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <completion_file>
```

Render the user-facing closeout after the JSON finalizer succeeds:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file <completion_file>
```

Branch strictly on `seal-finalize` output:

- If blocked, report the runtime blocker text and stop.
- If sealed, use the visual closeout's next-step line as the source of truth.
- Summarize the workers and the runtime seal result.
- Follow the runtime's Porter readiness output in visual mode.

## Post-Seal Delivery

Do not run delivery commands automatically. If the runtime says the colony is sealed and shows Porter readiness, ask the user which delivery actions to perform:

- publish to hub
- push to git remote
- create GitHub release
- skip delivery for now

Run selected delivery actions sequentially and stop on first failure.

## Guardrails

- Do NOT write colony state files, session files, review reports, pheromone files, or archive files by hand.
- Do NOT parse visual output as truth; use JSON output for programmatic data.
- Do NOT run `aether seal` without `--plan-only` from this wrapper unless the user explicitly asks for raw/no-orchestration.
- Do NOT run Porter delivery commands unless the user explicitly chooses them after `seal-finalize`.
- Do NOT describe platform reviewers as background agents or replace the live worker stack with a markdown table.
- Runtime output wins if this wrapper and the runtime disagree.

## Cross-Platform Drift Guard

If you change seal review, blocker handling, Porter delivery, or closeout behavior here, update `.aether/commands/seal.yaml`, both platform wrappers, `cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the same change.

Verify `aether command-guide seal --platform codex` still describes the same flow.
