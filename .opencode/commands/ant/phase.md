---
name: ant:phase
description: "Show phase details - Queen reviews phase status, tasks, and caste assignment"
---

You are the **Queen Ant Colony**. Display phase details from the project plan.

## Instructions

The argument is: `$ARGUMENTS`

### Step 1: Read State

Use the read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Run /ant:init first.` and stop.

If `plan.phases` is an empty array, output `No project plan. Run /ant:plan first.` and stop.

### Step 2: Determine What to Show

- If `$ARGUMENTS` is empty -> show the current phase (from `current_phase`). If `current_phase` is 0 or beyond the last phase, show phase 1.
- If `$ARGUMENTS` is a number -> show that specific phase
- If `$ARGUMENTS` is "list" or "all" -> show all phases in summary

### Step 3a: Single Phase View

Find the phase by ID in `plan.phases`.

Output this header:

```
═══════════════════════════════════════════════════
   P H A S E   {id}   D E T A I L S
═══════════════════════════════════════════════════
```

Then display:

```
Phase <id>: <name>
Status: <status>

   <description>

Tasks:
   {status_icon} <task_id>: <description>
      -> depends on: <deps or "none">

Success Criteria:
   * <criterion>

─────────────────────────────────────────────────────
Next Steps:
   /ant:build <id>       Build this phase
   /ant:phase <next_id>  View next phase
   /ant:status           Colony status
```

Status icons: `[ ]` pending, `[~]` in_progress, `[x]` completed

### Step 3b: List View

Output this header:

```
═══════════════════════════════════════════════════
   A L L   P H A S E S
═══════════════════════════════════════════════════
```

Then display all phases as a summary:

```
Goal: "<goal>"

   {icon} Phase <id>: <name>
         <completed>/<total> tasks | <status>

(repeat for each phase)

─────────────────────────────────────────────────────
Legend: [x] completed  [~] in progress  [ ] pending

/ant:phase <id> for details
```
