---
name: ant:entomb
description: "⚰️🐜⚰️ Entomb completed colony in chambers"
---

You are the **Queen**. Archive the completed colony to chambers.

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
entomb_id="entomb-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$entomb_id"
bash .aether/aether-utils.sh swarm-display-update "Queen" "prime" "excavating" "Entombing colony" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

### Step 1: Read State

Read `.aether/data/COLONY_STATE.json`.

If file missing or `goal: null`:
```
No colony to entomb. Run /ant:init first.
```
Stop here.

### Step 2: Validate Colony Can Be Entombed

Extract: `goal`, `state`, `current_phase`, `plan.phases`, `memory.decisions`, `memory.phase_learnings`.

**Precondition 1: All phases must be completed**

Check if all phases in `plan.phases` have `status: "completed"`:
```
all_completed = all(phase.status == "completed" for phase in plan.phases)
```

If NOT all completed:
```
Cannot entomb incomplete colony.

Completed phases: X of Y
Remaining: {list of incomplete phase names}

Run /ant:continue to complete remaining phases first.
```
Stop here.

**Precondition 2: State must not be EXECUTING**

If `state == "EXECUTING"`:
```
Colony is still executing. Run /ant:continue to reconcile first.
```
Stop here.

**Precondition 3: No critical errors**

Check `errors.records` for any entries with `severity: "critical"`.

If critical errors exist:
```
Cannot entomb colony with critical errors.

Critical errors: {count}
Run /ant:continue to resolve errors first.
```
Stop here.

### Step 3: Compute Milestone

Determine milestone based on phases completed:
- 0 phases: "Fresh Start"
- 1 phase: "First Mound"
- 2-4 phases: "Open Chambers"
- 5+ phases: "Sealed Chambers"

If all phases completed AND user explicitly sealing: "Crowned Anthill"

For entombment, use the computed milestone or extract from state if already set.

### Step 4: User Confirmation

Display:
```
🏺 ═══════════════════════════════════════════════════
   E N T O M B   C O L O N Y
══════════════════════════════════════════════════ 🏺

Goal: {goal}
Phases: {completed}/{total} completed
Milestone: {milestone}

Archive will include:
  - COLONY_STATE.json
  - manifest.json (pheromone trails)

This will reset the active colony. Continue? (yes/no)
```

Wait for explicit "yes" response before proceeding.

If user responds with anything other than "yes", display:
```
Entombment cancelled. Colony remains active.
```
Stop here.

### Step 4.5: Check XML Tools

XML archiving is required for entombment. Check tool availability before proceeding.
Uses `command -v xmllint` directly — consistent with seal.md's tool check.

```bash
if command -v xmllint >/dev/null 2>&1; then
  xmllint_available=true
else
  xmllint_available=false
fi
```

**If xmllint is NOT available:**

Ask the user:
```
xmllint is not installed — XML archiving requires it.

Install now?
  - macOS: xcode-select --install (or brew install libxml2)
  - Linux: apt-get install libxml2-utils

Install xmllint? (yes/no)
```

Use AskUserQuestion with yes/no options.

If yes:
- On macOS: Run `xcode-select --install` or `brew install libxml2`
- On Linux: Run `sudo apt-get install -y libxml2-utils`
- After install attempt, re-check: `command -v xmllint >/dev/null 2>&1`
- If still not available after install:
  ```
  xmllint installation failed. Cannot entomb without XML archiving.
  Install xmllint manually and try again.
  ```
  Stop here.

If no:
```
Entombment requires XML archiving. Install xmllint and try again.
```
Stop here.

### Step 5: Create Chamber

Generate chamber name:
```bash
sanitized_goal=$(echo "{goal}" | tr '[:upper:]' '[:lower:]' | tr -cs '[:alnum:]' '-' | sed 's/^-//;s/-$//' | cut -c1-50)
timestamp=$(date -u +%Y%m%d-%H%M%S)
chamber_name="${sanitized_goal}-${timestamp}"
```

Handle name collision: if directory exists, append counter:
```bash
counter=1
original_name="$chamber_name"
while [[ -d ".aether/chambers/$chamber_name" ]]; do
  chamber_name="${original_name}-${counter}"
  counter=$((counter + 1))
done
```

### Step 6: Create Chamber Using Utilities

Extract decisions and learnings as JSON arrays:
```bash
decisions_json=$(jq -c '.memory.decisions // []' .aether/data/COLONY_STATE.json)
learnings_json=$(jq -c '.memory.phase_learnings // []' .aether/data/COLONY_STATE.json)
phases_completed=$(jq '[.plan.phases[] | select(.status == "completed")] | length' .aether/data/COLONY_STATE.json)
total_phases=$(jq '.plan.phases | length' .aether/data/COLONY_STATE.json)
version=$(jq -r '.version // "3.0"' .aether/data/COLONY_STATE.json)
```

Create the chamber:
```bash
bash .aether/aether-utils.sh chamber-create \
  ".aether/chambers/{chamber_name}" \
  ".aether/data/COLONY_STATE.json" \
  "{goal}" \
  {phases_completed} \
  {total_phases} \
  "{milestone}" \
  "{version}" \
  '{decisions_json}' \
  '{learnings_json}'
```

