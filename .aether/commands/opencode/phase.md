<!-- Generated from .aether/commands/phase.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:phase
description: "📝🐜📍🐜📝 Show phase details - Queen reviews phase status, tasks, and caste assignment"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen Ant Colony**. Display phase details from the project plan.

## Instructions

The argument is: `$normalized_args`

### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Run /ant:init first.` and stop.

If `plan.phases` is an empty array, output `No project plan. Run /ant:plan first.` and stop.

### Step 2: Determine What to Show

- If `$normalized_args` is empty -> show the current phase (from `current_phase`). If `current_phase` is 0 or beyond the last phase, show phase 1.
- If `$normalized_args` is a number -> show that specific phase
- If `$normalized_args` is "list" or "all" -> show all phases in summary

### Step 3a: Single Phase View

Find the phase by ID in `plan.phases`.

**Calculate next phase ID:**
- Let `total_phases` = length of `plan.phases` array
- If current phase ID < total_phases, then `next_id` = current phase ID + 1
- If current phase ID >= total_phases (viewing last phase), omit the "View next phase" suggestion

Output this header:



```
📝🐜📍🐜📝 ═══════════════════════════════════════════════════
   P H A S E   {id}   D E T A I L S
═══════════════════════════════════════════════════ 📝🐜📍🐜📝


Then display:

```
📍 Phase <id>: <name>
📊 Status: <status>

   <description>

🐜 Tasks:
   {status_icon} <task_id>: <description>
      ↳ depends on: <deps or "none">

✅ Success Criteria:
   • <criterion>

───────────────────────────────────────────────────


🐜 Next Steps:

───────────────────────────────────────────────────
   /ant:build <id>       🔨 Phase <id>: <phase_name>
   /ant:phase <next_id>  📋 Phase <next_id>: <next_phase_name> (only if not last phase)
   /ant:status           📊 Colony status
```



Status icons: `[ ]` pending, `[~]` in_progress, `[✓]` completed

### Step 3b: List View

Output this header:



```
📝🐜📍🐜📝 ═══════════════════════════════════════════════════
   A L L   P H A S E S
═══════════════════════════════════════════════════ 📝🐜📍🐜📝


For each phase in `plan.phases`, display:
```
[{id}] {name} [{status}]
   {short_description}
```

Display phases grouped by status:
```
✓ Completed
   [{id}] {name}
~ In Progress
   [{id}] {name}
[ ] Pending
   [{id}] {name}
```

Display completion progress:
```
📊 Overall: {completed_count}/{total_count} phases complete
```
