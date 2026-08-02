<!-- Aether-managed: runtime spec at .aether/commands/build.yaml. Synced by aether update. -->
---
name: ant-build
description: "🔨 Build a phase — Queen dispatches workers, colony self-organizes"
---

<!-- PHASE-160: Fail Loudly merged first and made only narrow call-argument fixes to this file; it did not restructure it. PHASE-165 (Core Lifecycle Commands) is the sole structural owner of this file for milestone v1.25. PHASE-168 (Live Visibility) appends a visual-guidance trailer after this file lands. -->

🐜👑 You are the **Queen**. You DIRECTLY spawn multiple workers — do not delegate to a single Prime Worker. A single agent doing everything is not a colony; "justifications" for not spawning are not accepted.

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

The wrapper is the sole conductor for interactive worker spawning. The TS host provides the manifest only.

## Required Cross-Stage State

Carry these values forward across stages when produced:

- `phase_id`
- `verification_depth`
- `manifest_file`
- `context_capsule`
- `wave_results`
- `verification_status`
- `next_action`

<success_criteria>
A finished build has: every manifest dispatch spawned, every terminal result collected with a non-empty `handoff`, `build-finalize` run on the Go-owned completion path, and the closeout rendered.
</success_criteria>

<failure_modes>
- Wave failure mid-build: do not continue to the next wave; failed dependencies cascade.
- Provider dispatch unavailable: surface the sanitized availability message and stop.
- Boundary guidance active: route to `aether discuss` and request a fresh manifest.
</failure_modes>

<read_only>
This wrapper may read but never write: colony state, session files, and pheromone files. All persistence goes through the Go-owned finalizers named below.
</read_only>

## Colony Context

🐜 Before spawning anyone, the Queen grounds herself in what the colony already knows.

**Purpose:** Load current colony state and phase progress so framing and dispatch decisions are made against runtime truth, not memory.

**Reads:** `aether status` output (phase progress, colony health, active signals).

1. Run `AETHER_OUTPUT_MODE=visual aether status` to see current colony state, phase progress, and active signals.
2. Keep that runtime context in view while framing the phase.

**Stop conditions:** None — this stage only observes; it never blocks the build.

## Active Signals

🐜 The colony listens to its own pheromones before it moves.

**Purpose:** Surface REDIRECT/FOCUS/FEEDBACK signals as a compact steering block so the Queen and the user share the same constraints before workers spawn.

**Reads:** the active signal set already returned by `aether status`.

Present active pheromones as a compact steering block:

- `REDIRECT` first -- hard constraints.
- `FOCUS` second -- main attention areas.
- `FEEDBACK` last -- lightweight adjustments.
- Include strength or remaining-life context.
- If no active signals, say so plainly.

**Stop conditions:** None — signals inform framing; they never halt the build on their own.

## Phase Framing

🐜 Name the work before the colony moves on it.

**Purpose:** State what this build is, in one line a human can repeat back, before any worker spawns.

**Reads:** the phase name and total phase count from colony state.

Frame the requested work as `Phase N of M -- Name` with a one-line purpose.

**Stop conditions:** None.

## Dispatch Manifest

🐜 The manifest is the colony's marching order — fetch it, then spawn exactly what it names.

**Purpose:** Get the dispatch plan from the TS host without dispatching workers, so the wrapper — not the host — controls interactive spawning.

**Reads:** the TS host's plan-only build response.

Fetch the manifest from the TS host in plan-only mode:

```
aether host build --dry-run $ARGUMENTS
```

The TS host calls `aether build <phase> --plan-only` and returns JSON without dispatching workers. The wrapper is responsible for spawning from this manifest.

Parse `result.manifest.dispatch_manifest`. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.

See `.aether/docs/wrapper-host-contract.md` for the full field shapes this manifest carries.

**Stop conditions:** If provider dispatch is unavailable, surface only the Go-owned structured availability message: provider, sanitized cause, and next action. Do not include raw provider stdout, stderr, tokens, or auth probe output. Do not retry silently or fall back to a simulated dispatch.

## Guided Boundary Gate

🐜 Sometimes the colony must stop and ask before it moves.

**Purpose:** Catch orchestrator-level boundary guidance before any worker spawns, so a build never runs past a condition the runtime flagged as needing a decision.

**Reads:** `result.manifest.dispatch_manifest`'s `orchestrator_boundary_guidance` field.

Before spawning workers, inspect `result.manifest.dispatch_manifest` for `orchestrator_boundary_guidance`:

- If active or `next` is `aether discuss`, stop the build flow and route to `aether discuss`. Request a fresh manifest after resolution. Do not reuse the pre-discuss manifest. Rerun `after_discuss_next` after resolution.

**Stop conditions:** Boundary guidance active — route to `aether discuss` and request a fresh manifest; never proceed on the stale one.

## Runtime Spawn Ceremony

🐜 Before the first worker moves, the colony sees its own plan.