### Step 6.5: Export XML Archive (hard-stop)

Export combined XML archive to the chamber. This is a HARD REQUIREMENT — entomb fails if XML export fails.

```bash
chamber_dir=".aether/chambers/$chamber_name"
xml_result=$(bash .aether/aether-utils.sh colony-archive-xml "$chamber_dir/colony-archive.xml" 2>&1)
xml_ok=$(echo "$xml_result" | jq -r '.ok // false' 2>/dev/null)

if [[ "$xml_ok" != "true" ]]; then
  # HARD STOP — remove the chamber and abort
  rm -rf "$chamber_dir"
  echo "XML archive export failed. Colony NOT entombed."
  echo ""
  echo "Error: $(echo "$xml_result" | jq -r '.error // "Unknown error"' 2>/dev/null)"
  echo ""
  echo "The chamber has been cleaned up. Fix the XML issue and try again."
  # Do NOT proceed to state reset or any further steps
fi
```

If xml_ok is true, store for display:
```bash
xml_pheromone_count=$(echo "$xml_result" | jq -r '.result.pheromone_count // 0' 2>/dev/null)
xml_archive_line="XML Archive: colony-archive.xml (${xml_pheromone_count} active signals)"
```

**Critical behavior:** If XML export fails, entomb STOPS. The chamber directory is removed (cleanup). The colony state is NOT reset. The user sees a clear error and can retry after fixing the issue.

### Step 7: Verify Chamber Integrity

Run verification:
```bash
bash .aether/aether-utils.sh chamber-verify ".aether/chambers/{chamber_name}"
```

If verification fails, display error and stop:
```
❌ Chamber verification failed.

Error: {verification_error}

The colony has NOT been reset. Please check the chamber directory:
.aether/chambers/{chamber_name}/
```
Stop here.

### Step 8: Reset Colony State

Backup current state:
```bash
cp .aether/data/COLONY_STATE.json .aether/data/COLONY_STATE.json.bak
```

Reset state while preserving memory (pheromones):
```bash
jq '
  .goal = null |
  .state = "IDLE" |
  .current_phase = 0 |
  .plan.phases = [] |
  .plan.generated_at = null |
  .plan.confidence = null |
  .build_started_at = null |
  .session_id = null |
  .initialized_at = null |
  .events = [] |
  .errors.records = [] |
  .errors.flagged_patterns = [] |
  .signals = [] |
  .graveyards = []
' .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
```

Verify reset succeeded:
```bash
new_goal=$(jq -r '.goal' .aether/data/COLONY_STATE.json)
if [[ "$new_goal" != "null" ]]; then
  # Restore from backup
  mv .aether/data/COLONY_STATE.json.bak .aether/data/COLONY_STATE.json
  echo "Error: State reset failed. Restored from backup."
  exit 1
fi
```

Remove backup after successful reset:
```bash
rm -f .aether/data/COLONY_STATE.json.bak
```

### Step 8.5: Write Final Handoff

After entombing the colony, write the final handoff documenting the archived colony:

```bash
cat > .aether/HANDOFF.md << 'HANDOFF_EOF'
# Colony Session — ENTOMBED

## ⚰️ Colony Archived
**Status:** Entombed in Chambers — Colony work preserved

## Chamber Location
.aether/chambers/{chamber_name}/

## Colony Summary
- Goal: "{goal}"
- Phases: {completed} completed of {total}
- Milestone: {milestone}
- Entombed At: {timestamp}

## Chamber Contents
- colony-state.json — Full colony state
- manifest.json — Archive metadata
- activity.log — Colony activity history
- spawn-tree.txt — Worker spawn records
- flags.json — Project flags (if existed)

## Session Note
This colony has been entombed and the active state reset.
The colony rests. Its learnings are preserved in the chamber.

To start anew: /ant:lay-eggs "<new goal>"
To explore chambers: /ant:tunnels
HANDOFF_EOF
```

This handoff serves as the record of the entombed colony.

### Step 9: Display Result

```
🏺 ═══════════════════════════════════════════════════
   C O L O N Y   E N T O M B E D
══════════════════════════════════════════════════ 🏺

✅ Colony archived successfully

👑 Goal: {goal}
📍 Phases: {completed} completed
🏆 Milestone: {milestone}

📦 Chamber: .aether/chambers/{chamber_name}/
{xml_archive_line}

🐜 The colony rests. Its learnings are preserved.

💾 State persisted — safe to /clear

🐜 What would you like to do next?
   1. /ant:lay-eggs "<new goal>"  — Start a new colony
   2. /ant:tunnels                — Browse archived colonies
   3. /clear                      — Clear context and continue

Use AskUserQuestion with these three options.

If option 1 selected: proceed to run /ant:lay-eggs flow
If option 2 selected: run /ant:tunnels
If option 3 selected: display "Run /ant:lay-eggs to begin anew after clearing"
```

### Edge Cases

**Chamber name collision:** Automatically append counter to make unique.

**Missing files during archive:** Note in output but continue with available files.

**State reset failure:** Restore from backup, display error, do not claim success.

**Empty phases array:** Can entomb a colony that was initialized but had no phases planned (treat as 0 of 0 completed).
