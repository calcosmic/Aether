---
name: resume
description: "Resume Previous Session"
symbol: refresh
---

# /ant:resume — Resume Previous Session

Resume work from where you left off after a context clear or new session start.

## When to Use

- After running `/clear` and wanting to continue previous work
- When opening a new Claude Code session with existing colony state
- To check what you were working on and pick up where you left off

## What It Does

1. Reads `.aether/data/session.json` to understand previous activity
2. Reads `COLONY_STATE.json` to verify colony status
3. Checks `TO-DOs.md` for relevant pending work
4. Presents a summary and asks if you want to resume
5. If yes: Restores context and suggests next step
6. If no: Offers to start fresh with `/ant:init`

## Usage

```bash
/ant:resume
```

## Session Recovery Flow

### Step 1: Check for Existing Session

```bash
bash .aether/aether-utils.sh session-read
```

Check if session exists and whether it's stale (> 24 hours old).

### Step 2: If No Session Found

Display:
```
📋 SESSION RESUME

No previous session found.

Would you like to:
1. Start a new colony with /ant:init
2. Check status with /ant:status
3. View existing colonies with /ant:history
```

Offer to run `/ant:init` to start fresh.

### Step 3: If Session Exists

Read session data and colony state, then present summary:

```
📋 SESSION RESUME

You were working on: {colony_goal}
Current Phase: {current_phase} ({current_milestone})
Last Command: {last_command}
Last Active: {hours_ago} hours ago
Suggested Next: {suggested_next}

Active TODOs:
- {todo1}
- {todo2}

Would you like to resume this work?
```

Use `AskUserQuestion` to present options:
- **Yes, resume** — Continue where you left off
- **Show details** — See full colony state first
- **No, start fresh** — Clear session and start new

### Step 4: If User Chooses "Yes, resume"

1. Mark session as resumed:
   ```bash
   bash .aether/aether-utils.sh session-mark-resumed
   ```

2. Present context summary:
   ```
   🔄 Resuming Session

   Colony: {goal}
   Phase: {phase} - {milestone}
   Context restored.

   Next suggested step: {suggested_next}
   ```

3. Run the suggested command or present further options based on state.

### Step 5: If User Chooses "Show details"

Display:
```
📊 Colony Details

Goal: {goal}
State: {state}
Phase: {phase}
Milestone: {milestone}

Recent Activity (from activity.log):
{last 5 entries}

Pending TODOs:
{todos}

[Present Yes/No choice to resume]
```

### Step 6: If User Chooses "No, start fresh"

1. Clear the session:
   ```bash
   bash .aether/aether-utils.sh session-clear
   ```

2. Offer:
   ```
   Session cleared.

   Start fresh with:
   - /ant:init "your new goal"
   - /ant:status (to check other colonies)
   ```

## Integration with Colony Commands

All `/ant:*` commands should update the session after execution:

```bash
# Example: After /ant:build completes
bash .aether/aether-utils.sh session-update "/ant:build $phase" "/ant:continue" "Completed phase $phase"
```

This ensures session.json is always current for resume functionality.

## Session File Location

`.aether/data/session.json`

This file persists across context clears and Claude Code sessions.

## Stale Session Detection

Sessions older than 24 hours are marked "stale":
- User is warned: "This session is X days old"
- Given option to resume anyway or start fresh
- Helps prevent accidentally resuming ancient work

## Implementation

Execute this flow when user runs `/ant:resume`.
