# Playbook: Plan

> Consolidated from plan-prep and plan-dispatch.

## Overview

The Queen prepares planning state, selects depth, and dispatches Scout and Route-Setter waves to generate a project phase plan.

## Stage 1: Prep

### Step 1: Status Check

Run `aether status` to verify colony is in a state that allows planning (not sealed, not executing).

Display brief context:
```
📋 Planning — Phase {current_phase}/{total_phases}
Goal: "{colony_goal}"
```

### Step 2: Planning Depth Selection (Phase Granularity)

Prompt for planning depth:
1. **Fast** — sprint granularity, 1-3 phases. Target confidence: 80, max iterations: 4.
2. **Balanced** — milestone granularity, 4-7 phases. Target confidence: 90, max iterations: 6. **Recommended default.**
3. **Deep** — quarter granularity, 8-12 phases. Target confidence: 95, max iterations: 8.
4. **Exhaustive** — major granularity, 13-20 phases. Target confidence: 99, max iterations: 12.

If `$ARGUMENTS` contains `--depth`, use it directly.

### Step 3: Planning Depth Selection (Task Decomposition)

Choose task decomposition depth:
1. **Light** — coarse tasks, 1-3 per plan with objective-level descriptions.
2. **Standard** — normal task breakdown. **Recommended default.**
3. **Deep** — granular subtasks including edge cases, error handling, and test coverage as separate tasks.

If `$ARGUMENTS` contains `--planning-depth`, use it directly.

### Step 4: Unresolved Clarifications Gate

Check for unresolved clarifications:
```bash
aether pending-decision-list | jq '.unresolved' 2>/dev/null || echo "0"
```

If unresolved exist, warn and route to `/ant-discuss` if user wants to resolve them first.

### Step 5: Request Plan Manifest

Run:
```bash
aether host plan --depth <choice> --planning-depth <choice2> --target <target> --max-iterations <iterations> $ARGUMENTS
```

Save full JSON envelope to temporary manifest file outside `.aether/data/`.

If `orchestrator_boundary_guidance` is active or next is `aether discuss`, stop and route to `/ant-discuss`.

## Stage 2: Dispatch

### Step 1: Render Spawn-Plan Ceremony

```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest_file>
```

### Step 2: Dispatch Scout Workers (Wave 1)

Refresh colony context via `aether colony-prime --compact`.

Render wave-start:
```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave 1
```

Spawn Scout workers using manifest `agent_name`. Use visible live Task panels (no `run_in_background`).

Per worker:
- Log spawn: `aether spawn-log --parent "Queen" --caste "scout" --name "{name}" --task "{task}" --depth 0`
- Description must be exactly: `{caste emoji} {Caste} {name}: {task}`
- Inject `skill_section` when present
- Pass each dispatch's `brief` verbatim under `Runtime Worker Brief`

### Step 3: Render Wave-Start for Wave 2

After Scout results return:
```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave 2
```

### Step 4: Dispatch Route-Setter Workers (Wave 2)

Spawn Route-Setter workers. Include Scout terminal results in prompts so they consume findings directly.

Per worker:
- Log spawn: `aether spawn-log --parent "Queen" --caste "route_setter" --name "{name}" --task "{task}" --depth 0`

For each terminal result:
1. Write result to temp JSON file
2. Render worker-complete ceremony:
   ```bash
   AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>
   ```
3. Call `aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`

If a worker keeps rereading the same file, mark `blocked` with concrete blocker.

### Step 5: Grounding Gate Check

After all workers complete, check for `grounding_warning` or `generic_plan_warning` from `plan-finalize`.

If warning present:
```
⚠ Grounding warning: {warning_message}

The plan may be too generic for this codebase. Consider re-planning with more specific targets.
```

This is a soft gate — proceed unless user wants to adjust.

### Step 6: Write Completion and Finalize

Collect terminal worker results into `${TMPDIR:-/tmp}/aether-plan-<run>/plan-completion.json`.

Run JSON finalizer:
```bash
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <file>
```

This lets the runtime write canonical planning artifacts and state.

### Step 7: Render Closeout Ceremony

After finalizer succeeds:
```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <file>
```

Summarize: selected depth, planning depth, phase count, confidence, `planning_loop.stop_reason`, and which planning agents ran.

Route first to `/ant-build 1` or runtime-surfaced next build command.
