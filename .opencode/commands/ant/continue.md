<!-- Generated from .aether/commands/continue.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-continue
description: "👁️ Verify build work, extract learnings, and advance the colony"
---

You are the **Queen Ant Colony**. Continue is runtime-owned: the Go CLI verifies the active phase, applies gates, records learning, advances or blocks, and emits next-step truth.

Use the Go `aether` CLI as the source of truth.

## Ownership Split

| Concern | Owner |
|---------|-------|
| Manifest generation | TS host (`aether host continue --dry-run`) for heavy review |
| Reviewer spawning | Wrapper (platform Agent tool) for heavy review |
| Verification + gating | Go runtime (default continue command) |
| State mutation | Go runtime (`continue-finalize`) |

The wrapper is the sole conductor for interactive reviewer spawning in the heavy
review path. The TS host provides the manifest only. See
`.aether/docs/wrapper-host-contract.md`.

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

Only use this path when the user explicitly requests `--classic-ceremony`, `--verification-depth heavy`, or the runtime asks for wrapper-spawned review workers.

Fetch the manifest from the TS host in plan-only mode:

```
aether host continue --dry-run --classic-ceremony $ARGUMENTS
```

`aether host continue --dry-run --verification-depth heavy $ARGUMENTS` is equivalent for callers that already use depth flags.

The TS host calls `aether continue --plan-only --classic-ceremony` and returns JSON without dispatching workers. The wrapper is responsible for spawning reviewers from this manifest.

The TS host is the sole manifest authority. See `.aether/docs/wrapper-host-contract.md`.

Parse `result.manifest.continue_manifest`. Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.

Before spawning reviewers, inspect `result.manifest.continue_manifest` for `orchestrator_boundary_guidance`. If active or `next` is `aether discuss`, stop the flow, route to `aether discuss`, and request a fresh manifest after resolution. Rerun `after_discuss_next` after resolution.

Render the runtime-owned heavy-review ceremony:

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow continue --manifest-file <manifest_file>
```

Spawn reviewers as visible live Task/subagent panels. Do not set `run_in_background`. Pass each dispatch's runtime-provided `brief` verbatim.

For each heavy-review wave:

1. Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow continue --manifest-file <manifest_file> --execution-wave "<execution_wave>"`.
2. Run `AETHER_OUTPUT_MODE=json aether spawn-log --parent "Queen" --caste "<caste>" --name "<name>" --task "<task>" --depth 1` before each reviewer.
3. Spawn the matching platform agent using `agent_name` as the subagent type.
4. Use the exact visible description: `{caste emoji} {Caste} {name}: {task}`.
5. Pass each dispatch's runtime-provided `brief` verbatim.
6. After each reviewer returns, run `AETHER_OUTPUT_MODE=json aether spawn-complete --name "<name>" --status "<status>" --summary "<summary>"`.
7. Write that one terminal result to a temporary worker JSON file and render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow continue --worker-file <worker_file>`.

Collect terminal results into a completion JSON containing the original `continue_manifest` and a `dispatches` array.

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
same change. Verify `aether command-guide continue --platform codex` still describes the matching Codex flow.

## Guardrails

- Do NOT use `--plan-only` or `continue-finalize` for default fast continue.
- Do NOT run `aether host continue --classic-ceremony` without `--dry-run` from this wrapper; that triggers the TS host dispatched path which duplicates the wrapper's own reviewer spawning. Always use `aether host continue --dry-run`.
- Do NOT replay verification loops or reimplement runtime gate logic.
- Do NOT read or write colony state files by hand.
- Do NOT mutate `COLONY_STATE.json`, `session.json`, `CONTEXT.md`, `HANDOFF.md`, or pheromone files.
- Do NOT invent worker names, castes, or waves; use `continue_manifest` only in the heavy path.
- Do NOT describe platform reviewers as background agents or replace the live worker stack with a markdown worker table.
- If docs and runtime disagree, runtime wins.
