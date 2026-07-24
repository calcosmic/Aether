---
name: ant:plan-dispatch
description: "Plan dispatch: Scout wave, Route-Setter wave, grounding gate, finalize"
---

### Step 1: Render Spawn-Plan Ceremony

Render the planning spawn ceremony:
```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest_file>
```

### Step 2: Dispatch Scout Workers (Wave 1)

Before dispatching, refresh colony context so new pheromones/memory are visible:
```bash
prime_result=$(aether colony-prime --compact 2>/dev/null)
```

Render wave-start:
```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave 1
```

Spawn Scout workers using the manifest's `agent_name`. Use visible live Task panels:
- Do not set `run_in_background`
- Do not describe workers as background agents
- Do not replace the live stack with a markdown worker table

For each Scout worker:
```bash
aether spawn-log --parent "Queen" --caste "scout" --name "{name}" --task "{task}" --depth 0
```

Each worker description must be exactly caste-labelled from the manifest: `{caste emoji} {Caste} {name}: {task}`.

Inject each dispatch's `skill_section` when present. Pass each dispatch's `brief` verbatim under a `Runtime Worker Brief` heading.

### Step 3: Render Wave-Start for Wave 2

After Scout results return, render wave 2:
```bash
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest_file> --execution-wave 2
```

### Step 4: Dispatch Route-Setter Workers (Wave 2)

Spawn Route-Setter workers using the manifest. Include Scout terminal results in the Route-Setter prompts so they can consume Scout findings directly instead of re-running the survey.

For each Route-Setter worker:
```bash
aether spawn-log --parent "Queen" --caste "route_setter" --name "{name}" --task "{task}" --depth 0
```

For each terminal worker result:
1. Write the result to a temp JSON file
2. Render worker-complete ceremony:
```bash
AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>
3. Call `aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`
```

If a planning worker keeps rereading the same file or command, stop waiting and mark that worker `blocked` with a concrete blocker.

### Step 5: Grounding Gate Check

After all workers complete, check the grounding gate (soft warning from `plan-finalize`).

If the plan-finalize result includes a `grounding_warning` or `generic_plan_warning`:
```
⚠ Grounding warning: {warning_message}

The plan may be too generic for this codebase. Consider re-planning with more specific targets.
```

This is a soft gate -- proceed unless the user wants to adjust.

### Step 6: Write Completion and Finalize

Collect terminal worker results into a completion JSON file under `${TMPDIR:-/tmp}/aether-plan-<run>/plan-completion.json`. Never write wrapper result artifacts under `.aether/data/`.

Run the JSON finalizer:
```bash
AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <file>
```

This lets the runtime write canonical planning artifacts and state.

### Step 7: Render Closeout Ceremony

After the JSON finalizer succeeds:
```bash
AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <file>
```

Summarize: selected depth, planning depth, phase count, confidence, `planning_loop.stop_reason`, and which planning agents ran.

Route first to `/ant-build 1` or the runtime-surfaced next build command.
