# plan -- Lifecycle Contract

**Last verified:** 2026-07-22
**Source files:** cmd/codex_plan.go, cmd/codex_plan_finalize.go, cmd/plan_revision.go, cmd/codex_workflow_cmds.go

## Lifecycle

`plan` has separate manifest, host-dispatch, and finalization steps.

1. `aether plan --plan-only` and `aether host plan` emit a planning dispatch manifest for one iteration. They do not prove that workers ran.
2. The Codex or host layer dispatches the manifest's Scout, then Route-Setter, outside the `plan` command.
3. `aether plan-finalize --completion-file <file>` validates the manifest and terminal worker results. It writes the canonical plan state only when the runtime stop condition is reached; otherwise it records inspectable iteration state and asks the host to request the next manifest.

Direct `aether plan` may still run Go-owned local planning, but host/wrapper orchestration must follow the manifest and finalizer contract above.

## Inputs

### `aether plan` Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --refresh | bool | no | false | Regenerate the plan even when an existing plan is present |
| --force | bool | no | false | Alias for --refresh |
| --plan-only | bool | no | false | Emit a dispatch manifest for host orchestration |
| --repair-artifact | bool | no | false | Repair and validate dependency references in `.aether/data/planning/phase-plan.json` without rerunning workers |
| --depth | string | no | "" | Planning depth: fast, balanced, deep, or exhaustive |
| --planning-depth | string | no | "" | Task decomposition depth: light, standard, or deep |
| --verification-depth | string | no | "" | Verification depth: light, standard, or heavy |
| --target | int | no | depth preset | Planning confidence target, clamped to 70-99 |
| --max-iterations | int | no | depth preset | Planning loop budget, clamped to 2-12 |
| --accept | bool | no | false | Accept the current best plan even if confidence remains below target |
| --revision-type | string | with completed-phase refresh | manual | Why future work is changing: manual, user_feedback, research, verification_failure, or scope_change |
| --revision-reason | string | with completed-phase refresh | "" | Traceable explanation for changing unfinished work |
| --revision-evidence | path[] | research/verification revision | [] | Repository-relative source artifact; repeatable |
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
| New `--plan-only` / `host plan` | `plan_only: true`, `existing_plan: false`, `dispatch_mode: "plan-only"` or `"agent-delegate"`, `requires_finalizer: true`, `dispatches`, `plan_manifest`, and `planning_manifest`. The manifest includes `planning_run_id`, `iteration`, `target_confidence`, `max_iterations`, `previous_confidence`, `selected_gaps`, `previous_plan_draft`, and `expected_workers`. |
| Existing plan without refresh | `plan_only: true`, `existing_plan: true`, `requires_finalizer: false`, existing `phases`, `count`, and `next` build command |
| Pending `plan-finalize` iteration | `planned: false`, `iteration_completed: true`, `requires_next_iteration: true`, `planning_loop.stop_reason: "pending"`, `selected_gaps`, `evidence_hash`, and next `aether host plan ...` command |
| `plan-finalize` | Final `phases`, `confidence`, accepted `plan_revision`, planning artifact paths, terminal `dispatches`, `dispatch_mode: "external-task"`, and next build command |
| `plan --repair-artifact` | `repaired`, `repairs`, `validated`, `phase_plan`, `phase_count`, `task_count`, and next finalizer command |

Both manifest and finalizer outputs include `planning_loop` with
`target_confidence`, `max_iterations`, `iterations`, `stop_reason`, and final
confidence evidence. Stop reasons are `target_reached`, `stalled`,
`max_iterations`, `accepted`, or `pending` for an intermediate iteration that
must not be treated as a completed colony plan.

### Files Created/Modified

| Path | Operation | When |
|------|-----------|------|
| .aether/data/pending-decisions.json | create/update | Only in Orchestrator mode when plan materializes boundary questions |
| .aether/data/planning/iteration-state.json | create/update | `plan-finalize` when a valid iteration is below target and requires another manifest |
| .aether/data/planning/iterations/ | create/update | `plan-finalize` stores inspectable per-iteration Scout and phase-plan evidence |
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

The host dispatches the manifest's exactly one Scout and exactly one Route-Setter externally, preserving that order. `planned` and `spawned` mean pending or active work; they must not be persisted, rendered, or summarized as completed worker results.

Only terminal worker evidence, such as completed or failed worker result JSON, may be passed to `plan-finalize` as execution evidence.

