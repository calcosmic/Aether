---
name: ant:continue
description: Queen approves phase completion and clears check-in for colony to proceed
---

You are the **Queen Ant Colony**. Advance to the next phase.

## Instructions

### Step 1: Read State

Use the Read tool to read these files (in parallel):
- `.aether/data/COLONY_STATE.json`
- `.aether/data/pheromones.json`
- `.aether/data/PROJECT_PLAN.json`
- `.aether/data/errors.json`
- `.aether/data/memory.json`
- `.aether/data/events.json`

If `COLONY_STATE.json` has `goal: null`, output `No colony initialized. Run /ant:init first.` and stop.

If `PROJECT_PLAN.json` has empty `phases`, output `No project plan. Run /ant:plan first.` and stop.

### Step 2: Determine Next Phase

Look at `current_phase` in `COLONY_STATE.json`. The next phase is `current_phase + 1`.

If there is no next phase (current is the last phase), output:

```
All phases complete. Colony has finished the project plan.

  /ant:status   View final colony status
  /ant:plan     Generate a new plan (will replace current)
```

Stop here.

### Step 3: Phase Completion Summary

Before advancing, display a summary of the completed phase using data from the state files read in Step 1.

Output:

```
---------------------------------------------------
PHASE <N> REVIEW: <phase_name>
---------------------------------------------------

  Tasks:
    [x] <task_id>: <description>
    [x] <task_id>: <description>
    ...
    Completed: <N>/<total>

  Errors:
    <count> errors encountered
    (list severity counts: N critical, N high, N medium, N low)

  Decisions:
    <count> decisions logged during this phase
    (list last 3 decisions from memory.json decisions array: "<content>")

---------------------------------------------------
```

Get task data from `PROJECT_PLAN.json` -- look at the current phase's `tasks` array. Show `[x]` for completed, `[ ]` for incomplete.

Get error data from `errors.json` -- filter the `errors` array by `phase` field matching the current phase number. Count by severity level.

Get decision data from `memory.json` -- count the `decisions` array entries. Show last 3 decisions.

If no errors were encountered during this phase:
```
  Errors: None
```

If no decisions were logged:
```
  Decisions: None
```

This step is DISPLAY ONLY -- it reads state but does not write anything. The purpose is to give the user a retrospective before the phase advances.

### Step 4: Extract Phase Learnings

