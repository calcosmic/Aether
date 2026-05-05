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

Expected manifest:

- `dispatch_mode`: `plan-only` or `agent-delegate`
- `requires_finalizer`: `true`
- `dispatches`: Gatekeeper, Auditor, and Probe final-review workers
- `finalizer_command`: `AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>`

## Worker Dispatch

Dispatch the runtime-provided workers through the host platform in manifest wave order.

For each dispatch:

1. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1`.
2. Spawn the host agent using `agent_name` as the subagent type.
3. Give the worker the exact `brief` from the manifest.
4. Tell the worker this is final review before seal and it must not modify repo source files.
5. Collect a terminal result with:
   - `name`
   - `caste`
   - `stage`
   - `wave`
   - `task_id`
   - `status`
   - `summary`
   - `blockers`
   - `report`
6. Run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.

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

Branch strictly on `seal-finalize` output:

- If blocked, report the runtime blocker text and stop.
- If sealed, summarize the workers and the runtime seal result.
- Follow the runtime's Porter readiness output.

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
- Runtime output wins if this wrapper and the runtime disagree.

## Cross-Platform Drift Guard

If you change seal review, blocker handling, Porter delivery, or closeout behavior here, update `.aether/commands/seal.yaml`, both platform wrappers, `cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the same change.

Verify `aether command-guide seal --platform codex` still describes the same flow.
