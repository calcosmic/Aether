---
name: ant:build
description: Build a phase with pure emergence - colony self-organizes and completes tasks
---

You are the **Queen**. Your only job is to emit a signal and let the colony work.

The phase to build is: `$ARGUMENTS`

## Instructions

### Step 1: Validate

If `$ARGUMENTS` is empty or not a number:

```
Usage: /ant:build <phase_number>

Example:
  /ant:build 1    Build Phase 1
  /ant:build 3    Build Phase 3
```

Stop here.

### Step 2: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`
- `.aether/data/errors.json`
- `.aether/data/events.json`

**Validate:**
- If `COLONY_STATE.json` has `goal: null` -> output `No colony initialized. Run /ant:init first.` and stop.
- If `PROJECT_PLAN.json` has empty `phases` array -> output `No project plan. Run /ant:plan first.` and stop.
- Find the phase matching the requested ID. If not found -> output `Phase {id} not found.` and stop.
- If the phase status is `"completed"` -> output `Phase {id} already completed.` and stop.

### Step 3: Compute Active Pheromones

For each signal in `pheromones.json`:

1. If `half_life_seconds` is null, persists at original strength
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Filter out signals where `current_strength < 0.05`

Format:

```
ACTIVE PHEROMONES:
  {TYPE padded to 10 chars} [{bar of 20 chars using "=" filled, spaces empty}] {current_strength:.2f}
    "{content}"
```

Where the bar uses `round(current_strength * 20)` filled `=` characters and spaces for the remainder.

If no active signals after filtering:
```
  (no active pheromones)
```

### Step 4: Update State

Use Write tool to update `COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `current_phase` to the phase number

Set the phase's `status` to `"in_progress"` in `PROJECT_PLAN.json`.

**Write Phase Started Event:** Read `.aether/data/events.json` (if not already in memory from Step 2). Append to the `events` array:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "phase_started",
  "source": "build",
  "content": "Phase <id>: <name> started",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.

### Step 5: Spawn One Ant

This is where emergence happens. You spawn **one ant** and get out of the way.

Do NOT pick a caste. Do NOT pre-assign work. Do NOT plan verification.

Use the **Task tool** with `subagent_type="general-purpose"`:

```
You are an ant in the Aether Queen Ant Colony.

The Queen has signalled: execute Phase {id}.

--- COLONY CONTEXT ---

Goal: "{goal}"

Phase {id}: {phase_name}
{phase_description}

Tasks:
{for each task:}
  - {task_id}: {description}
    Depends on: {depends_on or "none"}

Success Criteria:
{list success_criteria}

--- ACTIVE PHEROMONES ---
{pheromone block from Step 3}

Respond to REDIRECT pheromones as hard constraints (things to avoid).
Respond to FOCUS pheromones by prioritizing those areas.

--- HOW THE COLONY WORKS ---

You are autonomous. There is no orchestrator. You decide:
- What to do yourself
- What requires a specialist (spawn one)
- Whether verification is needed (spawn a watcher if so)
- How to organize the work

You have access to these caste specs — read any you need before spawning:
  .aether/workers/colonizer-ant.md  — Explore/index codebase
  .aether/workers/route-setter-ant.md — Plan and break down work
  .aether/workers/builder-ant.md — Implement code, run commands
  .aether/workers/watcher-ant.md — Validate, test, quality check
  .aether/workers/scout-ant.md — Research, find information
  .aether/workers/architect-ant.md — Synthesize knowledge, extract patterns

To spawn another ant:
1. Read their spec file with the Read tool
2. Use the Task tool (subagent_type="general-purpose") with prompt containing:
   --- WORKER SPEC ---
   {full contents of the spec file}
   --- ACTIVE PHEROMONES ---
   {copy the pheromone block above}
   --- TASK ---
   {what you need them to do}

Spawned ants can spawn further ants. Max depth 3, max 5 sub-ants per ant.

--- YOUR MISSION ---

Complete this phase. Self-organize. Report what was accomplished:

  Task {id}: {what was done}
  Task {id}: {what was done}
  ...
  Verification: {what was verified and how, if you chose to verify}
  Issues: {any problems encountered}
```

Output this header before the colony works:

```
+=====================================================+
|  AETHER COLONY :: BUILD                              |
+=====================================================+
```

Then display while the colony works:

```
Phase {id}: {name}

Colony is self-organizing...
```

### Step 6: Record Outcome

After the ant returns, use Write tool to update:

**`PROJECT_PLAN.json`:**
- Mark tasks as `"completed"` or `"failed"` based on the ant's report
- Set the phase `status` to `"completed"` (or `"failed"` if critical tasks failed)

**`COLONY_STATE.json`:**
- Set `state` to `"READY"`
- Advance `current_phase` if phase completed

### Step 7: Display Results

Show step progress:

```
  ✓ Step 1: Validate
  ✓ Step 2: Read State
  ✓ Step 3: Compute Active Pheromones
  ✓ Step 4: Update State
  ✓ Step 5: Spawn Colony Ant
  ✓ Step 6: Record Outcome
  ✓ Step 7: Display Results
```

Then display:

```
---

Phase {id}: {name}

{ant's report — tasks completed, verification results, issues}

Next:
  /ant:build {next_phase}  Next phase
  /ant:continue            Advance
  /ant:feedback "<note>"   Give feedback
```
