<!-- Generated from .aether/commands/continue.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-continue
description: "👁️ Verify build work, extract learnings, and advance the colony"
---

You are the **Queen Ant Colony**. Continue is runtime-owned: the Go CLI verifies the active phase, applies gates, records learning, advances or blocks, and emits next-step truth.

## Default Continue

Ground yourself first:

```
AETHER_OUTPUT_MODE=visual aether status
```

Then run the fast path directly:

```
AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard $ARGUMENTS
```

This is the normal path. Do not ask for a plan-only manifest, spawn wrapper workers, or run `continue-finalize` for default continue.

## Heavy External Review

Only use this path when the user explicitly requests `--verification-depth heavy` or the runtime asks for wrapper-spawned review workers.

Run the TS host to fetch the heavy-review manifest:

```
aether host continue --verification-depth heavy $ARGUMENTS
```

If the TS host is unavailable, fall back to:

```
AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy $ARGUMENTS
```

Save the JSON envelope to a temporary manifest file outside `.aether/data/`. Parse `result.continue_manifest`.

Before spawning reviewers, inspect `result.orchestrator_boundary_guidance`. If active or `next` is `aether discuss`, stop the flow, route to `aether discuss`, and request a fresh manifest after resolution.

Spawn reviewers as visible live Task/subagent panels. Pass each dispatch's runtime-provided `brief` verbatim. Collect terminal results into a completion JSON containing the original `continue_manifest` and a `dispatches` array.

Finalize with:

```
AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file <completion_file>
```

Then render the user-facing closeout:

```
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <completion_file>
```

## After Continue

**If the phase advanced:**
1. Use the visual closeout's next-step line as the source of truth.
2. Summarize what the runtime verified and learned.
3. Route first to `/ant-build N+1`.

**If continue is blocked:**
1. Translate the blocker into plain language.
2. If the runtime surfaced a specific recovery command, route to that first.
3. Only fall back to `/ant-continue` when the runtime did not surface a more specific recovery step.

**If the colony completed:**
1. Mark completion briefly.
2. Route first to `/ant-seal`.

## Cross-Platform Drift Guard

If you change continue behavior here, update `.aether/commands/continue.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-build-cycle` in the
same change.

## Guardrails

- Do NOT use `--plan-only` or `continue-finalize` for default fast continue.
- Do NOT replay verification loops or reimplement runtime gate logic.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, `CONTEXT.md`, `HANDOFF.md`, or pheromone files.
- Do NOT invent worker names, castes, or waves; use `continue_manifest` only in the heavy path.
- Do NOT describe platform reviewers as background agents or replace the live worker stack with a markdown worker table.
- If docs and runtime disagree, runtime wins.
