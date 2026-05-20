# plan -- Lifecycle Contract

**Last verified:** 2026-05-16
**Source files:** cmd/codex_plan.go, cmd/codex_plan_finalize.go, cmd/codex_workflow_cmds.go

## Lifecycle

`plan` has separate manifest, host-dispatch, and finalization steps.

1. `aether plan --plan-only` and `aether host plan` emit a planning dispatch manifest. They do not prove that workers ran.
2. The Codex or host layer dispatches the manifest's Scout and Route-Setter workers outside the `plan` command.
3. `aether plan-finalize --completion-file <file>` validates the manifest and terminal worker results, then writes the canonical plan state.

Direct `aether plan` may still run Go-owned local planning, but host/wrapper orchestration must follow the manifest and finalizer contract above.

## Inputs

### `aether plan` Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --refresh | bool | no | false | Regenerate the plan even when an existing plan is present |
| --force | bool | no | false | Alias for --refresh |
| --plan-only | bool | no | false | Emit a dispatch manifest for host orchestration |
| --depth | string | no | "" | Planning depth: fast, balanced, deep, or exhaustive |
| --planning-depth | string | no | "" | Task decomposition depth: light, standard, or deep |
| --verification-depth | string | no | "" | Verification depth: light, standard, or heavy |
| --target | int | no | depth preset | Planning confidence target, clamped to 70-99 |
| --max-iterations | int | no | depth preset | Planning loop budget, clamped to 2-12 |
| --accept | bool | no | false | Accept the current best plan even if confidence remains below target |
| --synthetic | bool | no | false | Use local synthesis behavior instead of external worker completion |
| --worker-timeout | duration | no | 0 | Include a per-worker timeout in planning dispatch contracts |

### `aether plan-finalize` Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --completion-file | path | yes | "" | JSON completion packet containing `plan_manifest` and terminal worker results |

### Arguments

None.

## Outputs

### Stdout

All paths return a JSON envelope through `outputWorkflow`.

| Path | Structured result |
|------|-------------------|
| New `--plan-only` / `host plan` | `plan_only: true`, `existing_plan: false`, `dispatch_mode: "plan-only"` or `"agent-delegate"`, `requires_finalizer: true`, `dispatches`, `plan_manifest`, and `planning_manifest` |
| Existing plan without refresh | `plan_only: true`, `existing_plan: true`, `requires_finalizer: false`, existing `phases`, `count`, and `next` build command |
| `plan-finalize` | Final `phases`, `confidence`, planning artifact paths, terminal `dispatches`, `dispatch_mode: "external-task"`, and next build command |

Both manifest and finalizer outputs include `planning_loop` with
`target_confidence`, `max_iterations`, `iterations`, `stop_reason`, and final
confidence evidence. Stop reasons are `target_reached`, `stalled`,
`max_iterations`, or `accepted`.

### Files Created/Modified

| Path | Operation | When |
|------|-----------|------|
| .aether/data/pending-decisions.json | create/update | Only in Orchestrator mode when plan materializes boundary questions |
| .aether/data/planning/SCOUT.md | create/update | `plan-finalize` after validation |
| .aether/data/planning/ROUTE-SETTER.md | create/update | `plan-finalize` after validation |
| .aether/data/planning/phase-plan.json | create/update | `plan-finalize` after validation |
| .aether/data/phase-research/ | recreate | `plan-finalize` after validation |
| .aether/data/COLONY_STATE.json | update | `plan-finalize` writes phases, current phase, granularity, confidence, and events |
| .aether/data/spawn-tree.txt | append/update | `plan-finalize` records external planning worker results |
| .aether/data/spawn-runs.json | update | `plan-finalize` records runtime spawn-run status |
| .aether/data/session.json | update | `plan-finalize` updates the next-command session summary |

`aether plan --plan-only` must not create `.aether/data/planning/`, `.aether/data/phase-research/`, spawn-tree records, session summaries, spawn-run records, `.aether/CONTEXT.md`, `.aether/HANDOFF.md`, or plan fields in `.aether/data/COLONY_STATE.json`.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Colony not initialized, missing active goal, invalid manifest/completion evidence, stale workspace, or planning failure |

## State Mutations

### Manifest Step

`aether plan --plan-only` and `aether host plan` are planning-intent surfaces. Dispatch entries with `status: "planned"` describe work the host should run later. They are not completion evidence.

When an existing plan is present and no refresh is requested, the result is a no-finalizer response: `existing_plan: true`, `requires_finalizer: false`, and no `plan_manifest` or `planning_manifest`. Renderers should report the existing plan and next command only. They must not claim Scout or Route-Setter execution, render synthetic worker-complete ceremony, or ask the host to call `plan-finalize`.

### Host Dispatch Step

The host dispatches the manifest's Scout and Route-Setter workers externally. `planned` and `spawned` mean pending or active work; they must not be persisted, rendered, or summarized as completed worker results.

Only terminal worker evidence, such as completed or failed worker result JSON, may be passed to `plan-finalize` as execution evidence.

### Finalizer Step

`aether plan-finalize --completion-file <file>` is the state-mutating host-planning step. Before writing state, it validates:

- the completion file includes a `plan_manifest`
- the manifest came from `plan-only` or `agent-delegate` mode and has `requires_finalizer: true`
- the manifest root, goal, colony mode, granularity, freshness, and workspace still match
- Scout and Route-Setter results are terminal and complete
- the Route-Setter phase plan is valid and not stale pre-existing evidence
- planning-loop stop evidence is computed by the Go finalizer from the accepted
  confidence and manifest loop controls

After validation, the finalizer writes canonical planning artifacts, updates `.aether/data/COLONY_STATE.json`, records spawn/run metadata, emits completion ceremony, and updates session summary.

## Orchestrator Boundary Questions

In Orchestrator colony mode, `plan --plan-only` may materialize at most one planning-scope boundary question in `.aether/data/pending-decisions.json` and add `orchestrator_boundary_guidance` that routes `next` to `aether discuss`. This clarification side effect is not worker execution and does not make planned or spawned dispatches completed.

Outside Orchestrator mode, boundary-question fields should be empty/no-op.

## Preconditions

- Colony state exists in `.aether/data/COLONY_STATE.json`
- The colony has an active goal
- Store initialization succeeds
- `plan-finalize` receives a fresh completion file produced from the current manifest and workspace
