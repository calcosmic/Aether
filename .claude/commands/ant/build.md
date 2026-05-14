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

## Active Signals

Before spawning workers, present active pheromones as a compact steering block:

- `REDIRECT` first — hard constraints.
- `FOCUS` second — main attention areas.
- `FEEDBACK` last — lightweight adjustments.
- Include strength or remaining-life context.
- If no active signals, say so plainly.

## Phase Framing

Frame the requested work as `Phase N of M — Name` with a one-line purpose.

## Dispatch Manifest

Run the TS host to fetch the authoritative dispatch manifest:

```
aether host build $ARGUMENTS
```

If the TS host is unavailable, fall back to:

```
AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only
```

Parse `result.dispatch_manifest`. Save the JSON envelope to a temporary manifest file outside `.aether/data/`.

## Guided Boundary Gate

Before spawning workers, inspect `result.orchestrator_boundary_guidance`:

- If active or `next` is `aether discuss`, stop the build flow and route to `aether discuss`. Request a fresh manifest after resolution. Do not reuse the pre-discuss manifest.

## Worker Spawning

For each step in `dispatch_manifest.execution_plan`, spawn matching dispatches:

- Use visible live Task/subagent calls. Do not set `run_in_background`.
- Each worker description: `{caste emoji} {Caste} {name}: {task}`.
- Inject phase objective, task metadata, dependencies, success criteria, active signals, and `skill_section` when present.
- Require terminal structured result with: `name`, `caste`, `stage`, `execution_wave`, `task_id`, `status`, `summary`, `files_created`, `files_modified`, `tests_written`, `blockers`, `duration`.

Respect `execution_plan`: serial steps stay serial; parallel steps may spawn together.

## Finalize

After all workers return, collect results into a completion JSON and finalize:

```
AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file <completion_file>
```

Then render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <completion_file>
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

- Do NOT run `aether build` without `--plan-only` from this wrapper.
- Do NOT run `aether build --synthetic` after real agent workers complete.
- Do NOT describe parallel workers as background agents or say you will be notified later.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, or pheromone files.
- Do NOT parse visual output as authoritative state.
- Do NOT invent worker names, castes, or waves; use `dispatch_manifest`.
- If docs and runtime disagree, runtime wins.
