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

### Step 3: Clean Expired Pheromones

Compute current strength for each signal in `pheromones.json`:
1. If `half_life_seconds` is null -> keep (persistent)
2. Otherwise: `current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)`
3. Remove signals where `current_strength < 0.05`

Use the Write tool to write the cleaned `pheromones.json` (keep only non-expired signals).

### Step 4: Update Colony State

Use the Write tool to update `COLONY_STATE.json`:
- Set `current_phase` to the next phase number
- Set `state` to `"READY"`
- Set all workers to `"idle"`

### Step 5: Display Result

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
  ✓ Step 3: Clean Expired Pheromones
  ✓ Step 4: Update Colony State
  ✓ Step 5: Display Result
```

Then output a divider and the result:

```
---

Phase <current> approved. Advancing to Phase <next>.

  Phase <next>: <name>
  <description>

  Tasks: <count>
  State: READY

Next Steps:
  /ant:build <next>      Start building Phase <next>
  /ant:phase <next>      Review phase details first
  /ant:focus "<area>"    Guide colony attention before building
  /ant:redirect "<pat>"  Set constraints before building
```