Review the completed phase by analyzing:
- Tasks completed in this phase (from PROJECT_PLAN.json -- look at the current phase's tasks)
- Errors encountered during this phase (from errors.json -- filter by `phase` field matching current phase)
- Events that occurred (from events.json -- recent events related to this phase)
- Flagged patterns (from errors.json `flagged_patterns` array)

Read `.aether/data/memory.json`. Append a phase learning entry to the `phase_learnings` array:

```json
{
  "id": "learn_<unix_timestamp>_<4_random_hex>",
  "phase": <current_phase_number>,
  "phase_name": "<phase name from PROJECT_PLAN.json>",
  "learnings": [
    "<specific thing learned -- what worked, what didn't, what to remember>",
    "<another specific learning>"
  ],
  "errors_encountered": <count of errors with this phase number>,
  "timestamp": "<ISO-8601 UTC>"
}
```

Learnings must be SPECIFIC and ACTIONABLE. Good: "TypeScript strict mode caught 12 type errors early." Bad: "Phase completed successfully." Draw from actual task outcomes, errors, and events -- not boilerplate.

If the `phase_learnings` array exceeds 20 entries, remove the oldest entries to keep only 20.

Use the Write tool to write the updated memory.json.

**Update Spawn Outcomes:** Read `.aether/data/COLONY_STATE.json`. Review the events in `events.json` for events of type `phase_completed` or `phase_failed` related to the current phase. If the phase completed successfully, look at events.json for spawn-related events or the build report to identify which castes contributed. For each identified caste, increment `alpha` and `successes` in `spawn_outcomes`. If the phase failed, increment `beta` and `failures` for identified castes. Increment `total_spawns` regardless. Write the updated COLONY_STATE.json.

If no castes can be identified from events, skip this step.

### Step 4.5: Auto-Emit Pheromones

After extracting learnings, automatically emit pheromones based on phase outcomes.

Read `.aether/data/pheromones.json` (if not already in memory from Step 1).

**Always emit a FEEDBACK pheromone** summarizing what worked and what didn't from the phase learnings:

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "FEEDBACK",
  "content": "<summary of what worked and what didn't from the phase learnings — be specific, reference actual task outcomes>",
  "strength": 0.5,
  "half_life_seconds": 21600,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:continue",
  "auto": true
}
```

**Conditionally emit a REDIRECT pheromone** if `errors.json` has any `flagged_patterns` entries related to this phase (check if any flagged pattern's errors occurred during this phase):

```json
{
  "id": "auto_<unix_timestamp>_<4_random_hex>",
  "type": "REDIRECT",
  "content": "Avoid repeating: <description of the flagged pattern and its root causes>",
  "strength": 0.9,
  "half_life_seconds": 86400,
  "created_at": "<ISO-8601 UTC>",
  "source": "auto:continue",
  "auto": true
}
```

Append these to the `signals` array in `pheromones.json`. Use the Write tool to write the updated file.

**Log Events:** For each auto-emitted pheromone, append to events.json:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "pheromone_auto_emitted",
  "source": "continue",
  "content": "<TYPE> pheromone auto-emitted: <first 80 chars of content>",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100. Write the updated events.json.

### Step 5: Clean Expired Pheromones

Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-cleanup
```

This removes signals with `current_strength` below 0.05 from `pheromones.json` and returns `{"ok":true,"result":{"removed":N,"remaining":N}}`. The cleanup result (removed count) can be mentioned in the display output.

### Step 6: Write Events

Read `.aether/data/events.json`. Append two events to the `events` array:

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "learnings_extracted",
  "source": "continue",
  "content": "Extracted <N> learnings from Phase <id>: <name>",
  "timestamp": "<ISO-8601 UTC>"
}
```

```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "phase_advanced",
  "source": "continue",
  "content": "Advanced from Phase <current> to Phase <next>",
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated events.json.

### Step 7: Update Colony State

Use the Write tool to update `COLONY_STATE.json`:
- Set `current_phase` to the next phase number
- Set `state` to `"READY"`
- Set all workers to `"idle"`

### Step 8: Display Result

Output this header at the start of your response:

```
+=====================================================+
|  AETHER COLONY :: CONTINUE                           |
+=====================================================+
```

Then show step progress:

```
  ✓ Step 1: Read State
  ✓ Step 2: Determine Next Phase
  ✓ Step 3: Phase Completion Summary
  ✓ Step 4: Extract Phase Learnings
  ✓ Step 4.5: Auto-Emit Pheromones
  ✓ Step 5: Clean Expired Pheromones
  ✓ Step 6: Write Events
  ✓ Step 7: Update Colony State
  ✓ Step 8: Display Result
```

Then output a divider and the result:

```
---

Phase <current> approved. Advancing to Phase <next>.

  Phase <next>: <name>
  <description>

  Tasks: <count>
  State: READY

  Learnings Extracted:
    - <learning 1>
    - <learning 2>

  Auto-Emitted Pheromones:
    FEEDBACK (strength 0.5, half-life 6h): "<first 80 chars of content>"
    {if REDIRECT was emitted:}
    REDIRECT (strength 0.9, half-life 24h): "<first 80 chars of content>"

Next Steps:
  /ant:build <next>      Start building Phase <next>
  /ant:phase <next>      Review phase details first
  /ant:focus "<area>"    Guide colony attention before building
  /ant:redirect "<pat>"  Set constraints before building
```
