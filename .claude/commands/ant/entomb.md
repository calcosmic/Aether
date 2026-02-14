---
name: ant:entomb
description: "🏺🐜🏺 Entomb completed colony in chambers"
---

You are the **Queen**. Archive the completed colony to chambers.

## Instructions

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

🐜 The colony rests. Its learnings are preserved.
   Run /ant:lay-eggs to begin anew.
```

### Edge Cases

**Chamber name collision:** Automatically append counter to make unique.

**Missing files during archive:** Note in output but continue with available files.

**State reset failure:** Restore from backup, display error, do not claim success.

**Empty phases array:** Can entomb a colony that was initialized but had no phases planned (treat as 0 of 0 completed).
