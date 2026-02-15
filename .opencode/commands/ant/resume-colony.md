---
name: ant:resume-colony
description: "🚦➡️🐜💨💨 Resume colony from saved session - restores all state"
---

You are the **Queen Ant Colony**. Restore state from a paused session.

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
resume_id="resume-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$resume_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Resuming colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 0.5: Version Check (Non-blocking)

Run using the Bash tool: `bash .aether/aether-utils.sh version-check 2>/dev/null || true`

If the command succeeds and the JSON result contains a non-empty string, display it as a one-line notice. Proceed regardless of outcome.

### Step 1: Load State and Validate

Run using Bash tool: `bash .aether/aether-utils.sh load-state`

If successful:
1. Parse state from result
2. If goal is null: Show "No colony state found..." message and stop
3. Check if paused flag is true - if not, note "Colony was not paused, but resuming anyway"
4. Extract all state fields for display

Keep state loaded (don't unload yet) - we'll need it for the full display.

### Step 2: Compute Active Signals

Read active signals from COLONY_STATE.json `signals` array (already loaded in Step 1).

Filter signals where:
- `expires_at` is null (permanent signals like INIT), OR
- `expires_at` > current timestamp (not expired)

If `signals` array is empty or all expired, treat as "no active pheromones."

### Step 3: Display Restored State

**Note:** Other ant commands (`/ant:status`, `/ant:build`, `/ant:plan`, `/ant:continue`) also show brief resumption context automatically. This full resume provides complete state restoration for explicit session recovery.

Output header:

```
🚦➡️🐜💨💨 ═══════════════════════════════════════════════════
   C O L O N Y   R E S U M E D
═══════════════════════════════════════════════════ 🚦➡️🐜💨💨
```

Read the .aether/HANDOFF.md for context about what was happening, then display:

```
+=====================================================+
|  AETHER COLONY :: RESUMED                            |
+=====================================================+

  Goal: "<goal>"
  State: <state>
  Session: <session_id>
  Phase: <current_phase>

ACTIVE PHEROMONES
  {TYPE padded to 10 chars} [{bar of 20 chars using filled/empty}] {current_strength:.2f}
    "{content}"

  Where the bar uses round(current_strength * 20) filled characters and spaces for the remainder.

  If no active signals: (no active pheromones)

PHASE PROGRESS
  Phase <id>: <name> [<status>]
  (list all phases from plan.phases)

CONTEXT FROM HANDOFF
  <summarize what was happening from .aether/HANDOFF.md>

NEXT ACTIONS
```

**If visual_mode is true, render final swarm display:**
```bash
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "completed" "Colony resumed" "Colony" '{"read":3,"grep":0,"edit":2,"bash":1}' 100 "fungus_garden" 100
bash .aether/aether-utils.sh swarm-display-render "$resume_id"
```

Route to next action based on state:
- If state is `READY` and there's a pending phase -> suggest `/ant:build <phase>`
- If state is `EXECUTING` -> note that a build was interrupted, suggest restarting with `/ant:build <phase>`
- If state is `PLANNING` -> note that planning was interrupted, suggest `/ant:plan`
- Otherwise -> suggest `/ant:status` for full overview

### Step 6: Clear Paused State and Cleanup

Use Write tool to update COLONY_STATE.json:
- Remove or set to false: `"paused": false`
- Remove: `"paused_at"` field
- Update last_updated timestamp
- Add event: `{timestamp, type: "colony_resumed", worker: "resume", details: "Session resumed"}`

Use Bash tool to remove HANDOFF.md: `rm -f .aether/HANDOFF.md`

Run: `bash .aether/aether-utils.sh unload-state` to release lock.

---

## Auto-Recovery Pattern Reference

The colony uses a tiered auto-recovery pattern to maintain context across session boundaries:

### Format Tiers

| Context | Format | When Used |
|---------|--------|-----------|
| Brief | `🔄 Resuming: Phase X - Name` | Action commands (build, plan, continue) |
| Extended | Brief + last activity timestamp | Status command |
| Full | Complete state with pheromones, workers, context | resume-colony command |

### Brief Format (Action Commands)

Used by `/ant:build`, `/ant:plan`, `/ant:continue`:

```
🔄 Resuming: Phase <current_phase> - <phase_name>
```

Provides minimal orientation before executing the command's primary function.

### Extended Format (Status Command)

Used by `/ant:status` Step 1.5:

```
🔄 Resuming: Phase <current_phase> - <phase_name>
   Last activity: <last_event_timestamp>
```

Adds temporal context to help gauge session staleness.

### Full Format (Resume-Colony)

Used by `/ant:resume-colony`:

- Complete header with ASCII art
- Goal, state, session ID, phase
- Active pheromones with strength bars
- Worker status by caste
- Phase progress for all phases
- Handoff context summary
- Next action routing

### Implementation Notes

1. **State Source:** All formats read from `.aether/data/COLONY_STATE.json`
2. **Phase Name:** Extracted from `plan.phases[current_phase - 1].name`
3. **Last Activity:** Parsed from the last entry in `events` array
4. **Edge Cases:** Handle missing phase names, empty events, phase 0