**Purpose:** Render the runtime-owned spawn plan so the user sees exactly what is about to be spawned, in the Go renderer's own caste-identity styling — never hand-rendered by the wrapper.

**Reads:** the manifest file written in Dispatch Manifest.

Render the runtime-owned spawn ceremony:

```
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>
```

**Stop conditions:** None — this stage only renders; the plan was already fixed in Dispatch Manifest.

## Worker Spawning

🐜 The Queen spawns directly. The colony requires actual parallelism.

**Why this matters:** the Queen spawns workers herself because a single agent doing everything is not a colony — it is one worker wearing a costume. Builders who verify their own work fall into confirmation bias; independent castes catch what a single agent misses. "Justifications" for not spawning are not accepted.

**Purpose:** Spawn every dispatch the manifest names, in wave order, carrying each worker's brief verbatim and collecting a concrete terminal result from every one.

**Reads:** `dispatch_manifest.execution_plan` and each dispatch's `brief` / `brief_path` / `context_capsule` / `skill_section` / `permission_profile`.

**Spawns:** every worker named in `dispatch_manifest.execution_plan`, wave by wave.

The wrapper spawns workers. The TS host does NOT dispatch workers for the interactive path.

For each step in `dispatch_manifest.execution_plan`, spawn matching dispatches:

- Use visible live Task/subagent calls. Do not set `run_in_background`.
- Each worker description: `{caste emoji} {Caste} {name}: {task}`.
- Each dispatch carries `brief` — the complete runtime-rendered worker prompt (phase objective, constraints, hints, success criteria, pheromone signals, survey paths, prior handoffs) — and, when present, `brief_path`: a repo-display path to a file holding the identical composed brief, byte for byte. Reading `brief_path` is the preferred channel — inline JSON briefs of 6-22KB are subject to Read-tool long-line truncation — but both carry the same bytes, so use either VERBATIM; never merge, summarize, or reconstruct. Read `dispatch_manifest.context_capsule` ONCE from the manifest — it is not per-dispatch data, reuse the same value for every worker this build spawns — and prepend it VERBATIM ahead of the brief, then append `dispatch.skill_section` when present. Do not summarize, reorder, or reconstruct any of it — the runtime already assembled it.
- Inspect and preserve each dispatch `permission_profile`. A `repository_read_only` worker must use a host-enforced no-write boundary. Reject `scoped_write` or `test_write` when the host cannot enforce it. `behavioral_restrictions` inside `workspace_write` are instructions, not a sandbox claim.
- Require terminal structured result with: `name`, `caste`, `stage`, `execution_wave`, `task_id`, `status`, `summary`, `files_created`, `files_modified`, `tests_written`, `blockers`, `duration`, `handoff`.
- The `handoff` object is mandatory for completed workers and must be concrete: `{changed_files, commands_run, verification_status, known_failures, open_decisions, assumptions, next_worker_instructions, do_not_repeat, freshness}` (freshness: RFC3339 timestamp of evidence collection, or `not-run`). It is what the next phase's workers receive as context — an empty handoff will be rejected by the finalizer.

Respect `execution_plan`: serial steps stay serial; parallel steps may spawn together.

For each manifest wave:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each worker.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. The worker's prompt = `dispatch_manifest.context_capsule` (read once, prepended verbatim) + the brief read VERBATIM from `dispatch.brief_path` when present (falling back to `dispatch.brief` inline when it is absent) + `dispatch.skill_section` when present. Nothing else, nothing invented.
6. After each worker returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
7. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --worker-file <worker_file>`.

**Stop conditions:** All workers in a wave fail — do not continue to the next wave; failed dependencies cascade into work built on broken foundations.

## Finalize

🐜 The colony's work becomes durable only through the Go runtime.

**Purpose:** Turn the wave results into one durable completion packet and hand it to the Go-owned finalizer — the wrapper never mutates state itself.

**Reads:** each worker's terminal structured result collected during Worker Spawning.

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

**Stop conditions:** `build-finalize` reports failure — do not render closeout as success; surface the runtime's own error instead.

## After the Build

🐜 A build that nobody looks at again was wasted effort.

**Purpose:** Hand the user back a clear picture of what moved and where to go next, using the runtime's own closeout as the source of truth.

**Reads:** the visual closeout rendered in Finalize.

1. Use the visual closeout's next-step line as the source of truth.
2. Summarize what moved forward and which workers/castes ran.
3. Note the most relevant signal or risk.
4. Guide the user first to `/ant-continue`.

**Stop conditions:** None — this stage only reports.

## Verification Depth

🐜 Not every phase needs the same amount of scrutiny.

**Purpose:** Explain the Queen's review-depth choice in plain terms so the user does not need to remember flag combinations.

**Reads:** the `--verification-depth` flag or the Queen's smart default for this phase.

The runtime supports `--verification-depth <light|standard|heavy>` for post-build review. Default is "standard". Use `--heavy` for full gates or `--light` to skip review agents.

**Stop conditions:** None.

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

<!-- PHASE-168: visual-guidance trailer appends below this line -->
