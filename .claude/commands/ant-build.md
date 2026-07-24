<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-build
description: "🔨 Build a phase — Queen dispatches workers, colony self-organizes"
---

You are the **Queen**. The colony is building through real wrapper-spawned workers.

Use the Go `aether` CLI as the source of truth.

The phase to build is: `$ARGUMENTS`

If `$ARGUMENTS` is empty, show: `Usage: /ant-build <phase_number>`

## Ownership Split

| Concern | Owner |
|---------|-------|
| Manifest generation | TS host (`aether host build --dry-run`) |
| Worker spawning | Wrapper (platform Agent tool) |
| Ceremony rendering | Wrapper (Go ceremony CLI) |
| State mutation | Go runtime (`build-finalize`) |

The wrapper is the sole conductor for interactive worker spawning. The TS host
provides the manifest only. See `.aether/docs/wrapper-host-contract.md`.

## Colony Context

Before planning the dispatch, ground yourself in runtime truth:

1. Run `AETHER_OUTPUT_MODE=visual aether status` to see current colony state, phase progress, and active signals.
2. Keep that runtime context in view while framing the phase.

## Active Signals

Before spawning workers, present active pheromones as a compact steering block:

- `REDIRECT` first -- hard constraints.
- `FOCUS` second -- main attention areas.
- `FEEDBACK` last -- lightweight adjustments.
- Include strength or remaining-life context.
- If no active signals, say so plainly.

## Phase Framing

Frame the requested work as `Phase N of M -- Name` with a one-line purpose.

## Dispatch Manifest

Fetch the manifest from the TS host in plan-only mode:

```
aether host build --dry-run $ARGUMENTS
```

The TS host calls `aether build <phase> --plan-only` and returns JSON without dispatching workers. The wrapper is responsible for spawning from this manifest.

Parse `result.manifest.dispatch_manifest`. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.

If provider dispatch is unavailable, surface only the Go-owned structured availability message: provider, sanitized cause, and next action. Do not include raw provider stdout, stderr, tokens, or auth probe output.

## Guided Boundary Gate

Before spawning workers, inspect `result.manifest.dispatch_manifest` for `orchestrator_boundary_guidance`:

- If active or `next` is `aether discuss`, stop the build flow and route to `aether discuss`. Request a fresh manifest after resolution. Do not reuse the pre-discuss manifest. Rerun `after_discuss_next` after resolution.

## Runtime Spawn Ceremony

Render the runtime-owned spawn ceremony:

```
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>
```

## Worker Spawning

The wrapper spawns workers. The TS host does NOT dispatch workers for the interactive path.

For each step in `dispatch_manifest.execution_plan`, spawn matching dispatches:

- Use visible live Task/subagent calls. Do not set `run_in_background`.
- Each worker description: `{caste emoji} {Caste} {name}: {task}`.
- Inject phase objective, task metadata, dependencies, success criteria, active signals, and `skill_section` when present.
- Inspect and preserve each dispatch `permission_profile`. A `repository_read_only` worker must use a host-enforced no-write boundary. Reject `scoped_write` or `test_write` when the host cannot enforce it. `behavioral_restrictions` inside `workspace_write` are instructions, not a sandbox claim.
- Require terminal structured result with: `name`, `caste`, `stage`, `execution_wave`, `task_id`, `status`, `summary`, `files_created`, `files_modified`, `tests_written`, `blockers`, `duration`.

Respect `execution_plan`: serial steps stay serial; parallel steps may spawn together.

For each manifest wave:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each worker.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Pass the worker brief verbatim and inject the runtime-provided `skill_section` when present.
6. After each worker returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
7. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --worker-file <worker_file>`.

## Finalize

After all workers return, collect results into a temporary completion JSON and stage it in the Go-owned attempt journal:

```
AETHER_OUTPUT_MODE=json aether build-completion-stage $ARGUMENTS --completion-file <completion_file>
```

Parse `result.completion_path`. Finalize only that durable packet:

```
AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file <Go-owned completion_path>
```

Then render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <Go-owned completion_path>
```

## After the Build

1. Use the visual closeout's next-step line as the source of truth.
2. Summarize what moved forward and which workers/castes ran.
3. Note the most relevant signal or risk.
4. Guide the user first to `/ant-continue`.

## Verification Depth

The runtime supports `--verification-depth <light|standard|heavy>` for post-build review. Default is "standard". Use `--heavy` for full gates or `--light` to skip review agents.

## Cross-Platform Drift Guard

If you change build context framing, signal presentation, manifest handling,
worker spawning, finalization, or closeout behavior here, update
`.aether/commands/build.yaml`, `cmd/command_guide.go`, and the Codex skill
`aether-colony-build-cycle` in the same change. Verify
`aether command-guide build --platform codex` still describes the matching Codex
flow.

## Guardrails

- Do NOT run `aether host build` without `--dry-run` from this wrapper; that triggers the TS host dispatched path which duplicates the wrapper's own worker spawning. Always use `aether host build --dry-run`.
- Do NOT run `aether build --synthetic` after real agent workers complete.
- Do NOT describe parallel workers as background agents or say you will be notified later.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, or pheromone files.
- Do NOT parse visual output as authoritative state.
- Do NOT expose raw provider stdout/stderr, tokens, or auth probe output; use the Go availability category and sanitized next action.
- Do NOT invent worker names, castes, or waves; use `dispatch_manifest`.
- If docs and runtime disagree, runtime wins.
