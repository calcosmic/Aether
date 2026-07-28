---
source: shipped
name: aether-colony-build-cycle
description: Use when Codex is asked to colonize, plan, build, continue, swarm, or seal an Aether colony and must mirror wrapper orchestration safely
type: colony
domains: [aether, codex, colonize, planning, build, verification, swarm, seal, orchestration]
agent_roles: [queen, builder, watcher, scout, route_setter, tracker, archaeologist, auditor, probe]
workflow_triggers: [colonize, plan, build, continue, swarm, seal]
task_keywords: [aether colonize, aether plan, aether build, aether continue, aether swarm, aether seal, dispatch manifest, plan-only, finalize]
priority: high
version: "1.0"
---

# Aether Colony Build Cycle

## Purpose

Give Codex the wrapper-equivalent behavior for the lifecycle commands where AI
orchestration matters: `colonize`, `plan`, `build`, `continue`, `swarm`, and `seal`. Runtime JSON
manifests remain authoritative. Codex may spawn workers and summarize results,
but it must not invent state or write state files by hand.

For beginners: the runtime prints the recipe and owns the kitchen ledger. Codex
can coordinate helpers, but it must use the recipe the runtime gave it.

## Required First Step

Run or inspect the guide for the command being handled:

```bash
aether command-guide <colonize|plan|build|continue|swarm|seal> --platform codex
```

If this skill and `command-guide` disagree, follow `command-guide` and update
the skill.

## Raw Bypass

If the user explicitly says raw, exact, no orchestration, or "just run this
exact command", run the literal CLI command they provided. Say briefly that the
Codex orchestration layer was bypassed.

## Live Worker Ceremony

For wrapper-orchestrated worker flows, the visible live agent stack is part of
the user experience. Spawn same-wave workers as visible Task/subagent panels
with caste-labelled descriptions. Do not use background-only dispatch as the
ceremony, do not say you will be notified later, and do not replace the live
stack with a markdown worker table.

## Guided Boundary Gate

For `plan`, `build`, heavy external-review `continue`, and `seal`, inspect
`result.orchestrator_boundary_guidance` and the matching manifest
`orchestrator_boundary_guidance` before any spawn ceremony, worker dispatch, or
finalizer packet. If guidance is active or `next` is `aether discuss`, stop the
lifecycle flow, show the guidance summary, route to `aether discuss`, and tell
the user to rerun `after_discuss_next` after the answer is resolved. Then request
a fresh host manifest; never reuse the pre-discuss manifest, and never ask,
answer, or store boundary questions in Codex chat or wrapper state.

## Plan Flow

1. If the colony already has completed phases and the user is revising future
   work, keep the existing goal and pass `--refresh`, `--revision-type`, and a
   concrete `--revision-reason` to every host-plan iteration. Research and
   verification revisions also require one or more repository-relative
   `--revision-evidence` files. Do not create a new colony just to replan.
2. Select planning depth and decomposition depth unless arguments already make
   them clear.
3. Run `AETHER_OUTPUT_MODE=visual aether status`.
4. Run the TS host manifest command for one planning iteration:

```bash
aether host plan --depth <choice> --planning-depth <choice>
```

5. Save the full JSON envelope to a temporary manifest file outside
   `.aether/data/`.
6. Parse `result.plan_manifest` or `result.planning_manifest`. Never parse
   visual output as state. Treat `planning_run_id`, `iteration`,
   `target_confidence`, `max_iterations`, `previous_confidence`,
   `selected_gaps`, `previous_plan_draft`, and `expected_workers` as
   authoritative loop state.
7. When the manifest includes `revision`, preserve it and the worker briefs
   verbatim. Completed phases are immutable, and Route-Setter must output only
   replacement unfinished phases; Go assigns their final phase and task IDs.
8. Apply the Guided Boundary Gate before rendering spawn ceremonies or spawning
   planning workers.
9. If runtime reports unresolved clarifications, route to `aether discuss`
   unless the user explicitly approves continuing with assumptions.
