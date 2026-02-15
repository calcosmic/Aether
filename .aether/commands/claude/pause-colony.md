---
name: ant:pause-colony
description: "💾🐜⏸️🐜💾 Pause colony work and create handoff document for resuming later"
---

You are the **Queen Ant Colony**. Save current state for session handoff.

## Instructions

Parse `$ARGUMENTS`:
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

Read active signals from COLONY_STATE.json `signals` array (already loaded in Step 1).

Filter signals where:
- `expires_at` is null (permanent signals like INIT), OR
- `expires_at` > current timestamp (not expired)

If `signals` array is empty or all expired, treat as "no active pheromones."

### Step 3: Build Handoff Summary

Gather context for the handoff from `COLONY_STATE.json`:
- `goal` from top level
- `state` and `current_phase` from top level
- `workers` object
- Active signals from `signals` array (with current decayed strengths from Step 2)
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

**1. Check for uncommitted changes:**
```bash
git status --porcelain 2>/dev/null
```
If the output is empty (nothing to commit) or the command fails (not a git repo), skip this step silently and continue to Step 5.

**2. Check for double-prompting:**
Read `last_commit_suggestion_phase` from COLONY_STATE.json (already loaded in Step 1).
If `last_commit_suggestion_phase` equals the current phase, skip this step silently — the user was already prompted at POST-ADVANCE. Continue to Step 5.

**3. Capture AI Description:**

**As the AI, briefly describe what was in progress when pausing.**

Examples:
- "Mid-implementation of task-based routing, tests passing"
- "Completed model selection logic, integration tests pending"
- "Fixed file locking, ready for verification"

Store this as `ai_description`. If no clear description emerges, leave empty (will use fallback).

**4. Generate Enhanced Commit Message:**
```bash
bash .aether/aether-utils.sh generate-commit-message "contextual" {current_phase} "{phase_name}" "{ai_description}" {plan_number}
```

Parse the returned JSON to extract `message`, `body`, `files_changed`, `subsystem`, and `scope`.

**5. Display the enhanced suggestion:**
```
──────────────────────────────────────────────────
Commit Suggestion
──────────────────────────────────────────────────

  AI Description: {ai_description}

  Formatted Message:
  {message}

  Metadata:
  Scope: {scope}
  Files: {files_changed} files changed

──────────────────────────────────────────────────
```

**6. Use AskUserQuestion:**
```
Commit your work before pausing?

1. Yes, commit with this message
2. Yes, but let me edit the description
3. No, I'll commit later
```

**7. If option 1 ("Yes, commit with this message"):**
```bash
git add -A && git commit -m "{message}" -m "{body}"
```
Display: `Committed: {message} ({files_changed} files)`

**8. If option 2 ("Yes, but let me edit"):**
Prompt for custom description, then regenerate and commit.

**9. If option 3 ("No, I'll commit later"):**
Display: `Skipped. Your changes are saved on disk but not committed.`

**10. Record the suggestion:**
Set `last_commit_suggestion_phase` to `{current_phase}` in COLONY_STATE.json.

**Error handling:** If any git command fails, display the error and continue to Step 5.

Continue to Step 5.

### Step 4.8: Update Context Document

Log this pause activity to `.aether/CONTEXT.md`:

```bash
bash .aether/aether-utils.sh context-update activity "pause-colony" "Colony paused — handoff created" "—"
```

Update safe-to-clear status:
```bash
bash .aether/aether-utils.sh context-update safe-to-clear "YES" "Colony paused — safe to /clear, run /ant:resume-colony to continue"
```

### Step 5: Display Confirmation

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony paused" "Colony" '{"read":3,"grep":0,"edit":2,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$pause_id"
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

📋 Context document updated at `.aether/CONTEXT.md`

🐜 What would you like to do next?
   1. /ant:resume-colony              — Resume work in this session
   2. /ant:lay-eggs "<new goal>"      — Start a new colony
   3. /clear                          — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:resume-colony flow
If option 2 selected: run /ant:lay-eggs flow
If option 3 selected: display "Run /ant:resume-colony when ready to continue, or /ant:lay-eggs to start fresh"
```