Completion packets must include the current `planning_run_id`, `iteration`, Scout evidence (`scout_report`), Route-Setter draft (`phase_plan` with confidence dimensions and unresolved gaps), and a compact source summary. If an `evidence_hash` is supplied, Go verifies it against the Scout and Route-Setter evidence; either way Go computes the authoritative evidence hash.

### Finalizer Step

`aether plan-finalize --completion-file <file>` is the state-mutating host-planning step. Before writing state, it validates:

- the completion file includes a `plan_manifest`
- the manifest came from `plan-only` or `agent-delegate` mode and has `requires_finalizer: true`
- the manifest root, goal, colony mode, granularity, freshness, and workspace still match
- `base_revision_id` and `base_plan_state_hash` still match the active canonical plan
- each revision evidence file still exists inside the repository and its content hash matches the dispatched manifest
- the manifest has exactly one expected Scout followed by exactly one expected Route-Setter
- Scout and Route-Setter results are terminal and complete
- Scout evidence and the Route-Setter phase plan are present
- the Route-Setter phase plan is valid and not stale pre-existing evidence
- completion packets are not reused for a planning iteration
- confidence does not change without changed Scout/Route-Setter evidence or resolved gaps
- every task dependency references a known runtime task id and the dependency graph has no cycles
- planning-loop stop evidence is computed by the Go finalizer from the accepted
  confidence and manifest loop controls

After validation, if the stop reason is `pending`, the finalizer writes only inspectable planning iteration state and returns `requires_next_iteration: true`. It does not write a completed colony plan. If the stop reason is `target_reached`, `stalled`, `max_iterations`, or `accepted`, the finalizer writes canonical planning artifacts, updates `.aether/data/COLONY_STATE.json`, records spawn/run metadata, emits completion ceremony, clears intermediate iteration state, and updates session summary.

For `--refresh` after completed work, the finalizer performs one atomic revision transaction. Completed phases and their task/evidence state remain byte-equivalent snapshots with the same IDs. Only the unfinished suffix is replaced, and its local task dependencies are offset behind the immutable prefix. `COLONY_STATE.json` records the active revision ID, parent, reason, evidence paths and input-content hash, planning evidence hash, plan hash, full immutable snapshot, and preserved/superseded/replacement phase IDs. If validation or the atomic write fails, the prior active plan remains canonical.

### `phase-plan.json` Schema

Route-Setter writes the machine plan artifact at `.aether/data/planning/phase-plan.json`:

```json
{
  "phases": [
    {
      "name": "Phase name",
      "description": "Phase objective",
      "tasks": [
        {
          "goal": "Concrete task outcome",
          "constraints": [],
          "hints": [],
          "success_criteria": [],
          "evidence_requirements": [],
          "depends_on": ["1.1"]
        }
      ],
      "success_criteria": [],
      "evidence_requirements": []
    }
  ],
  "confidence": {
    "knowledge": 0,
    "requirements": 0,
    "risks": 0,
    "dependencies": 0,
    "effort": 0,
    "overall": 0
  },
  "gaps": []
}
```

Do not include task id fields in the artifact. Aether assigns task ids from the task's array position after ignoring empty-goal tasks: first task in phase 1 is `1.1`, second task in phase 1 is `1.2`, first task in phase 2 is `2.1`.

`depends_on` must be an array of those runtime task ids only. Do not use task text, file paths, descriptions, or custom ids such as `P1-T1`. The finalizer rejects invalid text dependencies with an actionable error. Common custom aliases such as `P1-T1` are normalized to `1.1` only when the target runtime task exists; validation is not weakened.

`evidence_requirements` optionally binds each success criterion to exact repository-relative `artifacts` and/or named `checks`. Supported checks are `build`, `types`, `lint`, `tests`, `claims`, and `watcher`. Once one requirement is present, every phase-level and task-level criterion in that phase must be bound. Runtime state under `.aether/data` cannot be used as product evidence.

If the worker artifact is nearly valid but has repairable dependency aliases, run:

```bash
aether plan --repair-artifact
```

Then rerun `aether plan-finalize --completion-file <file>` with the same completion packet.

## Orchestrator Boundary Questions

In Orchestrator colony mode, `plan --plan-only` may materialize at most one planning-scope boundary question in `.aether/data/pending-decisions.json` and add `orchestrator_boundary_guidance` that routes `next` to `aether discuss`. This clarification side effect is not worker execution and does not make planned or spawned dispatches completed.

Outside Orchestrator mode, boundary-question fields should be empty/no-op.

## Preconditions

- Colony state exists in `.aether/data/COLONY_STATE.json`
- The colony has an active goal
- Store initialization succeeds
- `plan-finalize` receives a fresh completion file produced from the current manifest and workspace