10. Render the runtime-owned spawn ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest file>
```

11. Spawn every runtime-specified dispatch: the base Scout plus any
   phase_research Scouts in wave 1 (parallel), then exactly one
   runtime-specified Route-Setter in wave 2, using visible live Task/subagent panels with
   caste-labelled descriptions, manifest names, castes, task IDs, briefs, and
   `skill_section` values. Do not add extra planning workers.
12. Before each manifest wave, render `aether ceremony wave-start` for that
   workflow and execution wave.
13. Pass each dispatch `brief` verbatim and enforce its read budget, no-repeat
   loop guard, output contract, and stop condition. If a planning worker keeps
   rereading the same file or command, mark it `blocked` with a concrete
   blocker instead of manually reconciling it as completed.
14. Include the Scout terminal result in the Route-Setter prompt so Route-Setter
   consumes Scout findings directly instead of re-running the survey. If the
   manifest includes `selected_gaps` or `previous_plan_draft`, require fresh
   evidence or resolved gaps before confidence may rise.
15. Call `aether spawn-log` before each planning worker and
   `aether spawn-complete` after each terminal result.
16. After each terminal result, render `aether ceremony worker-complete`.
17. Build the completion packet with `planning_run_id`, `iteration`, Scout
   `scout_report`, Route-Setter `phase_plan`, and a compact `source_summary`.
   Never reuse a manifest or completion packet across iterations. Finalize
   through:

```bash
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <worker completion JSON>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <worker completion JSON>
```

If the JSON finalizer returns `requires_next_iteration: true`, do not render
final closeout and do not claim the colony plan is complete. Request a fresh
`aether host plan` manifest with the same depth, planning depth, target, and
max-iteration controls, then repeat Scout -> Route-Setter -> `plan-finalize`.
Only `plan-finalize` may decide that the target was reached, the loop stalled,
the max iteration cap was hit, or explicit `--accept` finalized below target.
For a completed-prefix revision, report the accepted `plan_revision` and never
dispatch a task from the superseded revision.

## Colonize Flow

1. Run:

```bash
aether host colonize <args>
```

2. Save the full JSON envelope to a temporary manifest file outside
   `.aether/data/`.
3. Parse `result.colonize_manifest`. Never parse visual output as state.
4. Render the runtime-owned survey ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow colonize --manifest-file <manifest file>
```

5. Dispatch the runtime-specified Surveyor workers through the host platform
   with caste-labelled descriptions, runtime names, briefs, output paths, and
   skill sections.
6. Render `aether ceremony wave-start` before each surveyor wave.
7. Call `aether spawn-log` before each surveyor and `aether spawn-complete`
   after each terminal result.
8. After each terminal result, render `aether ceremony worker-complete`.
9. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <worker completion JSON>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow colonize --completion-file <worker completion JSON>
```

## Build Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Surface active REDIRECT, FOCUS, and FEEDBACK signals compactly.
3. Run the TS host manifest command:

```bash
aether host build --dry-run <phase>
```

4. Save the full JSON envelope to a temporary manifest file outside
   `.aether/data/`.
5. Parse `result.manifest.dispatch_manifest`.
6. Apply the Guided Boundary Gate before rendering spawn ceremonies or spawning
   build workers.
7. Render the user-facing spawn ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest file>
```

8. Follow the installed build-wave playbook. Use runtime-provided agent names,
   castes, task IDs, briefs, and skill sections.
9. Before each manifest wave, render `aether ceremony wave-start` for the build
   workflow and execution wave.
10. Spawn parallel waves as visible live Task/subagent panels with caste-labelled
   descriptions. Do not use background-only dispatch as the ceremony, and do not
   replace the live stack with a markdown worker table.
11. Enforce read cache discipline for every worker: pass runtime briefs verbatim,
   treat "File unchanged since last read" as an instruction to use earlier content,
   and mark workers `blocked` if they keep re-reading the same unchanged file.
12. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
13. After each terminal result, render `aether ceremony worker-complete`.
14. Stage the accepted completion packet in the Go-owned attempt journal:

