---
source: shipped
name: aether-colony-build-cycle
description: Use when Codex is asked to plan, build, continue, swarm, or seal an Aether colony and must mirror wrapper orchestration safely
type: colony
domains: [aether, codex, planning, build, verification, swarm, seal, orchestration]
agent_roles: [queen, builder, watcher, scout, route_setter, tracker, archaeologist, auditor, probe]
workflow_triggers: [plan, build, continue, swarm, seal]
task_keywords: [aether plan, aether build, aether continue, aether swarm, aether seal, dispatch manifest, plan-only, finalize]
priority: high
version: "1.0"
---

# Aether Colony Build Cycle

## Purpose

Give Codex the wrapper-equivalent behavior for the lifecycle commands where AI
orchestration matters: `plan`, `build`, `continue`, `swarm`, and `seal`. Runtime JSON
manifests remain authoritative. Codex may spawn workers and summarize results,
but it must not invent state or write state files by hand.

For beginners: the runtime prints the recipe and owns the kitchen ledger. Codex
can coordinate helpers, but it must use the recipe the runtime gave it.

## Required First Step

Run or inspect the guide for the command being handled:

```bash
aether command-guide <plan|build|continue|swarm|seal> --platform codex
```

If this skill and `command-guide` disagree, follow `command-guide` and update
the skill.

## Raw Bypass

If the user explicitly says raw, exact, no orchestration, or "just run this
exact command", run the literal CLI command they provided. Say briefly that the
Codex orchestration layer was bypassed.

## Plan Flow

1. Select planning depth and decomposition depth unless arguments already make
   them clear.
2. Run `AETHER_OUTPUT_MODE=visual aether status`.
3. Run:

```bash
AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice>
```

4. Parse `result.plan_manifest` or `result.planning_manifest`. Never parse
   visual output as state.
5. If runtime reports unresolved clarifications, route to `aether discuss`
   unless the user explicitly approves continuing with assumptions.
6. Spawn the runtime-specified Scout and Route-Setter workers using manifest
   names, castes, task IDs, briefs, and `skill_section` values.
7. Pass each dispatch `brief` verbatim and enforce its read budget, no-repeat
   loop guard, output contract, and stop condition. If a planning worker keeps
   rereading the same file or command, mark it `blocked` with a concrete
   blocker instead of manually reconciling it as completed.
8. Include the Scout terminal result in the Route-Setter prompt so Route-Setter
   consumes Scout findings directly instead of re-running the survey.
9. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <worker completion JSON>
```

## Build Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Surface active REDIRECT, FOCUS, and FEEDBACK signals compactly.
3. Run:

```bash
AETHER_OUTPUT_MODE=json aether build <phase> --plan-only
```

4. Parse `result.dispatch_manifest`.
5. Follow the installed build-wave playbook. Use runtime-provided agent names,
   castes, task IDs, briefs, and skill sections.
6. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
7. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <worker completion JSON>
```

## Continue Flow

Default path:

```bash
AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard <args>
```

Use external review orchestration only when the user explicitly requested heavy
review or the runtime asks for wrapper-spawned review workers. In that case,
request the runtime manifest, spawn only the planned reviewers, collect results,
and finalize through `aether continue-finalize`.

## Swarm Flow

Watch mode stays direct:

```bash
AETHER_OUTPUT_MODE=visual aether swarm --watch
```

For bug-destroyer targets, use the external worker contract:

```bash
AETHER_OUTPUT_MODE=json aether swarm --plan-only <problem>
```

1. Parse `result.swarm_manifest`. Never parse visual output as state.
2. Preserve manifest wave order: investigation workers first, then builder,
   then watcher.
3. Use runtime-provided names, castes, roles, task IDs, briefs, and response
   contracts.
4. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
5. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <worker completion JSON>
```

## Seal Flow

1. Run `AETHER_OUTPUT_MODE=visual aether status`.
2. Run:

```bash
AETHER_OUTPUT_MODE=json aether seal --plan-only <args>
```

3. If the runtime returns blockers or recovery guidance, surface that and stop.
4. Parse `result.seal_manifest` and dispatch the Gatekeeper, Auditor, and Probe
   final-review workers through the host platform.
5. Use runtime-provided names, castes, task IDs, briefs, and skill sections.
6. Call `aether spawn-log` before each worker and `aether spawn-complete` after
   each terminal result.
7. Finalize through:

```bash
AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <worker completion JSON>
```

8. Follow runtime Porter readiness output only after `seal-finalize` succeeds.
   Do not run delivery commands unless the user chooses them.

## Guardrails

- Do not write `.aether/data/COLONY_STATE.json`, `session.json`, `CONTEXT.md`,
  `HANDOFF.md`, planning artifacts, or pheromone files by hand.
- Do not invent worker names, castes, task IDs, waves, or dispatches.
- Do not parse visual output for authoritative state. Use JSON mode for
  manifests.
- If Claude/OpenCode lifecycle wrapper behavior changes, update the matching
  YAML, this skill, and `cmd/command_guide.go` together.
