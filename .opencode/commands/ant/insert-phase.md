---
name: ant:insert-phase
description: "➕🐜 Insert a corrective phase into the active plan"
---

You are the **Queen**. Insert a new corrective phase without forcing the user to provide flags.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

Use `$normalized_args` throughout this command.

### Step 1: Validate Colony State

Read `.aether/data/COLONY_STATE.json`.

If `goal` is null:
`No colony initialized. Run /ant:init first.`
Stop.

If `plan.phases` is empty:
`No project plan. Run /ant:plan first.`
Stop.

Determine:
- `current_phase`
- `total_phases`
- `current_phase_name` (if available)

### Step 2: Collect Minimal Input

This command is designed to work with no arguments.

If `$normalized_args` is non-empty:
- Use `$normalized_args` as the issue summary.
- Skip the first question.

Otherwise ask:
```text
What is not working and needs a corrective phase?
```

Then ask:
```text
What should the inserted phase accomplish?
```

Then ask (optional):
```text
Any hard constraints to enforce while fixing this? (or say "none")
```

If constraints answer is `none` (case-insensitive), treat constraints as empty.

### Step 3: Derive Phase Name

Create a concise phase name from the user goal:
- Start with `Stabilize`
- Add a short noun phrase from the goal (max 6 words)
- Keep under 80 characters

Example:
- Goal: `Fix retry flow and stop duplicate requests`
- Name: `Stabilize retry flow and duplicate requests`

### Step 4: Insert Phase Safely

Run:

```bash
bash .aether/aether-utils.sh phase-insert "<phase_name>" "<goal_text>" "<constraints_text>"
```

Parse JSON result:
- If `ok != true`, display error and stop.
- Extract `inserted_phase_id` and `after_phase`.

### Step 5: Confirm Outcome

Display:

```text
Inserted corrective phase successfully.
  New phase: <inserted_phase_id> — <phase_name>
  Inserted after phase: <after_phase>
  Goal: <goal_text>
```

If constraints were provided, also display:
```text
  Constraints captured: <constraints_text>
```

### Step 6: Next Up

Show:

```text
Next steps:
  /ant:phase <inserted_phase_id>   Review inserted phase details
  /ant:build <inserted_phase_id>   Execute corrective phase
  /ant:status                      Verify updated plan state
```
