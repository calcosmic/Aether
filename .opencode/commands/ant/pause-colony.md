---
name: ant:pause-colony
description: "💾🐜⏸️🐜💾 Pause colony work and create handoff document for resuming later"
---

You are the **Queen Ant Colony**. Save current state for session handoff.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

Parse `$normalized_args`:
- If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
- Otherwise: set `visual_mode = true`

### Step 0: Initialize Visual Mode (if enabled)

If `visual_mode` is true:
```bash
# Generate session ID
pause_id="pause-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$pause_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Pausing colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

If `goal` is null, output `No colony initialized. Nothing to pause.` and stop.

### Step 2: Compute Active Signals

Run using the Bash tool:
```bash
bash .aether/aether-utils.sh pheromone-read
```

Use `.result.signals` as the active signal list (already decay-filtered by runtime logic).
If empty, treat as "no active pheromones."

### Step 3: Build Handoff Summary

Gather context for the handoff from `COLONY_STATE.json`:
- `goal` from top level
- `state` and `current_phase` from top level
- `workers` object
- Active signals from `pheromone-read` output (with current decayed strengths from Step 2)
- Phase progress from `plan.phases` (how many complete, current phase tasks)
- What was in progress or pending

### Step 4: Write Handoff

Use the Write tool to update `.aether/HANDOFF.md` with a session handoff section at the top. The format:

```markdown
# Colony Session Paused

## Quick Resume
Run `/ant:resume-colony` in a new session.

## State at Pause
- Goal: "<goal>"
- State: <state>
- Current Phase: <phase_number> — <phase_name>
- Session: <session_id>
- Paused: <ISO-8601 timestamp>

## Active Pheromones
- <TYPE> (strength <current>): "<content>"
(list each non-expired signal)

## Phase Progress
(for each phase, show status)
- Phase <id>: <name> [<status>]

## Current Phase Tasks
(list tasks in the current phase with their statuses)
- [<icon>] <task_id>: <description>

## What Was Happening
<brief description of what the colony was doing>

## Next Steps on Resume
<what should happen next>
```

### Step 4.5: Set Paused Flag in State

Use Read tool to get current COLONY_STATE.json.

Use Write tool to update COLONY_STATE.json with paused flag:
- Add field: `"paused": true`
- Add field: `"paused_at": "<ISO-8601 timestamp>"`
- Update last_updated timestamp

This flag indicates the colony is in a paused state and will be cleared on resume.

### Step 4.6: Commit Suggestion (Optional)

**This step is non-blocking. Skipping does not affect the pause or any subsequent steps. Failure to commit has zero consequences.**

Before displaying the pause confirmation, check if the user has uncommitted work worth preserving.

1. **Check for uncommitted changes:**
```bash
git status --porcelain 2>/dev/null
```
If the output is empty (nothing to commit) or the command fails (not a git repo), skip this step silently and continue to Step 5.

2. **Check for double-prompting:**
Read `last_commit_suggestion_phase` from COLONY_STATE.json (already loaded in Step 1).
If `last_commit_suggestion_phase` equals the current phase, skip this step silently — the user was already prompted at POST-ADVANCE. Continue to Step 5.

3. **Generate the commit message:**
```bash
bash .aether/aether-utils.sh generate-commit-message "pause" {current_phase} "{phase_name}"
```
Parse the returned JSON to extract `message` and `files_changed`.

4. **Check files changed:**
```bash
git diff --stat HEAD 2>/dev/null | tail -5
```

5. **Display the suggestion:**
```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  Message:  {generated_message}
  Files:    {files_changed} files changed
  Preview:  {first 5 lines of git diff --stat}

──────────────────────────────────────────────────
```

6. **Use AskUserQuestion:**
```
Commit your work before pausing?

1. Yes, commit with this message
2. Yes, but let me write the message
3. No, I'll commit later
```

7. **If option 1 ("Yes, commit with this message"):**
```bash
git add -A && git commit -m "{generated_message}"
```
Display: `Committed: {generated_message} ({files_changed} files)`

8. **If option 2 ("Yes, but let me write the message"):**
Use AskUserQuestion to get the user's custom commit message, then:
```bash
git add -A && git commit -m "{custom_message}"
```
Display: `Committed: {custom_message} ({files_changed} files)`

9. **If option 3 ("No, I'll commit later"):**
Display: `Skipped. Your changes are saved on disk but not committed.`

10. **Record the suggestion to prevent double-prompting:**
Set `last_commit_suggestion_phase` to `{current_phase}` in COLONY_STATE.json (add the field at the top level if it does not exist).

**Error handling:** If any git command fails (not a repo, merge conflict, pre-commit hook rejection), display the error output and continue to Step 5. The commit suggestion is advisory only — it never blocks the pause flow.

Continue to Step 5.

### Step 5: Display Confirmation

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony paused" "Colony" '{"read":3,"grep":0,"edit":2,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-text "$pause_id"
```

Output header:

```
💾🐜⏸️🐜💾 ═══════════════════════════════════════════════════
   C O L O N Y   P A U S E D
═══════════════════════════════════════════════════ 💾🐜⏸️🐜💾
```

Then output:
+=====================================================+
|  AETHER COLONY :: PAUSED                             |
+=====================================================+

  Goal: "<goal>"
  Phase: <current_phase> — <phase_name>
  Pheromones: <active_count> active

  Handoff saved to .aether/HANDOFF.md
  Paused state saved to COLONY_STATE.json

To resume in a new session:
  /ant:resume-colony

💾 State persisted — safe to /clear

🐜 What would you like to do next?
   1. /ant:resume-colony              — Resume work in this session
   2. /ant:lay-eggs "<new goal>"      — Start a new colony
   3. /clear                          — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:resume-colony flow
If option 2 selected: run /ant:lay-eggs flow
If option 3 selected: display "Run /ant:resume-colony when ready to continue, or /ant:lay-eggs to start fresh"
```
