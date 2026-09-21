<!-- Aether-managed: runtime spec at .aether/commands/seal.yaml. Synced by aether update. -->
---
name: ant-seal
description: "🏺 Seal the colony with visible final review workers"
---

You are the **Queen**. Seal the colony through the runtime manifest/finalizer contract.

Use the Go `aether` CLI as the source of truth. The wrapper only dispatches host-platform agents and reports their terminal results back to the runtime.

## Closure Contract

Close a verified colony, or explicitly record an owner-forced incomplete closure.
Force flags pass only when directly supplied by the owner. The Go runtime owns final review, preflight, confirmation, transaction, and rendering. A forced-incomplete closure is not verified success.

The wrapper never offers, constructs, or reruns a force command. If an owner directly supplies force flags to the runtime, preserve them verbatim; never invent the
reason. Do not ask to Force the seal from wrapper guidance.

## Raw Bypass

If the user explicitly asks for raw, exact, direct, or no-orchestration seal, run:

```bash
AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS
```

Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.

Otherwise use the hosted review flow below.

## Seal Manifest

Run:

```bash
aether host seal $ARGUMENTS
```

Parse `result.seal_manifest`. If the runtime returns blockers or recovery guidance, surface that output and stop. Do not fabricate review results.

**Force-seal (owner override, runtime-only):** the runtime alone validates a
direct owner force request and records its reason. The wrapper does not create
force authority, offer an override choice, or construct force flags. NEVER add `--force` on your own initiative, and never invent the
reason. AskUserQuestion remains available only for the runtime's explicit
owner-confirmation question.

Save the full JSON envelope to a temporary manifest file outside `.aether/data/`. The ceremony commands read that file so the final-review display uses the same runtime manifest.

Expected manifest:

- `dispatch_mode`: `plan-only` or `agent-delegate`
- `requires_finalizer`: `true`
- `dispatches`: the program's own choice of reviewers for this project — picked from the project's own progress and how deep the check runs, never a fixed list; a security reviewer joins only once the review reaches its deepest setting, which happens automatically after the project shipped real production work (or when the final stage of work is itself clearly security-related); the exact reviewers for this run are named in the plan the command prints before it starts
- `finalizer_command`: `AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>`

## Guided Boundary Gate

Before rendering spawn ceremonies or spawning final-review workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`.

- If it is active or `next` is `aether discuss`, stop the seal flow and show its summary.
- Route to `aether discuss` so the user can resolve the runtime-owned questions.
- Tell the user to rerun `after_discuss_next` after answers are resolved.
- Request a fresh plan-only manifest after the guided answer is resolved. Do not reuse the pre-discuss manifest and do not ask, answer, or store boundary questions in wrapper markdown.

## Runtime Spawn Ceremony

Before spawning final-review workers, render the runtime-owned old-style seal ceremony:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow seal --manifest-file <manifest_file>
```

Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. This output is display-only; do not parse it as state. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.

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

1. Render `AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow seal --manifest-file <manifest_file> --execution-wave "<execution_wave>"`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
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
   - optional `findings` or `issues` objects shaped as `{domain,severity,file,line,category,description,suggestion,blocking}`
   - optional `recommendations`, `weak_spots`, `edge_cases_discovered`, and `reusable_lessons`
8. Run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
9. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow seal --worker-file <worker_file>`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.

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
      "report": "...",
      "findings": [
        {
          "domain": "security",
          "severity": "LOW",
          "category": "release-integrity",
          "description": "Release provenance should be signed before public distribution.",
          "suggestion": "Add signed provenance in the next release hardening pass.",
          "blocking": false
        }
      ],
      "reusable_lessons": ["Keep release provenance checks in the final seal review."]
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

Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.

Branch strictly on `seal-finalize` output:

- If blocked, report the runtime blocker text and stop.
- If `result.awaiting_owner_confirmation` is `true`, the project is NOT finished yet — see "Before Finishing" below.
- If sealed, use the visual closeout's next-step line as the source of truth.
- Summarize the workers and the runtime seal result.
- Follow the runtime's Porter readiness output in visual mode.

## Before Finishing: The Owner Is Always Asked

Both the raw-bypass `aether seal` command and `aether seal-finalize` stop and ask before
the project is actually marked finished. "Finishing" a project means writing a summary
document, filing the project away, and pooling its lessons into the shared store other
projects read — never assume the user already understands that word.

1. The runtime first prints a short state-of-play card: how many phases are done out of
   the total, anything still failing, any open warnings, and what finishing will actually
   do.
2. It then runs the "what did we learn" review and shows what it found — this runs, and
   its lessons are kept, even if the user goes on to say no.
3. It asks one question: `Finish this project?` — or, when something is still failing or
   unresolved, a second, more specific question naming exactly what: `Finish anyway with
   N check(s) failing: <problem>, <problem>?`.

If the JSON result carries `"awaiting_owner_confirmation": true`, the project is NOT
finished. Ask the user the exact question in `result.question` (the AskUserQuestion
tool). Relay their answer with the exact command in `result.next` — never type that
command on your own initiative, and never infer a "yes" from anything else the user said.
Only after that command reports the answer as recorded should you rerun `aether seal` (or
`aether seal-finalize` with the same completion file) to actually finish.

Autopilot never seals a project. It stops at the explicit seal boundary and
leaves final review, confirmation, and any owner-supplied force request to the
Go runtime.

## Post-Seal Delivery

Do not run delivery commands automatically. If the runtime says the colony is sealed and shows Porter readiness, ask the user which delivery actions to perform:

- publish to hub
- push to git remote
- create GitHub release
- skip delivery for now

Run selected delivery actions sequentially and stop on first failure.

## Post-Seal Review

After sealing, run `AETHER_OUTPUT_MODE=visual aether status` first to review the retained sealed state; `aether entomb` is a separate optional owner-confirmed archive-and-clear action. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
Never invoke entomb automatically; sealing retains active state for owner review.

## Guardrails

- Do NOT write colony state files, session files, review reports, pheromone files, or archive files by hand.
- Do NOT parse visual output as truth; use JSON output for programmatic data.
- Do NOT bypass `aether host seal` from this wrapper unless the user explicitly asks for raw/no-orchestration.
- Do NOT run Porter delivery commands unless the user explicitly chooses them after `seal-finalize`.
- Do NOT describe platform reviewers as background agents or replace the live worker stack with a markdown table.
- Do NOT drop structured reviewer findings; `seal-finalize` persists them to final-review.json, review ledgers, the post-seal backlog, and QUEEN.md lessons when supplied.
- Do NOT type the `aether decision-answer` finishing command on your own initiative, and NEVER infer a "yes" from anything other than the user explicitly answering the printed question.
- Runtime output wins if this wrapper and the runtime disagree.

## Cross-Platform Drift Guard

If you change seal review, blocker handling, Porter delivery, or closeout behavior here, update `.aether/commands/seal.yaml`, both platform wrappers, `cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the same change.

Verify `aether command-guide seal --platform codex` still describes the same flow.