```bash
AETHER_OUTPUT_MODE=json aether build-completion-stage <phase> --completion-file <worker completion JSON>
```

Parse `result.completion_path`. If the wrapper stops after this point, `aether resume`
must offer this exact packet rather than redispatching workers.

15. Finalize through the durable packet only:

```bash
AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <Go-owned completion_path>
```

## Continue Flow

Default path:

```bash
AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard <args>
```

Use external review orchestration only when the user explicitly requested
`--classic-ceremony`, heavy review, or the runtime asks for wrapper-spawned
review workers. In that case, request the runtime manifest:

```bash
aether host continue --dry-run --classic-ceremony <args>
```

Save the JSON manifest envelope to a temporary file, parse
`result.manifest.continue_manifest`, apply the Guided Boundary Gate before rendering
spawn ceremonies or spawning reviewers, and spawn only the planned reviewers as
visible live Task/subagent panels with caste-labelled descriptions. Use
`aether ceremony spawn-plan`, `aether ceremony wave-start`, and
`aether ceremony worker-complete` around the live reviewers. Call
`aether spawn-log` before each reviewer and `aether spawn-complete` after each
terminal result. Pass each reviewer brief verbatim; it contains read cache
discipline. If a reviewer keeps re-reading the same unchanged file or artifact,
mark it `blocked` with the missing context instead of waiting through another
loop. Collect results, finalize through `aether continue-finalize`, then render.

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <worker completion JSON>
```

## Swarm Flow

Watch mode stays direct:

```bash
AETHER_OUTPUT_MODE=visual aether swarm --watch
```

For bug-destroyer targets, use the external worker contract:

```bash
AETHER_OUTPUT_MODE=json aether swarm --plan-only <problem>
```

1. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.
2. Parse `result.swarm_manifest`. Never parse visual output as state.
3. Render the runtime-owned spawn ceremony with `aether ceremony spawn-plan`.
4. Preserve manifest wave order: investigation workers first, then builder,
   then watcher.
5. Use runtime-provided names, castes, roles, task IDs, briefs, and response
   contracts.
6. Render `aether ceremony wave-start` before each same-wave group.
7. Spawn each same-wave group as visible live Task/subagent panels with
   caste-labelled descriptions.
8. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
9. After each terminal result, render `aether ceremony worker-complete`.
10. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <worker completion JSON>
```

Then render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow swarm --completion-file <worker completion JSON>
```

## Seal Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Run:

```bash
aether host seal <args>
```

3. If the runtime returns blockers or recovery guidance, surface that and stop.
4. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.
5. Parse `result.seal_manifest`.
6. Apply the Guided Boundary Gate before rendering spawn ceremonies or spawning
   final-review workers.
7. Render the runtime-owned spawn ceremony with `aether ceremony spawn-plan`.
8. Use runtime-provided names, castes, task IDs, briefs, and skill sections.
9. Render `aether ceremony wave-start` before each final-review wave.
10. Spawn final-review workers as visible live Task/subagent panels with
   caste-labelled descriptions.
11. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
12. After each terminal result, render `aether ceremony worker-complete`.
13. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <worker completion JSON>
```

14. Render the wrapper closeout:

```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file <worker completion JSON>
```

15. Follow runtime Porter readiness output only after `seal-finalize` succeeds.
   Do not run delivery commands unless the user chooses them.

## Guardrails

- Do not write `.aether/data/COLONY_STATE.json`, `session.json`, `CONTEXT.md`,
  `HANDOFF.md`, planning artifacts, or pheromone files by hand.
- Do not invent worker names, castes, task IDs, waves, or dispatches.
- Preserve each manifest worker's typed `permission_profile`. Never broaden `repository_read_only`, and never present `behavioral_restrictions` under `workspace_write` as host-enforced isolation.
- Do not parse visual output for authoritative state. Use JSON mode for
  manifests.
- If Claude/OpenCode lifecycle wrapper behavior changes, update the matching
  YAML, this skill, and `cmd/command_guide.go` together.
